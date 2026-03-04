package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
)

type googleAccountSyncer struct {
	db         *pgxpool.Pool
	googleAuth portout.GoogleAuthClient
}

func NewGoogleAccountSyncer(db *pgxpool.Pool, googleAuth portout.GoogleAuthClient) *googleAccountSyncer {
	return &googleAccountSyncer{
		db:         db,
		googleAuth: googleAuth,
	}
}

func (s *googleAccountSyncer) SyncAccount(
	ctx context.Context,
	userID string,
	account *authdomain.GoogleAccount,
	options portout.GoogleSyncOptions,
) error {
	if account == nil {
		return fmt.Errorf("google account is required")
	}

	token := portout.OAuthToken{
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
	}
	if account.TokenExpiresAt != nil {
		token.Expiry = *account.TokenExpiresAt
	}

	now := time.Now()

	if options.SyncCalendar {
		if err := s.syncCalendarsAndEvents(ctx, userID, account.ID, token, now); err != nil {
			return err
		}
	}

	if options.SyncTodo {
		if err := s.syncTaskListsAndTodos(ctx, userID, account.ID, token, now); err != nil {
			return err
		}
	}

	return nil
}

func (s *googleAccountSyncer) syncCalendarsAndEvents(
	ctx context.Context,
	userID, accountID string,
	token portout.OAuthToken,
	now time.Time,
) error {
	calendars, err := s.googleAuth.ListCalendars(ctx, token)
	if err != nil {
		return fmt.Errorf("list google calendars: %w", err)
	}

	localCalendarIDByGoogleID := make(map[string]string, len(calendars))
	for _, cal := range calendars {
		_, err := s.db.Exec(ctx,
			`UPDATE calendars
			 SET google_account_id = $4, name = $3, color_id = $5, is_primary = $6, is_visible = $7, updated_at = $8
			 WHERE user_id = $1 AND google_id = $2`,
			userID, cal.GoogleID, cal.Name, accountID, cal.ColorID, cal.IsPrimary, cal.IsVisible, now,
		)
		if err != nil {
			return fmt.Errorf("update calendar: %w", err)
		}

		var localCalendarID string
		queryErr := s.db.QueryRow(
			ctx,
			`SELECT id FROM calendars WHERE user_id = $1 AND google_id = $2`,
			userID,
			cal.GoogleID,
		).Scan(&localCalendarID)
		if queryErr != nil {
			if queryErr != pgx.ErrNoRows {
				return fmt.Errorf("query calendar id: %w", queryErr)
			}
			localCalendarID = uuid.New().String()
			_, err = s.db.Exec(ctx,
				`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, color_id, is_primary, is_visible, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				localCalendarID, userID, cal.GoogleID, accountID, cal.Name, cal.ColorID, cal.IsPrimary, cal.IsVisible, now, now,
			)
			if err != nil {
				return fmt.Errorf("insert calendar: %w", err)
			}
		}
		localCalendarIDByGoogleID[cal.GoogleID] = localCalendarID
	}

	start := now.AddDate(0, 0, -90)
	end := now.AddDate(0, 0, 365)
	for googleCalendarID, localCalendarID := range localCalendarIDByGoogleID {
		var syncToken *string
		_ = s.db.QueryRow(
			ctx,
			`SELECT sync_token FROM calendars WHERE id = $1 AND user_id = $2`,
			localCalendarID,
			userID,
		).Scan(&syncToken)

		currentSyncToken := ""
		if syncToken != nil {
			currentSyncToken = *syncToken
		}

		pageToken := ""
		nextSyncToken := ""

		for {
			page, err := s.googleAuth.ListCalendarEvents(
				ctx,
				token,
				googleCalendarID,
				pageToken,
				currentSyncToken,
				&start,
				&end,
			)
			if err != nil {
				// sync token 만료 시 1회 초기 동기화로 재시도
				if currentSyncToken != "" {
					currentSyncToken = ""
					pageToken = ""
					nextSyncToken = ""
					continue
				}
				return fmt.Errorf("list google events: %w", err)
			}

			for _, event := range page.Events {
				if event.Status == "cancelled" || event.StartTime == nil || event.EndTime == nil {
					_, _ = s.db.Exec(ctx,
						`UPDATE events
						 SET deleted_at = $4, updated_at = $4
						 WHERE user_id = $1 AND calendar_id = $2 AND google_id = $3`,
						userID,
						localCalendarID,
						event.GoogleID,
						now,
					)
					continue
				}

				tag, err := s.db.Exec(ctx,
					`UPDATE events
					 SET title = $4, description = $5, location = $6, start_time = $7, end_time = $8,
					     timezone = $9, is_all_day = $10, color_id = $11, recurrence_rule = $12, etag = $13,
					     deleted_at = NULL, updated_at = $14
					 WHERE user_id = $1 AND calendar_id = $2 AND google_id = $3`,
					userID, localCalendarID, event.GoogleID, event.Title, event.Description, event.Location,
					*event.StartTime, *event.EndTime, event.Timezone, event.IsAllDay,
					event.ColorID, event.RecurrenceRule, event.ETag, now,
				)
				if err != nil {
					return fmt.Errorf("update event: %w", err)
				}
				if tag.RowsAffected() > 0 {
					continue
				}

				_, err = s.db.Exec(ctx,
					`INSERT INTO events (
					   id, calendar_id, user_id, google_id, title, description, location,
					   start_time, end_time, timezone, is_all_day, color_id, recurrence_rule, etag, created_at, updated_at
					 ) VALUES (
					   $1, $2, $3, $4, $5, $6, $7,
					   $8, $9, $10, $11, $12, $13, $14, $15, $16
					 )`,
					uuid.New().String(), localCalendarID, userID, event.GoogleID, event.Title, event.Description, event.Location,
					*event.StartTime, *event.EndTime, event.Timezone, event.IsAllDay, event.ColorID, event.RecurrenceRule, event.ETag, now, now,
				)
				if err != nil {
					return fmt.Errorf("insert event: %w", err)
				}
			}

			if page.NextSyncToken != "" {
				nextSyncToken = page.NextSyncToken
			}
			if page.NextPageToken == "" {
				break
			}
			pageToken = page.NextPageToken
		}

		if nextSyncToken != "" {
			_, _ = s.db.Exec(ctx,
				`UPDATE calendars SET sync_token = $3, updated_at = $4 WHERE id = $1 AND user_id = $2`,
				localCalendarID,
				userID,
				nextSyncToken,
				now,
			)
		}
	}

	return nil
}

func (s *googleAccountSyncer) syncTaskListsAndTodos(
	ctx context.Context,
	userID, accountID string,
	token portout.OAuthToken,
	now time.Time,
) error {
	taskLists, err := s.googleAuth.ListTaskLists(ctx, token)
	if err != nil {
		return fmt.Errorf("list google task lists: %w", err)
	}

	localListIDByGoogleID := make(map[string]string, len(taskLists))
	for idx, taskList := range taskLists {
		_, err := s.db.Exec(ctx,
			`UPDATE todo_lists
			 SET google_account_id = $3, name = $4, sort_order = $5, updated_at = $6
			 WHERE user_id = $1 AND google_id = $2`,
			userID,
			taskList.GoogleID,
			accountID,
			taskList.Name,
			idx,
			now,
		)
		if err != nil {
			return fmt.Errorf("update todo list: %w", err)
		}

		var localListID string
		queryErr := s.db.QueryRow(
			ctx,
			`SELECT id FROM todo_lists WHERE user_id = $1 AND google_id = $2`,
			userID,
			taskList.GoogleID,
		).Scan(&localListID)
		if queryErr != nil {
			if queryErr != pgx.ErrNoRows {
				return fmt.Errorf("query todo list id: %w", queryErr)
			}
			localListID = uuid.New().String()
			_, err = s.db.Exec(ctx,
				`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				localListID,
				userID,
				taskList.GoogleID,
				accountID,
				taskList.Name,
				idx,
				now,
				now,
			)
			if err != nil {
				return fmt.Errorf("insert todo list: %w", err)
			}
		}
		localListIDByGoogleID[taskList.GoogleID] = localListID
	}

	for googleListID, localListID := range localListIDByGoogleID {
		pageToken := ""
		for {
			page, err := s.googleAuth.ListTasks(ctx, token, googleListID, pageToken)
			if err != nil {
				return fmt.Errorf("list google tasks: %w", err)
			}

			for idx, task := range page.Items {
				if task.IsDeleted {
					_, _ = s.db.Exec(ctx,
						`UPDATE todos
						 SET deleted_at = $4, updated_at = $4
						 WHERE user_id = $1 AND list_id = $2 AND google_id = $3`,
						userID,
						localListID,
						task.GoogleID,
						now,
					)
					continue
				}

				tag, err := s.db.Exec(ctx,
					`UPDATE todos
					 SET title = $4, notes = $5, due = $6, is_done = $7, done_at = $8, deleted_at = NULL, sort_order = $9, updated_at = $10
					 WHERE user_id = $1 AND list_id = $2 AND google_id = $3`,
					userID,
					localListID,
					task.GoogleID,
					task.Title,
					task.Notes,
					task.DueDate,
					task.IsDone,
					completedAt(task, now),
					idx,
					now,
				)
				if err != nil {
					return fmt.Errorf("update todo: %w", err)
				}
				if tag.RowsAffected() > 0 {
					continue
				}

				_, err = s.db.Exec(ctx,
					`INSERT INTO todos (
					   id, list_id, user_id, parent_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, done_at, created_at, updated_at
					 ) VALUES (
					   $1, $2, $3, NULL, $4, $5, $6, $7, 'normal', $8, FALSE, $9, $10, $11, $12
					 )`,
					uuid.New().String(),
					localListID,
					userID,
					task.GoogleID,
					task.Title,
					task.Notes,
					task.DueDate,
					task.IsDone,
					idx,
					completedAt(task, now),
					now,
					now,
				)
				if err != nil {
					return fmt.Errorf("insert todo: %w", err)
				}
			}

			if page.NextPageToken == "" {
				break
			}
			pageToken = page.NextPageToken
		}

		// 완료 항목은 최근 90일만 유지.
		_, _ = s.db.Exec(ctx,
			`UPDATE todos
			 SET deleted_at = $4, updated_at = $4
			 WHERE user_id = $1 AND list_id = $2
			   AND is_done = TRUE
			   AND done_at IS NOT NULL
			   AND done_at < $3
			   AND deleted_at IS NULL`,
			userID,
			localListID,
			now.AddDate(0, 0, -90),
			now,
		)
	}

	return nil
}

func completedAt(task portout.OAuthTask, now time.Time) *time.Time {
	if !task.IsDone {
		return nil
	}
	if task.CompletedAt != nil {
		return task.CompletedAt
	}
	t := now
	return &t
}

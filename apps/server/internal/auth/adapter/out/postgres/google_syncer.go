package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
	calendarportout "lifebase/internal/calendar/port/out"
)

type googleAccountSyncer struct {
	db         *pgxpool.Pool
	googleAuth portout.GoogleAuthClient
}

type googleSyncRow interface {
	Scan(dest ...any) error
}

var queryGoogleSyncRowsFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
	return db.Query(ctx, sql, args...)
}

var execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
	return db.Exec(ctx, sql, args...)
}

var queryGoogleSyncRowFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) googleSyncRow {
	return db.QueryRow(ctx, sql, args...)
}

var googleSyncTryAdvisoryLockFn = func(ctx context.Context, conn *pgxpool.Conn, lockKey int64) (bool, error) {
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		return false, err
	}
	return locked, nil
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
	_, _ = s.db.Exec(ctx,
		`DELETE FROM events
		  WHERE user_id = $1
		    AND calendar_id IN (
		      SELECT id
		        FROM calendars
		       WHERE user_id = $1
		         AND google_account_id = $2
		         AND (is_special = TRUE OR kind IN ('holiday', 'birthday'))
		    )`,
		userID,
		accountID,
	)
	_, _ = s.db.Exec(ctx,
		`DELETE FROM calendars
		  WHERE user_id = $1
		    AND google_account_id = $2
		    AND (is_special = TRUE OR kind IN ('holiday', 'birthday'))`,
		userID,
		accountID,
	)

	calendars, err := s.googleAuth.ListCalendars(ctx, token)
	if err != nil {
		return fmt.Errorf("list google calendars: %w", err)
	}

	localCalendarIDByGoogleID := make(map[string]string, len(calendars))
	for _, cal := range calendars {
		if cal.IsSpecial || cal.Kind == "holiday" || cal.Kind == "birthday" {
			continue
		}

		_, err := execGoogleSyncFn(ctx, s.db,
			`UPDATE calendars
				 SET google_account_id = $4, name = $3, kind = $5, color_id = $6, is_primary = $7, is_visible = $8,
			     is_readonly = $9, is_special = $10, updated_at = $11
			 WHERE user_id = $1 AND google_id = $2`,
			userID, cal.GoogleID, cal.Name, accountID, cal.Kind, cal.ColorID, cal.IsPrimary, cal.IsVisible, cal.IsReadOnly, cal.IsSpecial, now,
		)
		if err != nil {
			return fmt.Errorf("update calendar: %w", err)
		}

		var localCalendarID string
		queryErr := queryGoogleSyncRowFn(
			ctx,
			s.db,
			`SELECT id FROM calendars WHERE user_id = $1 AND google_id = $2`,
			userID,
			cal.GoogleID,
		).Scan(&localCalendarID)
		if queryErr != nil {
			if queryErr != pgx.ErrNoRows {
				return fmt.Errorf("query calendar id: %w", queryErr)
			}
			localCalendarID = uuid.New().String()
			_, err = execGoogleSyncFn(ctx, s.db,
				`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, color_id, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
				localCalendarID, userID, cal.GoogleID, accountID, cal.Name, cal.Kind, cal.ColorID, cal.IsPrimary, cal.IsVisible, cal.IsReadOnly, cal.IsSpecial, now, now,
			)
			if err != nil {
				return fmt.Errorf("insert calendar: %w", err)
			}
		}
		localCalendarIDByGoogleID[cal.GoogleID] = localCalendarID
	}

	start := now.AddDate(-1, 0, 0)
	end := now.AddDate(2, 0, 0)
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
				// syncToken 만료 계열은 1회 full sync로 재시도
				if currentSyncToken != "" && shouldResetGoogleSyncToken(err) {
					currentSyncToken = ""
					pageToken = ""
					nextSyncToken = ""
					continue
				}
				return fmt.Errorf("list google events: %w", err)
			}

			for _, event := range page.Events {
			if _, _, err := s.applyOAuthEvent(ctx, userID, localCalendarID, event, now); err != nil {
				return err
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
		_ = s.expandCalendarCoverage(ctx, userID, localCalendarID, start, end, now)
	}

	return nil
}

func (s *googleAccountSyncer) BackfillEvents(
	ctx context.Context,
	userID string,
	start, end time.Time,
	calendarIDs []string,
) (*calendarportout.CalendarBackfillResult, error) {
	if end.Before(start) || end.Equal(start) {
		return nil, fmt.Errorf("invalid backfill range")
	}

	type rowData struct {
		localCalendarID  string
		googleCalendarID string
		accountID        string
		accessToken      string
		refreshToken     string
		tokenExpiresAt   *time.Time
	}
	args := []any{userID}
	query := `SELECT c.id, c.google_id, c.google_account_id, a.access_token, a.refresh_token, a.token_expires_at
	          FROM calendars c
	          JOIN user_google_accounts a ON a.id::text = c.google_account_id AND a.user_id::text = c.user_id
	         WHERE c.user_id = $1
		           AND c.google_id IS NOT NULL
		           AND c.google_account_id IS NOT NULL
		           AND c.is_special = FALSE
		           AND c.kind NOT IN ('holiday', 'birthday')
		           AND a.status = 'active'`
	if len(calendarIDs) > 0 {
		placeholders := make([]string, 0, len(calendarIDs))
		for _, calendarID := range calendarIDs {
			if strings.TrimSpace(calendarID) == "" {
				continue
			}
			args = append(args, calendarID)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		if len(placeholders) > 0 {
			query += " AND c.id IN (" + strings.Join(placeholders, ",") + ")"
		}
	}
	rows, err := queryGoogleSyncRowsFn(ctx, s.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	calsByAccount := map[string][]rowData{}
	for rows.Next() {
		var row rowData
		if err := rows.Scan(
			&row.localCalendarID,
			&row.googleCalendarID,
			&row.accountID,
			&row.accessToken,
			&row.refreshToken,
			&row.tokenExpiresAt,
		); err != nil {
			return nil, err
		}
		calsByAccount[row.accountID] = append(calsByAccount[row.accountID], row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &calendarportout.CalendarBackfillResult{
		CoveredStart: start,
		CoveredEnd:   end,
	}
	now := time.Now()
	for accountID, calendars := range calsByAccount {
		lockKey := advisoryLockKey(fmt.Sprintf("%s:%d:%d", accountID, start.Unix(), end.Unix()))
		lockConn, acquireErr := s.db.Acquire(ctx)
		if acquireErr != nil {
			return nil, acquireErr
		}

			locked, err := googleSyncTryAdvisoryLockFn(ctx, lockConn, lockKey)
			if err != nil {
				lockConn.Release()
				return nil, err
			}
		if !locked {
			lockConn.Release()
			continue
		}

		func() {
			defer lockConn.Release()
			defer lockConn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)

			token := portout.OAuthToken{
				AccessToken:  calendars[0].accessToken,
				RefreshToken: calendars[0].refreshToken,
			}
			if calendars[0].tokenExpiresAt != nil {
				token.Expiry = *calendars[0].tokenExpiresAt
			}

			for _, cal := range calendars {
				pageToken := ""
				for {
					page, listErr := s.googleAuth.ListCalendarEvents(
						ctx,
						token,
						cal.googleCalendarID,
						pageToken,
						"",
						&start,
						&end,
					)
					if listErr != nil {
						err = fmt.Errorf("list google events: %w", listErr)
						return
					}
					for _, event := range page.Events {
						result.FetchedEvents++
						updated, deleted, applyErr := s.applyOAuthEvent(ctx, userID, cal.localCalendarID, event, now)
						if applyErr != nil {
							err = applyErr
							return
						}
						if updated {
							result.UpdatedEvents++
						}
						if deleted {
							result.DeletedEvents++
						}
					}
					if page.NextPageToken == "" {
						break
					}
					pageToken = page.NextPageToken
				}
				_ = s.expandCalendarCoverage(ctx, userID, cal.localCalendarID, start, end, now)
			}
		}()
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *googleAccountSyncer) applyOAuthEvent(
	ctx context.Context,
	userID, localCalendarID string,
	event portout.OAuthCalendarEvent,
	now time.Time,
) (updated bool, deleted bool, err error) {
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
		return false, true, nil
	}

		tag, err := execGoogleSyncFn(ctx, s.db,
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
		return false, false, fmt.Errorf("update event: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return true, false, nil
	}

		_, err = execGoogleSyncFn(ctx, s.db,
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
		return false, false, fmt.Errorf("insert event: %w", err)
	}
	return true, false, nil
}

func (s *googleAccountSyncer) expandCalendarCoverage(
	ctx context.Context,
	userID, calendarID string,
	start, end, now time.Time,
) error {
	_, _ = s.db.Exec(ctx,
		`UPDATE calendars
		    SET synced_start = CASE
		                         WHEN synced_start IS NULL THEN $3
		                         WHEN synced_start > $3 THEN $3
		                         ELSE synced_start
		                       END,
		        synced_end = CASE
		                       WHEN synced_end IS NULL THEN $4
		                       WHEN synced_end < $4 THEN $4
		                       ELSE synced_end
		                     END,
		        updated_at = $5
		  WHERE id = $1 AND user_id = $2`,
		calendarID,
		userID,
		start,
		end,
		now,
	)

	_, _ = s.db.Exec(ctx,
		`INSERT INTO calendar_backfill_state (user_id, calendar_id, covered_start, covered_end, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id, calendar_id)
		 DO UPDATE SET
		   covered_start = CASE
		                     WHEN calendar_backfill_state.covered_start > EXCLUDED.covered_start THEN EXCLUDED.covered_start
		                     ELSE calendar_backfill_state.covered_start
		                   END,
		   covered_end = CASE
		                   WHEN calendar_backfill_state.covered_end < EXCLUDED.covered_end THEN EXCLUDED.covered_end
		                   ELSE calendar_backfill_state.covered_end
		                 END,
		   updated_at = EXCLUDED.updated_at`,
		userID,
		calendarID,
		start,
		end,
		now,
	)
	return nil
}

func (s *googleAccountSyncer) syncTaskListsAndTodos(
	ctx context.Context,
	userID, accountID string,
	token portout.OAuthToken,
	now time.Time,
) error {
	doneRetentionCutoff := s.resolveTodoDoneRetentionCutoff(ctx, userID, now)

	taskLists, err := s.googleAuth.ListTaskLists(ctx, token)
	if err != nil {
		return fmt.Errorf("list google task lists: %w", err)
	}

	localListIDByGoogleID := make(map[string]string, len(taskLists))
	for idx, taskList := range taskLists {
		_, err := execGoogleSyncFn(ctx, s.db,
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
		queryErr := queryGoogleSyncRowFn(
			ctx,
			s.db,
			`SELECT id FROM todo_lists WHERE user_id = $1 AND google_id = $2`,
			userID,
			taskList.GoogleID,
		).Scan(&localListID)
		if queryErr != nil {
			if queryErr != pgx.ErrNoRows {
				return fmt.Errorf("query todo list id: %w", queryErr)
			}
			localListID = uuid.New().String()
			_, err = execGoogleSyncFn(ctx, s.db,
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
		seenGoogleIDs := make([]string, 0, 128)
		seenGoogleIDSet := make(map[string]struct{})
		for {
			page, err := s.googleAuth.ListTasks(ctx, token, googleListID, pageToken)
			if err != nil {
				return fmt.Errorf("list google tasks: %w", err)
			}

			for idx, task := range page.Items {
				if task.GoogleID != "" {
					if _, exists := seenGoogleIDSet[task.GoogleID]; !exists {
						seenGoogleIDSet[task.GoogleID] = struct{}{}
						seenGoogleIDs = append(seenGoogleIDs, task.GoogleID)
					}
				}

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

				tag, err := execGoogleSyncFn(ctx, s.db,
					`UPDATE todos
					 SET title = $4, notes = $5, due_date = $6, due_time = NULL, is_done = $7, done_at = $8, deleted_at = NULL, sort_order = $9, updated_at = $10
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

				_, err = execGoogleSyncFn(ctx, s.db,
					`INSERT INTO todos (
					   id, list_id, user_id, parent_id, google_id, title, notes, due_date, due_time, priority, is_done, is_pinned, sort_order, done_at, created_at, updated_at
					 ) VALUES (
					   $1, $2, $3, NULL, $4, $5, $6, $7, NULL, 'normal', $8, FALSE, $9, $10, $11, $12
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

		// Some Google Task deletions are not always returned as tombstones.
		// For full-list polling, treat locally cached items that were not seen as deleted.
		if len(seenGoogleIDs) == 0 {
			_, _ = s.db.Exec(ctx,
				`UPDATE todos
				 SET deleted_at = $3, updated_at = $3
				 WHERE user_id = $1 AND list_id = $2
				   AND google_id IS NOT NULL
				   AND deleted_at IS NULL`,
				userID,
				localListID,
				now,
			)
		} else {
			_, _ = s.db.Exec(ctx,
				`UPDATE todos
				 SET deleted_at = $4, updated_at = $4
				 WHERE user_id = $1 AND list_id = $2
				   AND google_id IS NOT NULL
				   AND NOT (google_id = ANY($3::text[]))
				   AND deleted_at IS NULL`,
				userID,
				localListID,
				seenGoogleIDs,
				now,
			)
		}

		if doneRetentionCutoff != nil {
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
				*doneRetentionCutoff,
				now,
			)
		}
	}

	return nil
}

func (s *googleAccountSyncer) resolveTodoDoneRetentionCutoff(
	ctx context.Context,
	userID string,
	now time.Time,
) *time.Time {
	const defaultPeriod = "1y"

	var raw string
	err := s.db.QueryRow(
		ctx,
		`SELECT value FROM user_settings WHERE user_id = $1 AND key = 'todo_done_retention_period'`,
		userID,
	).Scan(&raw)
	if err != nil && err != pgx.ErrNoRows {
		raw = defaultPeriod
	}
	if err == pgx.ErrNoRows {
		raw = defaultPeriod
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1m":
		t := now.AddDate(0, -1, 0)
		return &t
	case "3m":
		t := now.AddDate(0, -3, 0)
		return &t
	case "6m":
		t := now.AddDate(0, -6, 0)
		return &t
	case "1y":
		t := now.AddDate(-1, 0, 0)
		return &t
	case "3y":
		t := now.AddDate(-3, 0, 0)
		return &t
	case "unlimited":
		return nil
	default:
		t := now.AddDate(-1, 0, 0)
		return &t
	}
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

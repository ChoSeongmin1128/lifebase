package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
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
		calendars, err := s.googleAuth.ListCalendars(ctx, token)
		if err != nil {
			return fmt.Errorf("list google calendars: %w", err)
		}
		for _, cal := range calendars {
			tag, err := s.db.Exec(ctx,
				`UPDATE calendars
				 SET google_account_id = $4, name = $3, color_id = $5, is_primary = $6, is_visible = $7, updated_at = $8
				 WHERE user_id = $1 AND google_id = $2`,
				userID, cal.GoogleID, cal.Name, account.ID, cal.ColorID, cal.IsPrimary, cal.IsVisible, now,
			)
			if err != nil {
				return fmt.Errorf("update calendar: %w", err)
			}
			if tag.RowsAffected() > 0 {
				continue
			}

			_, err = s.db.Exec(ctx,
				`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, color_id, is_primary, is_visible, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				uuid.New().String(), userID, cal.GoogleID, account.ID, cal.Name, cal.ColorID, cal.IsPrimary, cal.IsVisible, now, now,
			)
			if err != nil {
				return fmt.Errorf("insert calendar: %w", err)
			}
		}
	}

	if options.SyncTodo {
		taskLists, err := s.googleAuth.ListTaskLists(ctx, token)
		if err != nil {
			return fmt.Errorf("list google task lists: %w", err)
		}
		for idx, taskList := range taskLists {
			tag, err := s.db.Exec(ctx,
				`UPDATE todo_lists
				 SET name = $3, sort_order = $4, updated_at = $5
				 WHERE user_id = $1 AND google_id = $2`,
				userID, taskList.GoogleID, taskList.Name, idx, now,
			)
			if err != nil {
				return fmt.Errorf("update todo list: %w", err)
			}
			if tag.RowsAffected() > 0 {
				continue
			}

			_, err = s.db.Exec(ctx,
				`INSERT INTO todo_lists (id, user_id, google_id, name, sort_order, created_at, updated_at)
				 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				uuid.New().String(), userID, taskList.GoogleID, taskList.Name, idx, now, now,
			)
			if err != nil {
				return fmt.Errorf("insert todo list: %w", err)
			}
		}
	}

	return nil
}

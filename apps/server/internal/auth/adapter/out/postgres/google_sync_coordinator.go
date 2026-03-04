package postgres

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
)

type googleSyncCoordinator struct {
	db     *pgxpool.Pool
	syncer portout.GoogleAccountSyncer
}

func NewGoogleSyncCoordinator(db *pgxpool.Pool, syncer portout.GoogleAccountSyncer) *googleSyncCoordinator {
	return &googleSyncCoordinator{db: db, syncer: syncer}
}

func (c *googleSyncCoordinator) TriggerUserSync(ctx context.Context, userID, area, reason string) (int, error) {
	if userID == "" {
		return 0, fmt.Errorf("user id is required")
	}
	if c.syncer == nil {
		return 0, nil
	}

	accounts, err := c.listActiveAccountsByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	scheduled := 0
	for _, account := range accounts {
		options, enabled, err := c.resolveSyncOptions(ctx, userID, account.ID, area)
		if err != nil {
			continue
		}
		if !enabled {
			continue
		}
		performed, err := c.syncAccountIfDue(ctx, userID, account, options, reason)
		if err != nil {
			continue
		}
		if performed {
			scheduled++
		}
	}

	return scheduled, nil
}

func (c *googleSyncCoordinator) RunHourlySync(ctx context.Context) (int, error) {
	if c.syncer == nil {
		return 0, nil
	}
	accounts, err := c.listActiveAccounts(ctx)
	if err != nil {
		return 0, err
	}

	scheduled := 0
	for _, account := range accounts {
		options, enabled, err := c.resolveSyncOptions(ctx, account.UserID, account.ID, "both")
		if err != nil {
			continue
		}
		if !enabled {
			continue
		}
		performed, err := c.syncAccountIfDue(ctx, account.UserID, account, options, "background")
		if err != nil {
			continue
		}
		if performed {
			scheduled++
		}
	}
	return scheduled, nil
}

func (c *googleSyncCoordinator) syncAccountIfDue(
	ctx context.Context,
	userID string,
	account *domain.GoogleAccount,
	options portout.GoogleSyncOptions,
	reason string,
) (bool, error) {
	if account == nil {
		return false, nil
	}
	if !options.SyncCalendar && !options.SyncTodo {
		return false, nil
	}

	lockKey := advisoryLockKey(account.ID)
	var locked bool
	if err := c.db.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		return false, err
	}
	if !locked {
		return false, nil
	}
	defer c.db.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)

	now := time.Now()
	lastAt, err := c.lastSyncAt(ctx, account.ID, reason)
	if err != nil {
		return false, err
	}
	minInterval := minIntervalForReason(reason)
	if !lastAt.IsZero() && minInterval > 0 && now.Sub(lastAt) < minInterval {
		return false, nil
	}

	if err := c.touchSyncReason(ctx, account.ID, userID, reason, now); err != nil {
		return false, err
	}

	err = c.syncer.SyncAccount(ctx, userID, account, options)
	if err != nil {
		_ = c.updateSyncError(ctx, account.ID, err.Error(), now)
		return true, err
	}
	if err := c.updateSyncSuccess(ctx, account.ID, now); err != nil {
		return true, err
	}

	return true, nil
}

func minIntervalForReason(reason string) time.Duration {
	switch reason {
	case "hourly":
		return time.Hour
	case "background":
		return 10 * time.Minute
	case "tab_heartbeat":
		return 10 * time.Minute
	case "page_action":
		return 2 * time.Minute
	case "page_enter":
		return 0
	case "manual":
		return 0
	default:
		return 2 * time.Minute
	}
}

func (c *googleSyncCoordinator) resolveSyncOptions(
	ctx context.Context,
	userID, accountID, area string,
) (portout.GoogleSyncOptions, bool, error) {
	calendarEnabled, err := c.getSettingBool(ctx, userID, "google_account_sync_calendar_"+accountID, true)
	if err != nil {
		return portout.GoogleSyncOptions{}, false, err
	}
	todoEnabled, err := c.getSettingBool(ctx, userID, "google_account_sync_todo_"+accountID, true)
	if err != nil {
		return portout.GoogleSyncOptions{}, false, err
	}

	area = strings.TrimSpace(strings.ToLower(area))
	options := portout.GoogleSyncOptions{SyncCalendar: calendarEnabled, SyncTodo: todoEnabled}
	switch area {
	case "calendar":
		options.SyncTodo = false
	case "todo":
		options.SyncCalendar = false
	case "both", "":
	default:
		// unknown area: do not sync to be safe
		return options, false, nil
	}

	return options, options.SyncCalendar || options.SyncTodo, nil
}

func (c *googleSyncCoordinator) getSettingBool(ctx context.Context, userID, key string, fallback bool) (bool, error) {
	var value string
	err := c.db.QueryRow(ctx,
		`SELECT value FROM user_settings WHERE user_id = $1 AND key = $2`,
		userID,
		key,
	).Scan(&value)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fallback, nil
		}
		return fallback, err
	}
	if strings.EqualFold(value, "false") {
		return false, nil
	}
	if strings.EqualFold(value, "true") {
		return true, nil
	}
	return fallback, nil
}

func (c *googleSyncCoordinator) lastSyncAt(ctx context.Context, accountID, reason string) (time.Time, error) {
	var t *time.Time
	query := `SELECT NULL::timestamptz`
	switch reason {
	case "hourly", "background":
		query = `SELECT last_hourly_sync_at FROM google_sync_state WHERE account_id = $1`
	case "tab_heartbeat":
		query = `SELECT last_tab_sync_at FROM google_sync_state WHERE account_id = $1`
	case "page_action":
		query = `SELECT last_action_sync_at FROM google_sync_state WHERE account_id = $1`
	case "page_enter":
		query = `SELECT last_nav_sync_at FROM google_sync_state WHERE account_id = $1`
	case "manual":
		return time.Time{}, nil
	default:
		query = `SELECT last_action_sync_at FROM google_sync_state WHERE account_id = $1`
	}

	err := c.db.QueryRow(ctx, query, accountID).Scan(&t)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, err
	}
	if t == nil {
		return time.Time{}, nil
	}
	return *t, nil
}

func (c *googleSyncCoordinator) touchSyncReason(ctx context.Context, accountID, userID, reason string, now time.Time) error {
	switch reason {
	case "hourly", "background":
		_, err := c.db.Exec(ctx,
			`INSERT INTO google_sync_state (account_id, user_id, last_hourly_sync_at, updated_at)
			 VALUES ($1, $2, $3, $3)
			 ON CONFLICT (account_id)
			 DO UPDATE SET user_id = EXCLUDED.user_id, last_hourly_sync_at = EXCLUDED.last_hourly_sync_at, updated_at = EXCLUDED.updated_at`,
			accountID, userID, now,
		)
		return err
	case "tab_heartbeat":
		_, err := c.db.Exec(ctx,
			`INSERT INTO google_sync_state (account_id, user_id, last_tab_sync_at, updated_at)
			 VALUES ($1, $2, $3, $3)
			 ON CONFLICT (account_id)
			 DO UPDATE SET user_id = EXCLUDED.user_id, last_tab_sync_at = EXCLUDED.last_tab_sync_at, updated_at = EXCLUDED.updated_at`,
			accountID, userID, now,
		)
		return err
	case "page_enter":
		_, err := c.db.Exec(ctx,
			`INSERT INTO google_sync_state (account_id, user_id, last_nav_sync_at, updated_at)
			 VALUES ($1, $2, $3, $3)
			 ON CONFLICT (account_id)
			 DO UPDATE SET user_id = EXCLUDED.user_id, last_nav_sync_at = EXCLUDED.last_nav_sync_at, updated_at = EXCLUDED.updated_at`,
			accountID, userID, now,
		)
		return err
	default:
		_, err := c.db.Exec(ctx,
			`INSERT INTO google_sync_state (account_id, user_id, last_action_sync_at, updated_at)
			 VALUES ($1, $2, $3, $3)
			 ON CONFLICT (account_id)
			 DO UPDATE SET user_id = EXCLUDED.user_id, last_action_sync_at = EXCLUDED.last_action_sync_at, updated_at = EXCLUDED.updated_at`,
			accountID, userID, now,
		)
		return err
	}
}

func (c *googleSyncCoordinator) updateSyncSuccess(ctx context.Context, accountID string, now time.Time) error {
	_, err := c.db.Exec(ctx,
		`UPDATE google_sync_state
		 SET last_success_at = $2, last_error = NULL, updated_at = $2
		 WHERE account_id = $1`,
		accountID, now,
	)
	return err
}

func (c *googleSyncCoordinator) updateSyncError(ctx context.Context, accountID, message string, now time.Time) error {
	_, err := c.db.Exec(ctx,
		`UPDATE google_sync_state
		 SET last_error = $2, updated_at = $3
		 WHERE account_id = $1`,
		accountID, message, now,
	)
	return err
}

func (c *googleSyncCoordinator) listActiveAccountsByUser(ctx context.Context, userID string) ([]*domain.GoogleAccount, error) {
	rows, err := c.db.Query(ctx,
		`SELECT id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at,
		        scopes, status, is_primary, connected_at, created_at, updated_at
		 FROM user_google_accounts
		 WHERE user_id = $1 AND status = 'active'
		 ORDER BY is_primary DESC, connected_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]*domain.GoogleAccount, 0)
	for rows.Next() {
		var a domain.GoogleAccount
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.GoogleEmail,
			&a.GoogleID,
			&a.AccessToken,
			&a.RefreshToken,
			&a.TokenExpiresAt,
			&a.Scopes,
			&a.Status,
			&a.IsPrimary,
			&a.ConnectedAt,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (c *googleSyncCoordinator) listActiveAccounts(ctx context.Context) ([]*domain.GoogleAccount, error) {
	rows, err := c.db.Query(ctx,
		`SELECT id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at,
		        scopes, status, is_primary, connected_at, created_at, updated_at
		 FROM user_google_accounts
		 WHERE status = 'active'
		 ORDER BY updated_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := make([]*domain.GoogleAccount, 0)
	for rows.Next() {
		var a domain.GoogleAccount
		if err := rows.Scan(
			&a.ID,
			&a.UserID,
			&a.GoogleEmail,
			&a.GoogleID,
			&a.AccessToken,
			&a.RefreshToken,
			&a.TokenExpiresAt,
			&a.Scopes,
			&a.Status,
			&a.IsPrimary,
			&a.ConnectedAt,
			&a.CreatedAt,
			&a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func advisoryLockKey(input string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(input))
	return int64(h.Sum64())
}

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

type syncerStub struct {
	errByAccount map[string]error
	calls        int
	syncFn       func(context.Context, string, *authdomain.GoogleAccount, portout.GoogleSyncOptions) error
}

func (s *syncerStub) SyncAccount(ctx context.Context, userID string, account *authdomain.GoogleAccount, options portout.GoogleSyncOptions) error {
	s.calls++
	if s.syncFn != nil {
		return s.syncFn(ctx, userID, account, options)
	}
	if s.errByAccount != nil {
		if err, ok := s.errByAccount[account.ID]; ok {
			return err
		}
	}
	return nil
}

func TestGoogleSyncCoordinatorFlowIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	acc1 := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	acc2 := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	accInactive := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $4, 'u1@gmail.com', 'gid-1', 'at', 'rt', $5, 'scope', 'active', true, $5, $5, $5),
		    ($2, $4, 'u1+2@gmail.com', 'gid-2', 'at2', 'rt2', $5, 'scope', 'active', false, $6, $5, $5),
		    ($3, $4, 'u1+3@gmail.com', 'gid-3', 'at3', 'rt3', $5, 'scope', 'revoked', false, $7, $5, $5)`,
		acc1, acc2, accInactive, userID, now, now.Add(time.Minute), now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("insert google accounts: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES
		   ($1, $2, 'true', $3),
		   ($1, $4, 'false', $3),
		   ($1, $5, 'true', $3)`,
		userID,
		"google_account_sync_calendar_"+acc1,
		now,
		"google_account_sync_todo_"+acc1,
		"google_account_sync_calendar_"+acc2,
	)
	if err != nil {
		t.Fatalf("insert sync settings: %v", err)
	}

	stub := &syncerStub{
		errByAccount: map[string]error{
			acc2: &portout.GoogleAPIError{StatusCode: 401, Reason: "authError", Message: "reauth required"},
		},
	}
	coordinator := NewGoogleSyncCoordinator(db, stub)

	if _, err := coordinator.TriggerUserSync(ctx, "", "both", "manual"); err == nil {
		t.Fatal("expected empty user id validation error")
	}

	nilSyncCoordinator := NewGoogleSyncCoordinator(db, nil)
	if scheduled, err := nilSyncCoordinator.TriggerUserSync(ctx, userID, "both", "manual"); err != nil || scheduled != 0 {
		t.Fatalf("trigger with nil syncer failed: scheduled=%d err=%v", scheduled, err)
	}
	if scheduled, err := nilSyncCoordinator.RunHourlySync(ctx); err != nil || scheduled != 0 {
		t.Fatalf("hourly with nil syncer failed: scheduled=%d err=%v", scheduled, err)
	}

	scheduled, err := coordinator.TriggerUserSync(ctx, userID, "both", "page_action")
	if err != nil {
		t.Fatalf("trigger user sync: %v", err)
	}
	if scheduled != 1 {
		t.Fatalf("expected only one successful schedule, got %d", scheduled)
	}

	var acc2Status string
	if err := db.QueryRow(ctx, `SELECT status FROM user_google_accounts WHERE id = $1`, acc2).Scan(&acc2Status); err != nil {
		t.Fatalf("read account2 status: %v", err)
	}
	if acc2Status != "reauth_required" {
		t.Fatalf("expected acc2 reauth_required, got %s", acc2Status)
	}

	// Unknown area should be skipped safely.
	if scheduled, err := coordinator.TriggerUserSync(ctx, userID, "unknown", "manual"); err != nil || scheduled != 0 {
		t.Fatalf("trigger unknown area mismatch: scheduled=%d err=%v", scheduled, err)
	}

	// Run hourly should process active accounts (inactive/revoked excluded).
	if _, err := db.Exec(ctx,
		`UPDATE user_google_accounts SET status='active', updated_at=$2 WHERE id=$1`,
		acc2, now.Add(3*time.Minute),
	); err != nil {
		t.Fatalf("reactivate acc2: %v", err)
	}
	scheduled, err = coordinator.RunHourlySync(ctx)
	if err != nil {
		t.Fatalf("run hourly sync: %v", err)
	}
	if scheduled == 0 {
		t.Fatal("expected at least one hourly sync schedule")
	}

	// lock-busy path
	account := &authdomain.GoogleAccount{ID: acc1, UserID: userID}
	lockKey := advisoryLockKey(account.ID)
	lockConn, err := db.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock conn: %v", err)
	}
	defer lockConn.Release()
	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		t.Fatalf("acquire advisory lock: %v", err)
	}
	performed, err := coordinator.syncAccountIfDue(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true}, "manual")
	if err != nil {
		t.Fatalf("syncAccountIfDue lock-busy: %v", err)
	}
	if performed {
		t.Fatal("expected lock-busy path to skip sync")
	}
	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey); err != nil {
		t.Fatalf("release advisory lock: %v", err)
	}

	// interval skip path for page_action
	_, err = db.Exec(ctx,
		`INSERT INTO google_sync_state (account_id, user_id, last_action_sync_at, updated_at)
		 VALUES ($1, $2, $3, $3)
		 ON CONFLICT (account_id) DO UPDATE SET last_action_sync_at = EXCLUDED.last_action_sync_at, updated_at = EXCLUDED.updated_at`,
		acc1, userID, time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("seed google_sync_state: %v", err)
	}
	performed, err = coordinator.syncAccountIfDue(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true}, "page_action")
	if err != nil {
		t.Fatalf("syncAccountIfDue min interval: %v", err)
	}
	if performed {
		t.Fatal("expected min-interval path to skip sync")
	}
}

func TestGoogleSyncCoordinatorHelpersIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	coordinator := NewGoogleSyncCoordinator(db, &syncerStub{})

	// getSettingBool fallbacks and bool parsing
	got, err := coordinator.getSettingBool(ctx, userID, "missing", true)
	if err != nil || !got {
		t.Fatalf("getSettingBool fallback true failed: got=%v err=%v", got, err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, 'k_true', 'TRUE', $2), ($1, 'k_false', 'false', $2), ($1, 'k_other', 'x', $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert settings rows: %v", err)
	}
	if got, err = coordinator.getSettingBool(ctx, userID, "k_true", false); err != nil || !got {
		t.Fatalf("getSettingBool true failed: got=%v err=%v", got, err)
	}
	if got, err = coordinator.getSettingBool(ctx, userID, "k_false", true); err != nil || got {
		t.Fatalf("getSettingBool false failed: got=%v err=%v", got, err)
	}
	if got, err = coordinator.getSettingBool(ctx, userID, "k_other", true); err != nil || !got {
		t.Fatalf("getSettingBool fallback unknown failed: got=%v err=%v", got, err)
	}

	// resolveSyncOptions
	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES
		   ($1, $2, 'true', $3),
		   ($1, $4, 'false', $3)`,
		userID,
		"google_account_sync_calendar_"+accountID,
		now,
		"google_account_sync_todo_"+accountID,
	)
	if err != nil {
		t.Fatalf("insert option settings: %v", err)
	}
	opts, enabled, err := coordinator.resolveSyncOptions(ctx, userID, accountID, "calendar")
	if err != nil || !enabled || !opts.SyncCalendar || opts.SyncTodo {
		t.Fatalf("resolve calendar area failed: opts=%+v enabled=%v err=%v", opts, enabled, err)
	}
	opts, enabled, err = coordinator.resolveSyncOptions(ctx, userID, accountID, "todo")
	if err != nil || enabled {
		t.Fatalf("resolve todo area expected disabled: opts=%+v enabled=%v err=%v", opts, enabled, err)
	}
	if _, enabled, err = coordinator.resolveSyncOptions(ctx, userID, accountID, "unknown"); err != nil || enabled {
		t.Fatalf("resolve unknown area mismatch: enabled=%v err=%v", enabled, err)
	}

	// touch sync reason + last sync getters
	reasons := []string{"hourly", "tab_heartbeat", "page_enter", "page_action"}
	for _, reason := range reasons {
		if err := coordinator.touchSyncReason(ctx, accountID, userID, reason, now); err != nil {
			t.Fatalf("touchSyncReason(%s): %v", reason, err)
		}
		if _, err := coordinator.lastSyncAt(ctx, accountID, reason); err != nil {
			t.Fatalf("lastSyncAt(%s): %v", reason, err)
		}
	}
	if _, err := coordinator.lastSyncAt(ctx, accountID, "manual"); err != nil {
		t.Fatalf("lastSyncAt(manual): %v", err)
	}
	if _, err := coordinator.lastSyncAt(ctx, accountID, "custom"); err != nil {
		t.Fatalf("lastSyncAt(custom): %v", err)
	}

	if err := coordinator.updateSyncSuccess(ctx, accountID, now); err != nil {
		t.Fatalf("updateSyncSuccess: %v", err)
	}
	if err := coordinator.updateSyncError(ctx, accountID, "err", now); err != nil {
		t.Fatalf("updateSyncError: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert account for markAccountReauthRequired: %v", err)
	}
	if err := coordinator.markAccountReauthRequired(ctx, accountID); err != nil {
		t.Fatalf("markAccountReauthRequired: %v", err)
	}
}

func TestGoogleSyncCoordinatorListActiveAccountsIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const user1 = "11111111-1111-1111-1111-111111111111"
	const user2 = "22222222-2222-2222-2222-222222222222"
	accPrimary := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	accOlder := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	accNewer := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	accOtherUser := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	accRevoked := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $2, 'primary@gmail.com', 'gid-1', 'at', 'rt', $8, 'scope', 'active', true,  $8, $8, $8),
		    ($3, $2, 'older@gmail.com',   'gid-2', 'at', 'rt', $9, 'scope', 'active', false, $9, $8, $9),
		    ($4, $2, 'newer@gmail.com',   'gid-3', 'at', 'rt', $10,'scope', 'active', false, $10, $8, $10),
		    ($5, $6, 'other@gmail.com',   'gid-4', 'at', 'rt', $11,'scope', 'active', false, $11, $8, $11),
		    ($7, $2, 'revoked@gmail.com', 'gid-5', 'at', 'rt', $12,'scope', 'revoked', false, $12, $8, $12)`,
		accPrimary, user1, accOlder, accNewer, accOtherUser, user2, accRevoked,
		now, now.Add(time.Minute), now.Add(2*time.Minute), now.Add(3*time.Minute), now.Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}

	byUser, err := NewGoogleSyncCoordinator(db, &syncerStub{}).listActiveAccountsByUser(ctx, user1)
	if err != nil {
		t.Fatalf("listActiveAccountsByUser: %v", err)
	}
	if len(byUser) != 3 {
		t.Fatalf("expected 3 active accounts for user1, got %d", len(byUser))
	}
	if byUser[0].ID != accPrimary || byUser[1].ID != accOlder || byUser[2].ID != accNewer {
		t.Fatalf("unexpected user ordering: [%s %s %s]", byUser[0].ID, byUser[1].ID, byUser[2].ID)
	}

	all, err := NewGoogleSyncCoordinator(db, &syncerStub{}).listActiveAccounts(ctx)
	if err != nil {
		t.Fatalf("listActiveAccounts: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 active accounts total, got %d", len(all))
	}
	if all[0].ID != accPrimary || all[1].ID != accOlder || all[2].ID != accNewer || all[3].ID != accOtherUser {
		t.Fatalf("unexpected global ordering: [%s %s %s %s]", all[0].ID, all[1].ID, all[2].ID, all[3].ID)
	}
}

func TestGoogleSyncCoordinatorResolveSyncOptionsBothAndEmptyIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES
		   ($1, $2, 'true', $4),
		   ($1, $3, 'true', $4)`,
		userID,
		"google_account_sync_calendar_"+accountID,
		"google_account_sync_todo_"+accountID,
		now,
	)
	if err != nil {
		t.Fatalf("insert options: %v", err)
	}

	coordinator := NewGoogleSyncCoordinator(db, &syncerStub{})
	opts, enabled, err := coordinator.resolveSyncOptions(ctx, userID, accountID, " BOTH ")
	if err != nil || !enabled || !opts.SyncCalendar || !opts.SyncTodo {
		t.Fatalf("resolve BOTH failed: opts=%+v enabled=%v err=%v", opts, enabled, err)
	}
	opts, enabled, err = coordinator.resolveSyncOptions(ctx, userID, accountID, "")
	if err != nil || !enabled || !opts.SyncCalendar || !opts.SyncTodo {
		t.Fatalf("resolve empty area failed: opts=%+v enabled=%v err=%v", opts, enabled, err)
	}
}

func TestGoogleSyncCoordinatorTriggerAndHourlySkipBranchesIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	accDisabled := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	accInterval := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $3, 'disabled@gmail.com', 'gid-1', 'at', 'rt', $4, 'scope', 'active', false, $4, $4, $4),
		    ($2, $3, 'interval@gmail.com', 'gid-2', 'at', 'rt', $5, 'scope', 'active', false, $5, $4, $5)`,
		accDisabled, accInterval, userID, now, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES
		   ($1, $2, 'false', $6),
		   ($1, $3, 'false', $6),
		   ($1, $4, 'true', $6),
		   ($1, $5, 'true', $6)`,
		userID,
		"google_account_sync_calendar_"+accDisabled,
		"google_account_sync_todo_"+accDisabled,
		"google_account_sync_calendar_"+accInterval,
		"google_account_sync_todo_"+accInterval,
		now,
	)
	if err != nil {
		t.Fatalf("insert settings: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO google_sync_state (account_id, user_id, last_action_sync_at, last_hourly_sync_at, updated_at)
		 VALUES ($1, $2, $3, $3, $3)`,
		accInterval, userID, now,
	)
	if err != nil {
		t.Fatalf("seed google_sync_state: %v", err)
	}

	stub := &syncerStub{}
	coordinator := NewGoogleSyncCoordinator(db, stub)

	scheduled, err := coordinator.TriggerUserSync(ctx, userID, "both", "page_action")
	if err != nil {
		t.Fatalf("TriggerUserSync: %v", err)
	}
	if scheduled != 0 {
		t.Fatalf("expected no page_action sync due to disabled/interval branches, got %d", scheduled)
	}
	if stub.calls != 0 {
		t.Fatalf("expected no syncer calls for TriggerUserSync skips, got %d", stub.calls)
	}

	scheduled, err = coordinator.RunHourlySync(ctx)
	if err != nil {
		t.Fatalf("RunHourlySync: %v", err)
	}
	if scheduled != 0 {
		t.Fatalf("expected no hourly sync due to disabled/interval branches, got %d", scheduled)
	}
	if stub.calls != 0 {
		t.Fatalf("expected no syncer calls after RunHourlySync skips, got %d", stub.calls)
	}
}

func TestGoogleSyncCoordinatorSyncAccountIfDueEarlyReturnAndNonAuthErrorIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	stub := &syncerStub{errByAccount: map[string]error{accountID: errors.New("sync exploded")}}
	coordinator := NewGoogleSyncCoordinator(db, stub)

	performed, err := coordinator.syncAccountIfDue(ctx, userID, nil, portout.GoogleSyncOptions{SyncCalendar: true}, "manual")
	if err != nil || performed {
		t.Fatalf("nil account branch mismatch: performed=%v err=%v", performed, err)
	}

	performed, err = coordinator.syncAccountIfDue(ctx, userID, &authdomain.GoogleAccount{ID: accountID, UserID: userID}, portout.GoogleSyncOptions{}, "manual")
	if err != nil || performed {
		t.Fatalf("both sync flags off branch mismatch: performed=%v err=%v", performed, err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	performed, err = coordinator.syncAccountIfDue(
		ctx,
		userID,
		&authdomain.GoogleAccount{ID: accountID, UserID: userID},
		portout.GoogleSyncOptions{SyncCalendar: true},
		"manual",
	)
	if !performed || err == nil {
		t.Fatalf("expected performed=true and non-auth sync error, got performed=%v err=%v", performed, err)
	}

	var status string
	if err := db.QueryRow(ctx, `SELECT status FROM user_google_accounts WHERE id = $1`, accountID).Scan(&status); err != nil {
		t.Fatalf("read account status: %v", err)
	}
	if status != "active" {
		t.Fatalf("expected active status for non-auth error path, got %s", status)
	}

	var lastError string
	if err := db.QueryRow(ctx, `SELECT COALESCE(last_error, '') FROM google_sync_state WHERE account_id = $1`, accountID).Scan(&lastError); err != nil {
		t.Fatalf("read sync error: %v", err)
	}
	if lastError == "" {
		t.Fatal("expected last_error to be recorded for non-auth sync error")
	}
}

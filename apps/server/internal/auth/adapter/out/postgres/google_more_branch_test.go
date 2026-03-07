package postgres

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

func TestGooglePushProcessorClosedPoolAndErrorBranches(t *testing.T) {
	t.Run("process_pending_claim_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		processor := NewGooglePushProcessor(db, &googleAuthStub{})
		db.Close()

		if _, err := processor.ProcessPending(context.Background(), 1); err == nil {
			t.Fatal("expected ProcessPending error on closed pool")
		}
	})

	t.Run("process_one_acquire_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		processor := NewGooglePushProcessor(db, &googleAuthStub{})
		db.Close()

		err := processor.processOne(context.Background(), pushOutboxItem{AccountID: "acc", UserID: "user"})
		if err == nil {
			t.Fatal("expected processOne acquire error on closed pool")
		}
	})

	t.Run("calendar_push_load_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		processor := NewGooglePushProcessor(db, &googleAuthStub{})
		db.Close()

		err := processor.processCalendarPush(context.Background(), portout.OAuthToken{}, pushOutboxItem{UserID: "user", LocalResourceID: "evt"})
		if err == nil {
			t.Fatal("expected processCalendarPush load error on closed pool")
		}
	})

	t.Run("todo_push_load_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		processor := NewGooglePushProcessor(db, &googleAuthStub{})
		db.Close()

		err := processor.processTodoPush(context.Background(), portout.OAuthToken{}, pushOutboxItem{UserID: "user", LocalResourceID: "todo"})
		if err == nil {
			t.Fatal("expected processTodoPush load error on closed pool")
		}
	})
}

func TestGooglePushProcessorCalendarAndTodoFallbackBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'g-cal-1', $2, 'C1', 'google', true, true, false, false, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert calendar: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, created_at, updated_at, deleted_at)
		 VALUES
		    ('evt-delete-nil', 'cal-1', $1, NULL, 'E0', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, $4),
		    ('evt-delete-err', 'cal-1', $1, 'g-evt-del', 'E1', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, $4),
		    ('evt-create-fail', 'cal-1', $1, 'g-evt-create-fail', 'E2', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-update-create-fail', 'cal-1', $1, NULL, 'E3', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-update-success', 'cal-1', $1, 'g-evt-update-success', 'E4', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-update-fallback', 'cal-1', $1, 'g-evt-update-fallback', 'E5', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL)`,
		userID, now, now.Add(time.Hour), now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1', $1, 'g-list-1', $2, 'L1', 0, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
		 VALUES
		    ('todo-delete-nil', 'list-1', $1, NULL, 'T0', '', NULL, 'normal', false, false, 0, $2, $2, $3),
		    ('todo-delete-err', 'list-1', $1, 'g-todo-del', 'T1', '', NULL, 'normal', false, false, 1, $2, $2, $3),
		    ('todo-create-fail', 'list-1', $1, 'g-todo-create-fail', 'T2', '', NULL, 'normal', false, false, 2, $2, $2, NULL),
		    ('todo-update-create-fail', 'list-1', $1, NULL, 'T3', '', NULL, 'normal', false, false, 3, $2, $2, NULL),
		    ('todo-update-success', 'list-1', $1, 'g-todo-update-success', 'T4', '', NULL, 'normal', false, false, 4, $2, $2, NULL),
		    ('todo-update-fallback', 'list-1', $1, 'g-todo-update-fallback', 'T5', '', NULL, 'normal', false, false, 5, $2, $2, NULL)`,
		userID, now, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}

	t.Run("calendar_delete_branches", func(t *testing.T) {
		processor := NewGooglePushProcessor(db, &googleAuthStub{
			deleteCalendarEventFn: func(context.Context, portout.OAuthToken, string, string) error {
				return errors.New("delete failed")
			},
		})

		if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "delete", LocalResourceID: "evt-delete-nil", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("calendar delete with nil google id should no-op: %v", err)
		}

		err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "delete", LocalResourceID: "evt-delete-err", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		})
		if err == nil {
			t.Fatal("expected calendar delete error")
		}
	})

	t.Run("calendar_create_update_branches", func(t *testing.T) {
		processor := NewGooglePushProcessor(db, &googleAuthStub{
			updateCalendarEventFn: func(_ context.Context, _ portout.OAuthToken, _ string, eventID string, _ portout.CalendarEventUpsertInput) (*string, error) {
				switch eventID {
				case "g-evt-create-fail", "g-evt-update-fallback":
					return nil, &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
				case "g-evt-update-success":
					etag := "etag-updated"
					return &etag, nil
				default:
					return nil, errors.New("unexpected update")
				}
			},
			createCalendarEventFn: func(_ context.Context, _ portout.OAuthToken, _ string, input portout.CalendarEventUpsertInput) (string, *string, error) {
				switch input.Title {
				case "E2", "E3":
					return "", nil, errors.New("create failed")
				case "E5":
					etag := "etag-created"
					return "g-evt-new", &etag, nil
				default:
					etag := "etag-default"
					return "g-evt-default", &etag, nil
				}
			},
		})

		if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "create", LocalResourceID: "evt-create-fail", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err == nil {
			t.Fatal("expected calendar create fallback error")
		}

		if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-update-create-fail", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err == nil {
			t.Fatal("expected calendar update create error")
		}

		if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-update-success", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("expected calendar update success path: %v", err)
		}

		if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-update-fallback", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("expected calendar update fallback success: %v", err)
		}
	})

	t.Run("todo_delete_and_update_branches", func(t *testing.T) {
		processor := NewGooglePushProcessor(db, &googleAuthStub{
			deleteTaskFn: func(context.Context, portout.OAuthToken, string, string) error {
				return errors.New("delete failed")
			},
			updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, taskID string, _ portout.TodoUpsertInput) error {
				switch taskID {
				case "g-todo-create-fail", "g-todo-update-fallback":
					return &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
				case "g-todo-update-success":
					return nil
				default:
					return errors.New("unexpected update")
				}
			},
			createTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, input portout.TodoUpsertInput) (string, error) {
				switch input.Title {
				case "T2", "T3":
					return "", errors.New("create failed")
				case "T5":
					return "g-todo-new", nil
				default:
					return "g-todo-default", nil
				}
			},
		})

		if err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "delete", LocalResourceID: "todo-delete-nil", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("todo delete with nil google id should no-op: %v", err)
		}

		err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "delete", LocalResourceID: "todo-delete-err", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		})
		if err == nil {
			t.Fatal("expected todo delete error")
		}

		if err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "create", LocalResourceID: "todo-create-fail", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err == nil {
			t.Fatal("expected todo create fallback error")
		}

		if err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-update-create-fail", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err == nil {
			t.Fatal("expected todo update create error")
		}

		if err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-update-success", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("expected todo update success path: %v", err)
		}

		if err := processor.processTodoPush(ctx, token, pushOutboxItem{
			AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-update-fallback", ExpectedUpdatedAt: now.Add(2 * time.Minute),
		}); err != nil {
			t.Fatalf("expected todo update fallback success: %v", err)
		}
	})
}

func TestGoogleSyncCoordinatorClosedPoolAndSuccessErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("closed_pool_helpers", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		coordinator := NewGoogleSyncCoordinator(db, syncerFuncStub{})
		db.Close()

		if _, err := coordinator.TriggerUserSync(ctx, "user", "both", "manual"); err == nil {
			t.Fatal("expected TriggerUserSync error on closed pool")
		}
		if _, err := coordinator.RunHourlySync(ctx); err == nil {
			t.Fatal("expected RunHourlySync error on closed pool")
		}
		if _, err := coordinator.listActiveAccountsByUser(ctx, "user"); err == nil {
			t.Fatal("expected listActiveAccountsByUser error on closed pool")
		}
		if _, err := coordinator.listActiveAccounts(ctx); err == nil {
			t.Fatal("expected listActiveAccounts error on closed pool")
		}
		if _, err := coordinator.getSettingBool(ctx, "user", "key", true); err == nil {
			t.Fatal("expected getSettingBool error on closed pool")
		}
		if _, err := coordinator.lastSyncAt(ctx, "account", "background"); err == nil {
			t.Fatal("expected lastSyncAt error on closed pool")
		}
		if _, err := coordinator.syncAccountIfDue(ctx, "user", &authdomain.GoogleAccount{ID: "account"}, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil {
			t.Fatal("expected syncAccountIfDue acquire error on closed pool")
		}
	})

}

func TestGoogleSyncerDirectErrorAndPaginationBranches(t *testing.T) {
	ctx := context.Background()
	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	t.Run("sync_calendars_update_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main"}}, nil
			},
		})
		db.Close()

		err := syncer.syncCalendarsAndEvents(ctx, userID, accountID, token, now)
		if err == nil {
			t.Fatal("expected syncCalendarsAndEvents update error on closed pool")
		}
	})

		t.Run("sync_calendars_list_events_error", func(t *testing.T) {
			db := dbtest.Open(t)
			dbtest.Reset(t, db)
			_, err := db.Exec(ctx,
				`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
				 VALUES ('cal-1', $1, 'g-cal-1', $2, 'Main', 'google', true, true, false, false, $3, $3)`,
				userID, accountID, now,
		)
		if err != nil {
			t.Fatalf("insert calendar: %v", err)
		}

		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main"}}, nil
			},
			listCalendarEventsFn: func(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				return nil, errors.New("list events failed")
			},
		})

			err = syncer.syncCalendarsAndEvents(ctx, userID, accountID, token, now)
			if err == nil {
				t.Fatal("expected syncCalendarsAndEvents list events error")
			}
	})

	t.Run("sync_calendars_pagination", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		_, err := db.Exec(ctx,
			`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
			 VALUES ('cal-1', $1, 'g-cal-1', $2, 'Main', 'google', true, true, false, false, $3, $3)`,
			userID, accountID, now,
		)
		if err != nil {
			t.Fatalf("insert calendar: %v", err)
		}

		pages := 0
		start := now.Add(time.Hour)
		end := start.Add(time.Hour)
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main"}}, nil
			},
			listCalendarEventsFn: func(_ context.Context, _ portout.OAuthToken, _ string, pageToken string, _ string, _ *time.Time, _ *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				pages++
				if pageToken == "" {
					return &portout.OAuthCalendarEventsPage{
						Events: []portout.OAuthCalendarEvent{{GoogleID: "ge-1", Status: "confirmed", Title: "One", StartTime: &start, EndTime: &end}},
						NextPageToken: "p2",
					}, nil
				}
				return &portout.OAuthCalendarEventsPage{
					Events: []portout.OAuthCalendarEvent{{GoogleID: "ge-2", Status: "confirmed", Title: "Two", StartTime: &start, EndTime: &end}},
					NextSyncToken: "sync-2",
				}, nil
			},
		})

		if err := syncer.syncCalendarsAndEvents(ctx, userID, accountID, token, now); err != nil {
			t.Fatalf("syncCalendarsAndEvents pagination: %v", err)
		}
		if pages != 2 {
			t.Fatalf("expected 2 pages, got %d", pages)
		}
	})

	t.Run("backfill_query_error_on_closed_pool", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{})
		db.Close()

		_, err := syncer.BackfillEvents(ctx, userID, now, now.Add(time.Hour), nil)
		if err == nil {
			t.Fatal("expected BackfillEvents query error on closed pool")
		}
	})

	t.Run("backfill_lock_busy_and_list_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		seedBackfillAccountAndCalendar(t, db, userID, accountID, now)
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarEventsFn: func(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				return nil, errors.New("backfill list failed")
			},
		})

		lockKey := advisoryLockKey(accountID + ":" + strconv.FormatInt(now.Unix(), 10) + ":" + strconv.FormatInt(now.Add(time.Hour).Unix(), 10))
		lockConn, err := db.Acquire(ctx)
		if err != nil {
			t.Fatalf("acquire lock conn: %v", err)
		}
		defer lockConn.Release()
		if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
			t.Fatalf("acquire lock: %v", err)
		}

		result, err := syncer.BackfillEvents(ctx, userID, now, now.Add(time.Hour), []string{"cal-1"})
		if err != nil {
			t.Fatalf("BackfillEvents busy lock should skip, got %v", err)
		}
		if result == nil {
			t.Fatal("expected backfill result when lock is busy")
		}

		if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey); err != nil {
			t.Fatalf("unlock advisory lock: %v", err)
		}

		if _, err := syncer.BackfillEvents(ctx, userID, now, now.Add(time.Hour), []string{"cal-1"}); err == nil {
			t.Fatal("expected BackfillEvents list error")
		}
	})

	t.Run("sync_tasks_empty_seen_and_update_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		_, err := db.Exec(ctx,
			`INSERT INTO user_google_accounts
			    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
			 VALUES ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
			accountID, userID, now,
		)
		if err != nil {
			t.Fatalf("insert account: %v", err)
		}
			_, err = db.Exec(ctx,
				`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
				 VALUES ('list-1', $1, 'g-list-1', $2, 'Main', 0, $3, $3)`,
				userID, accountID, now,
		)
		if err != nil {
			t.Fatalf("insert todo list: %v", err)
		}
		_, err = db.Exec(ctx,
			`INSERT INTO todos (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
			 VALUES ('todo-1', 'list-1', $1, 'gt-1', 'Todo', '', NULL, 'normal', false, false, 0, $2, $2, NULL)`,
			userID, now,
		)
		if err != nil {
			t.Fatalf("insert todo: %v", err)
		}

		pageCalls := 0
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "Main"}}, nil
			},
			listTasksFn: func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
				pageCalls++
				if pageCalls == 1 {
					return &portout.OAuthTasksPage{NextPageToken: "p2"}, nil
				}
				db.Close()
				due := now.Format("2006-01-02")
				return &portout.OAuthTasksPage{Items: []portout.OAuthTask{{GoogleID: "gt-1", Title: "Todo", DueDate: &due}}}, nil
			},
		})

		err = syncer.syncTaskListsAndTodos(ctx, userID, accountID, token, now)
		if err == nil {
			t.Fatal("expected syncTaskListsAndTodos update error after db close")
		}
	})
}

type syncerFuncStub struct {
	syncFn func(context.Context, string, *authdomain.GoogleAccount, portout.GoogleSyncOptions) error
}

func (s syncerFuncStub) SyncAccount(ctx context.Context, userID string, account *authdomain.GoogleAccount, options portout.GoogleSyncOptions) error {
	if s.syncFn != nil {
		return s.syncFn(ctx, userID, account, options)
	}
	return nil
}

func seedBackfillAccountAndCalendar(t *testing.T, db *pgxpool.Pool, userID, accountID string, now time.Time) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	); err != nil {
		t.Fatalf("insert account: %v", err)
	}
	if _, err := db.Exec(ctx,
		`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'g-cal-1', $2, 'Main', 'google', true, true, false, false, $3, $3)`,
		userID, accountID, now,
	); err != nil {
		t.Fatalf("insert calendar: %v", err)
	}
}

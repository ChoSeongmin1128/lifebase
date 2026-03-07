package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

func TestGoogleAccountSyncerSyncAccountIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	const existingCalendarID = "cal-local-1"
	const existingListID = "list-local-1"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert google account: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO calendars
		    (id, user_id, google_id, google_account_id, name, kind, color_id, is_primary, is_visible, is_readonly, is_special, sync_token, created_at, updated_at)
		 VALUES
		    ($1, $2, 'g-cal-1', $3, 'Old Name', 'custom', '1', false, true, false, false, 'expired-sync', $4, $4),
		    ('cal-special', $2, 'g-special', $3, 'Holiday', 'holiday', '2', false, true, false, true, NULL, $4, $4)`,
		existingCalendarID, userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert calendars: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO events
		    (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, color_id, recurrence_rule, etag, created_at, updated_at, deleted_at)
		 VALUES
		    ('evt-existing', $1, $2, 'ge-1', 'Old Event', 'old', 'old', $3, $4, 'Asia/Seoul', false, NULL, NULL, NULL, $3, $3, NULL),
		    ('evt-cancel', $1, $2, 'ge-cancel', 'Cancel Me', '', '', $3, $4, 'Asia/Seoul', false, NULL, NULL, NULL, $3, $3, NULL)`,
		existingCalendarID, userID, now, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert existing events: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists
		    (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ($1, $2, 'g-list-1', $3, 'Old List', 9, $4, $4)`,
		existingListID, userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos
		    (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, done_at, created_at, updated_at, deleted_at)
		 VALUES
		    ('todo-existing', $1, $2, 'gt-1', 'Old Todo', 'old', NULL, 'normal', false, false, 5, NULL, $3, $3, NULL),
		    ('todo-delete', $1, $2, 'gt-del', 'Delete Todo', '', NULL, 'normal', false, false, 6, NULL, $3, $3, NULL),
		    ('todo-done-old', $1, $2, 'gt-old-done', 'Old Done', '', NULL, 'normal', true, false, 7, $4, $3, $3, NULL)`,
		existingListID, userID, now, now.AddDate(-2, 0, 0),
	)
	if err != nil {
		t.Fatalf("insert existing todos: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, 'todo_done_retention_period', '1y', $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert retention setting: %v", err)
	}

	var calendarEventsCalls int
	google := &googleAuthStub{
		listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
			return []portout.OAuthCalendar{
				{
					GoogleID:   "g-cal-1",
					Name:       "Calendar One",
					Kind:       "custom",
					IsPrimary:  true,
					IsVisible:  true,
					IsReadOnly: false,
					IsSpecial:  false,
				},
				{
					GoogleID:   "g-cal-2",
					Name:       "Calendar Two",
					Kind:       "custom",
					IsPrimary:  false,
					IsVisible:  true,
					IsReadOnly: false,
					IsSpecial:  false,
				},
				{GoogleID: "g-special", Name: "Special", Kind: "holiday", IsSpecial: true},
			}, nil
		},
		listCalendarEventsFn: func(_ context.Context, _ portout.OAuthToken, calendarID, _ string, syncToken string, _ *time.Time, _ *time.Time) (*portout.OAuthCalendarEventsPage, error) {
			if calendarID == "g-cal-1" && syncToken == "expired-sync" {
				return nil, &portout.GoogleAPIError{StatusCode: 410, Reason: "fullSyncRequired", Message: "expired"}
			}
			calendarEventsCalls++
			start1 := now.Add(24 * time.Hour)
			end1 := start1.Add(2 * time.Hour)
			start2 := now.Add(48 * time.Hour)
			end2 := start2.Add(time.Hour)
			var events []portout.OAuthCalendarEvent
			if calendarID == "g-cal-1" {
				events = []portout.OAuthCalendarEvent{
					{
						GoogleID:    "ge-1",
						Status:      "confirmed",
						Title:       "Updated Event",
						Description: "updated",
						Location:    "room",
						StartTime:   &start1,
						EndTime:     &end1,
						Timezone:    "Asia/Seoul",
					},
					{
						GoogleID:  "ge-cancel",
						Status:    "cancelled",
						StartTime: &start1,
						EndTime:   &end1,
					},
					{
						GoogleID:    "ge-new",
						Status:      "confirmed",
						Title:       "New Event",
						Description: "new",
						Location:    "room2",
						StartTime:   &start2,
						EndTime:     &end2,
						Timezone:    "Asia/Seoul",
					},
				}
			} else {
				events = []portout.OAuthCalendarEvent{
					{
						GoogleID:    "ge2-new",
						Status:      "confirmed",
						Title:       "New Event 2",
						Description: "new2",
						Location:    "room3",
						StartTime:   &start2,
						EndTime:     &end2,
						Timezone:    "Asia/Seoul",
					},
				}
			}
			return &portout.OAuthCalendarEventsPage{
				Events:        events,
				NextPageToken: "",
				NextSyncToken: "sync-new",
			}, nil
		},
		listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
			return []portout.OAuthTaskList{
				{GoogleID: "g-list-1", Name: "Main"},
				{GoogleID: "g-list-2", Name: "Secondary"},
			}, nil
		},
		listTasksFn: func(_ context.Context, _ portout.OAuthToken, taskListID, _ string) (*portout.OAuthTasksPage, error) {
			switch taskListID {
			case "g-list-1":
				due := now.AddDate(0, 0, 1).Format("2006-01-02")
				return &portout.OAuthTasksPage{
					Items: []portout.OAuthTask{
						{GoogleID: "gt-1", Title: "Updated Todo", Notes: "new", DueDate: &due, IsDone: false},
						{GoogleID: "gt-del", IsDeleted: true},
					},
				}, nil
			case "g-list-2":
				due := now.AddDate(0, 0, 3).Format("2006-01-02")
				completed := now.Add(-time.Hour)
				return &portout.OAuthTasksPage{
					Items: []portout.OAuthTask{
						{GoogleID: "gt-new", Title: "New Todo", Notes: "n", DueDate: &due, IsDone: true, CompletedAt: &completed},
					},
				}, nil
			default:
				return &portout.OAuthTasksPage{}, nil
			}
		},
	}

	syncer := NewGoogleAccountSyncer(db, google)

	if _, err := NewGoogleAccountSyncer(db, google).BackfillEvents(ctx, userID, now, now, nil); err == nil {
		t.Fatal("expected invalid backfill range error")
	}
	if err := syncer.SyncAccount(ctx, userID, nil, portout.GoogleSyncOptions{SyncCalendar: true}); err == nil {
		t.Fatal("expected nil account error")
	}

	account := &authdomain.GoogleAccount{
		ID:             accountID,
		UserID:         userID,
		AccessToken:    "at",
		RefreshToken:   "rt",
		TokenExpiresAt: timePtr(now.Add(time.Hour)),
	}
	if err := syncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true, SyncTodo: true}); err != nil {
		t.Fatalf("SyncAccount: %v", err)
	}
	if calendarEventsCalls == 0 {
		t.Fatal("expected listCalendarEvents to be called")
	}

	var syncToken *string
	if err := db.QueryRow(ctx, `SELECT sync_token FROM calendars WHERE user_id = $1 AND google_id = 'g-cal-1'`, userID).Scan(&syncToken); err != nil {
		t.Fatalf("read sync token: %v", err)
	}
	if syncToken == nil || *syncToken != "sync-new" {
		t.Fatalf("expected refreshed sync token, got %#v", syncToken)
	}

	var deletedAt *time.Time
	if err := db.QueryRow(ctx, `SELECT deleted_at FROM events WHERE user_id = $1 AND google_id = 'ge-cancel'`, userID).Scan(&deletedAt); err != nil {
		t.Fatalf("read cancelled event deleted_at: %v", err)
	}
	if deletedAt == nil {
		t.Fatal("expected cancelled event to be soft deleted")
	}

	var updatedTitle string
	if err := db.QueryRow(ctx, `SELECT title FROM events WHERE user_id = $1 AND google_id = 'ge-1'`, userID).Scan(&updatedTitle); err != nil {
		t.Fatalf("read updated event title: %v", err)
	}
	if updatedTitle != "Updated Event" {
		t.Fatalf("expected updated event title, got %s", updatedTitle)
	}

	var newTodoTitle string
	if err := db.QueryRow(ctx, `SELECT title FROM todos WHERE user_id = $1 AND google_id = 'gt-new'`, userID).Scan(&newTodoTitle); err != nil {
		t.Fatalf("read inserted todo: %v", err)
	}
	if newTodoTitle != "New Todo" {
		t.Fatalf("expected inserted todo title, got %s", newTodoTitle)
	}

	if err := db.QueryRow(ctx, `SELECT deleted_at FROM todos WHERE id = 'todo-delete'`).Scan(&deletedAt); err != nil {
		t.Fatalf("read deleted todo: %v", err)
	}
	if deletedAt == nil {
		t.Fatal("expected deleted google task to be soft deleted locally")
	}

	if err := db.QueryRow(ctx, `SELECT deleted_at FROM todos WHERE id = 'todo-done-old'`).Scan(&deletedAt); err != nil {
		t.Fatalf("read old done todo: %v", err)
	}
	if deletedAt == nil {
		t.Fatal("expected old done todo to be deleted by retention")
	}

	start := now.AddDate(0, 0, -3)
	end := now.AddDate(0, 0, 3)
	backfill, err := syncer.BackfillEvents(ctx, userID, start, end, []string{"", existingCalendarID})
	if err != nil {
		t.Fatalf("BackfillEvents: %v", err)
	}
	if backfill.FetchedEvents == 0 {
		t.Fatalf("expected backfill fetched events, got %#v", backfill)
	}
	if !backfill.CoveredStart.Equal(start) || !backfill.CoveredEnd.Equal(end) {
		t.Fatalf("backfill covered range mismatch: %#v", backfill)
	}

	// resolveTodoDoneRetentionCutoff fallback/unlimited branches
	_, err = db.Exec(ctx,
		`UPDATE user_settings SET value = 'unlimited' WHERE user_id = $1 AND key = 'todo_done_retention_period'`,
		userID,
	)
	if err != nil {
		t.Fatalf("update retention to unlimited: %v", err)
	}
	if cutoff := syncer.resolveTodoDoneRetentionCutoff(ctx, userID, now); cutoff != nil {
		t.Fatalf("expected nil cutoff for unlimited, got %v", cutoff)
	}
	_, err = db.Exec(ctx,
		`UPDATE user_settings SET value = 'invalid' WHERE user_id = $1 AND key = 'todo_done_retention_period'`,
		userID,
	)
	if err != nil {
		t.Fatalf("update retention to invalid: %v", err)
	}
	cutoff := syncer.resolveTodoDoneRetentionCutoff(ctx, userID, now)
	if cutoff == nil || !strings.HasPrefix(cutoff.Format("2006"), now.AddDate(-1, 0, 0).Format("2006")) {
		t.Fatalf("expected default yearly cutoff, got %v", cutoff)
	}
}

func TestGoogleAccountSyncerResolveTodoDoneRetentionCutoffSwitchCasesIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	const userID = "11111111-1111-1111-1111-111111111111"

	syncer := NewGoogleAccountSyncer(db, &googleAuthStub{})

	_, err := db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, 'todo_done_retention_period', '1y', $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert retention setting: %v", err)
	}

	cases := []struct {
		name      string
		raw       string
		expectNil bool
		expect    time.Time
	}{
		{name: "one_month", raw: "1m", expect: now.AddDate(0, -1, 0)},
		{name: "three_months_trimmed", raw: " 3M ", expect: now.AddDate(0, -3, 0)},
		{name: "six_months", raw: "6m", expect: now.AddDate(0, -6, 0)},
		{name: "one_year", raw: "1y", expect: now.AddDate(-1, 0, 0)},
		{name: "three_years", raw: "3y", expect: now.AddDate(-3, 0, 0)},
		{name: "unlimited", raw: "unlimited", expectNil: true},
		{name: "default_fallback", raw: "weird", expect: now.AddDate(-1, 0, 0)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(ctx,
				`UPDATE user_settings SET value = $2, updated_at = $3
				 WHERE user_id = $1 AND key = 'todo_done_retention_period'`,
				userID, tc.raw, now,
			)
			if err != nil {
				t.Fatalf("update retention setting: %v", err)
			}
			cutoff := syncer.resolveTodoDoneRetentionCutoff(ctx, userID, now)
			if tc.expectNil {
				if cutoff != nil {
					t.Fatalf("expected nil cutoff, got %v", cutoff)
				}
				return
			}
			if cutoff == nil || !cutoff.Equal(tc.expect) {
				t.Fatalf("unexpected cutoff: got=%v expect=%v", cutoff, tc.expect)
			}
		})
	}
}

func TestGoogleAccountSyncerResolveTodoDoneRetentionCutoffFallsBackOnDBError(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	const userID = "11111111-1111-1111-1111-111111111111"

	syncer := NewGoogleAccountSyncer(db, &googleAuthStub{})
	db.Close()

	cutoff := syncer.resolveTodoDoneRetentionCutoff(ctx, userID, now)
	expected := now.AddDate(-1, 0, 0)
	if cutoff == nil || !cutoff.Equal(expected) {
		t.Fatalf("expected default yearly cutoff on db error, got %v", cutoff)
	}
}

func TestGoogleAccountSyncerApplyOAuthEventErrorBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"

	syncer := NewGoogleAccountSyncer(db, &googleAuthStub{})
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)

	// Closed pool triggers update exec error path.
	db.Close()
	if _, _, err := syncer.applyOAuthEvent(ctx, userID, "cal-1", portout.OAuthCalendarEvent{
		GoogleID:   "ge-1",
		Status:     "confirmed",
		Title:      "T",
		StartTime:  &start,
		EndTime:    &end,
		Timezone:   "Asia/Seoul",
		IsAllDay:   false,
	}, now); err == nil || !strings.Contains(err.Error(), "update event") {
		t.Fatalf("expected wrapped update event error, got %v", err)
	}
}

func TestGoogleAccountSyncerSyncAccountPropagatesCalendarAndTodoErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	account := &authdomain.GoogleAccount{
		ID:             "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:         userID,
		AccessToken:    "at",
		RefreshToken:   "rt",
		TokenExpiresAt: timePtr(now.Add(time.Hour)),
	}

	t.Run("calendar_list_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)

		want := errors.New("calendar api down")
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return nil, want
			},
		})

		err := syncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true})
		if err == nil || !strings.Contains(err.Error(), "list google calendars") || !errors.Is(err, want) {
			t.Fatalf("expected wrapped calendar list error, got %v", err)
		}
	})

	t.Run("task_list_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)

		want := errors.New("task list api down")
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return nil, want
			},
		})

		err := syncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{SyncTodo: true})
		if err == nil || !strings.Contains(err.Error(), "list google task lists") || !errors.Is(err, want) {
			t.Fatalf("expected wrapped task-list error, got %v", err)
		}
	})

	t.Run("task_page_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)

		want := errors.New("tasks api down")
		syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "Main"}}, nil
			},
			listTasksFn: func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
				return nil, want
			},
		})

		err := syncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{SyncTodo: true})
		if err == nil || !strings.Contains(err.Error(), "list google tasks") || !errors.Is(err, want) {
			t.Fatalf("expected wrapped tasks-page error, got %v", err)
		}
	})
}

func TestGoogleAccountSyncerSyncAccountTodoListDBError(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"

	account := &authdomain.GoogleAccount{
		ID:             "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:         userID,
		AccessToken:    "at",
		RefreshToken:   "rt",
		TokenExpiresAt: timePtr(now.Add(time.Hour)),
	}
	syncer := NewGoogleAccountSyncer(db, &googleAuthStub{
		listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
			return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "Main"}}, nil
		},
	})
	db.Close()

	err := syncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{SyncTodo: true})
	if err == nil || !strings.Contains(err.Error(), "update todo list") {
		t.Fatalf("expected todo list db update error, got %v", err)
	}
}

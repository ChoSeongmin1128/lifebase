package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

type moveTaskCall struct {
	taskID         string
	parentTaskID   *string
	previousTaskID *string
}

type googleAuthStub struct {
	listCalendarsFn       func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error)
	listTaskListsFn       func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error)
	listCalendarEventsFn  func(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error)
	listTasksFn           func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error)
	createCalendarEventFn func(context.Context, portout.OAuthToken, string, portout.CalendarEventUpsertInput) (string, *string, error)
	updateCalendarEventFn func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error)
	deleteCalendarEventFn func(context.Context, portout.OAuthToken, string, string) error
	createTaskFn          func(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error)
	updateTaskFn          func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error
	moveTaskFn            func(context.Context, portout.OAuthToken, string, string, *string, *string) error
	deleteTaskFn          func(context.Context, portout.OAuthToken, string, string) error
}

func (s *googleAuthStub) AuthURL(string) string               { return "" }
func (s *googleAuthStub) AuthURLForApp(string, string) string { return "" }
func (s *googleAuthStub) ExchangeCode(context.Context, string) (*portout.OAuthToken, error) {
	return nil, nil
}
func (s *googleAuthStub) ExchangeCodeForApp(context.Context, string, string) (*portout.OAuthToken, error) {
	return nil, nil
}
func (s *googleAuthStub) FetchUserInfo(context.Context, portout.OAuthToken) (*portout.OAuthUserInfo, error) {
	return nil, nil
}
func (s *googleAuthStub) ListCalendars(ctx context.Context, token portout.OAuthToken) ([]portout.OAuthCalendar, error) {
	if s.listCalendarsFn != nil {
		return s.listCalendarsFn(ctx, token)
	}
	return nil, nil
}
func (s *googleAuthStub) ListTaskLists(ctx context.Context, token portout.OAuthToken) ([]portout.OAuthTaskList, error) {
	if s.listTaskListsFn != nil {
		return s.listTaskListsFn(ctx, token)
	}
	return nil, nil
}
func (s *googleAuthStub) ListCalendarEvents(
	ctx context.Context,
	token portout.OAuthToken,
	calendarID,
	pageToken,
	syncToken string,
	timeMin,
	timeMax *time.Time,
) (*portout.OAuthCalendarEventsPage, error) {
	if s.listCalendarEventsFn != nil {
		return s.listCalendarEventsFn(ctx, token, calendarID, pageToken, syncToken, timeMin, timeMax)
	}
	return nil, nil
}
func (s *googleAuthStub) ListTasks(ctx context.Context, token portout.OAuthToken, taskListID, pageToken string) (*portout.OAuthTasksPage, error) {
	if s.listTasksFn != nil {
		return s.listTasksFn(ctx, token, taskListID, pageToken)
	}
	return nil, nil
}
func (s *googleAuthStub) CreateCalendarEvent(ctx context.Context, token portout.OAuthToken, calendarID string, input portout.CalendarEventUpsertInput) (string, *string, error) {
	if s.createCalendarEventFn != nil {
		return s.createCalendarEventFn(ctx, token, calendarID, input)
	}
	return "g-created", nil, nil
}
func (s *googleAuthStub) UpdateCalendarEvent(ctx context.Context, token portout.OAuthToken, calendarID, eventID string, input portout.CalendarEventUpsertInput) (*string, error) {
	if s.updateCalendarEventFn != nil {
		return s.updateCalendarEventFn(ctx, token, calendarID, eventID, input)
	}
	return nil, nil
}
func (s *googleAuthStub) DeleteCalendarEvent(ctx context.Context, token portout.OAuthToken, calendarID, eventID string) error {
	if s.deleteCalendarEventFn != nil {
		return s.deleteCalendarEventFn(ctx, token, calendarID, eventID)
	}
	return nil
}
func (s *googleAuthStub) CreateTaskList(context.Context, portout.OAuthToken, string) (string, error) {
	return "", nil
}
func (s *googleAuthStub) DeleteTaskList(context.Context, portout.OAuthToken, string) error {
	return nil
}
func (s *googleAuthStub) CreateTask(ctx context.Context, token portout.OAuthToken, taskListID string, input portout.TodoUpsertInput) (string, error) {
	if s.createTaskFn != nil {
		return s.createTaskFn(ctx, token, taskListID, input)
	}
	return "t-created", nil
}
func (s *googleAuthStub) UpdateTask(ctx context.Context, token portout.OAuthToken, taskListID, taskID string, input portout.TodoUpsertInput) error {
	if s.updateTaskFn != nil {
		return s.updateTaskFn(ctx, token, taskListID, taskID, input)
	}
	return nil
}
func (s *googleAuthStub) MoveTask(ctx context.Context, token portout.OAuthToken, taskListID, taskID string, parentTaskID, previousTaskID *string) error {
	if s.moveTaskFn != nil {
		return s.moveTaskFn(ctx, token, taskListID, taskID, parentTaskID, previousTaskID)
	}
	return nil
}
func (s *googleAuthStub) DeleteTask(ctx context.Context, token portout.OAuthToken, taskListID, taskID string) error {
	if s.deleteTaskFn != nil {
		return s.deleteTaskFn(ctx, token, taskListID, taskID)
	}
	return nil
}

func TestGooglePushProcessorProcessPendingIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	seedPushProcessorFixtures(t, db, userID, accountID, now)

	var moveCalls []moveTaskCall
	processor := NewGooglePushProcessor(db, &googleAuthStub{
		createCalendarEventFn: func(context.Context, portout.OAuthToken, string, portout.CalendarEventUpsertInput) (string, *string, error) {
			etag := "etag-created"
			return "g-evt-created", &etag, nil
		},
		createTaskFn: func(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error) {
			return "g-task-created", nil
		},
		moveTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, taskID string, parentTaskID, previousTaskID *string) error {
			moveCalls = append(moveCalls, moveTaskCall{
				taskID:         taskID,
				parentTaskID:   parentTaskID,
				previousTaskID: previousTaskID,
			})
			return nil
		},
	})

	processed, err := processor.ProcessPending(ctx, 20)
	if err != nil {
		t.Fatalf("ProcessPending: %v", err)
	}
	if processed != 8 {
		t.Fatalf("expected processed=8, got %d", processed)
	}

	var doneCount, retryCount, deadCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox WHERE status = 'done'`).Scan(&doneCount); err != nil {
		t.Fatalf("count done rows: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox WHERE status = 'retry'`).Scan(&retryCount); err != nil {
		t.Fatalf("count retry rows: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox WHERE status = 'dead'`).Scan(&deadCount); err != nil {
		t.Fatalf("count dead rows: %v", err)
	}
	if doneCount != 6 || retryCount != 1 || deadCount != 1 {
		t.Fatalf("unexpected status counts: done=%d retry=%d dead=%d", doneCount, retryCount, deadCount)
	}

	var eventGoogleID *string
	if err := db.QueryRow(ctx, `SELECT google_id FROM events WHERE id = 'evt-create'`).Scan(&eventGoogleID); err != nil {
		t.Fatalf("read event google_id: %v", err)
	}
	if eventGoogleID == nil || *eventGoogleID != "g-evt-created" {
		t.Fatalf("expected event google_id set, got %#v", eventGoogleID)
	}

	var todoGoogleID *string
	if err := db.QueryRow(ctx, `SELECT google_id FROM todos WHERE id = 'todo-create'`).Scan(&todoGoogleID); err != nil {
		t.Fatalf("read todo google_id: %v", err)
	}
	if todoGoogleID == nil || *todoGoogleID != "g-task-created" {
		t.Fatalf("expected todo google_id set, got %#v", todoGoogleID)
	}
	if len(moveCalls) < 2 {
		t.Fatalf("expected move calls for todo create/update, got %#v", moveCalls)
	}
	assertMoveCall(t, moveCalls, "g-task-created", strPtr("g-parent"), nil)
	assertMoveCall(t, moveCalls, "g-todo-update", nil, strPtr("g-prev"))
}

func TestGooglePushProcessorAuthErrorMarksAccountReauth(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	seedPushProcessorFixtures(t, db, userID, accountID, now)

	processor := NewGooglePushProcessor(db, &googleAuthStub{
		updateTaskFn: func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
			return &portout.GoogleAPIError{StatusCode: 401, Reason: "authError", Message: "expired"}
		},
	})
	processed, err := processor.ProcessPending(ctx, 20)
	if err != nil {
		t.Fatalf("ProcessPending auth error case: %v", err)
	}
	if processed != 8 {
		t.Fatalf("expected processed=8, got %d", processed)
	}

	var status string
	if err := db.QueryRow(ctx, `SELECT status FROM user_google_accounts WHERE id = $1`, accountID).Scan(&status); err != nil {
		t.Fatalf("read account status: %v", err)
	}
	if status != "reauth_required" {
		t.Fatalf("expected reauth_required after auth error, got %s", status)
	}
}

func TestGooglePushProcessorNilGoogleAuth(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	processor := NewGooglePushProcessor(db, nil)
	processed, err := processor.ProcessPending(context.Background(), 0)
	if err != nil || processed != 0 {
		t.Fatalf("nil google auth should no-op: processed=%d err=%v", processed, err)
	}
}

func seedPushProcessorFixtures(t *testing.T, db *pgxpool.Pool, userID, accountID string, now time.Time) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert google account: %v", err)
	}
	_, err = db.Exec(context.Background(),
		`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'g-cal-1', $2, 'Primary', 'google', true, true, false, false, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert calendar: %v", err)
	}
	_, err = db.Exec(context.Background(),
		`INSERT INTO events (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, created_at, updated_at, deleted_at)
		 VALUES
		    ('evt-create', 'cal-1', $1, NULL, 'Create', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-delete', 'cal-1', $1, 'g-evt-delete', 'Delete', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, $2)`,
		userID, now, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}
	_, err = db.Exec(context.Background(),
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1', $1, 'g-list-1', $2, 'L1', 0, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(context.Background(),
		`INSERT INTO todos (id, list_id, user_id, parent_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
		 VALUES
		    ('todo-parent', 'list-1', $1, NULL, 'g-parent', 'Parent Todo', '', NULL, 'normal', false, false, 0, $2, $2, NULL),
		    ('todo-prev', 'list-1', $1, NULL, 'g-prev', 'Prev Todo', '', NULL, 'normal', false, false, 1, $2, $2, NULL),
		    ('todo-create', 'list-1', $1, 'todo-parent', NULL, 'Create Todo', '', NULL, 'normal', false, false, 0, $2, $2, NULL),
		    ('todo-update', 'list-1', $1, NULL, 'g-todo-update', 'Update Todo', '', NULL, 'normal', false, false, 2, $2, $2, NULL),
		    ('todo-delete', 'list-1', $1, NULL, 'g-todo-delete', 'Delete Todo', '', NULL, 'normal', false, false, 3, $2, $2, $2),
		    ('todo-dead', 'list-1', $1, NULL, 'g-todo-dead', 'Dead Todo', '', NULL, 'normal', false, false, 4, $2, $2, NULL)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}
	_, err = db.Exec(context.Background(),
		`INSERT INTO google_push_outbox
		    (id, account_id, user_id, domain, op, local_resource_id, expected_updated_at, payload_json, status, attempt_count, next_retry_at, created_at, updated_at)
		 VALUES
		    ('out-1', $1, $2, 'calendar', 'create', 'evt-create', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-2', $1, $2, 'calendar', 'delete', 'evt-delete', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-3', $1, $2, 'todo', 'create', 'todo-create', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-4', $1, $2, 'todo', 'delete', 'todo-delete', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-5', $1, $2, 'todo', 'update', 'todo-update', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-6', 'deadbeef-dead-beef-dead-beefdeadbeef', $2, 'todo', 'update', 'todo-dead', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-7', $1, $2, 'unknown', 'create', 'evt-create', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-8', $1, $2, 'todo', 'update', 'missing-todo', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4)`,
		accountID, userID, now.Add(time.Minute), now,
	)
	if err != nil {
		t.Fatalf("insert outbox rows: %v", err)
	}
}

func assertMoveCall(t *testing.T, calls []moveTaskCall, taskID string, parentTaskID, previousTaskID *string) {
	t.Helper()
	for _, call := range calls {
		if call.taskID != taskID {
			continue
		}
		if !sameOptionalString(call.parentTaskID, parentTaskID) || !sameOptionalString(call.previousTaskID, previousTaskID) {
			t.Fatalf("unexpected move call for %s: %#v", taskID, call)
		}
		return
	}
	t.Fatalf("move call for %s not found: %#v", taskID, calls)
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func strPtr(value string) *string {
	return &value
}

package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	portout "lifebase/internal/auth/port/out"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func jsonResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
	}
}

func ctxWithTransport(fn roundTripFunc) context.Context {
	client := &http.Client{Transport: fn}
	return context.WithValue(context.Background(), oauth2.HTTPClient, client)
}

func TestParseGoogleEventDateTime_AllDayUsesInclusiveLocalEnd(t *testing.T) {
	start, end, isAllDay, err := parseGoogleEventDateTime(
		"2026-03-03",
		"",
		"2026-03-04",
		"",
		"Asia/Seoul",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isAllDay {
		t.Fatalf("expected all-day event")
	}

	loc, _ := time.LoadLocation("Asia/Seoul")
	if got := start.In(loc).Format("2006-01-02"); got != "2026-03-03" {
		t.Fatalf("unexpected start date: %s", got)
	}
	if got := end.In(loc).Format("2006-01-02"); got != "2026-03-03" {
		t.Fatalf("unexpected end date: %s", got)
	}
}

func TestBuildGoogleCalendarEventBody_AllDayUsesExclusiveEnd(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	input := portout.CalendarEventUpsertInput{
		Title:     "테스트",
		StartTime: time.Date(2026, 3, 3, 0, 0, 0, 0, loc),
		EndTime:   time.Date(2026, 3, 4, 23, 59, 59, 0, loc),
		Timezone:  "Asia/Seoul",
		IsAllDay:  true,
	}

	body := buildGoogleCalendarEventBody(input)
	startPayload, ok := body["start"].(map[string]string)
	if !ok {
		t.Fatalf("start payload type mismatch")
	}
	endPayload, ok := body["end"].(map[string]string)
	if !ok {
		t.Fatalf("end payload type mismatch")
	}

	if startPayload["date"] != "2026-03-03" {
		t.Fatalf("unexpected start date payload: %s", startPayload["date"])
	}
	if endPayload["date"] != "2026-03-05" {
		t.Fatalf("unexpected end date payload: %s", endPayload["date"])
	}
}

func TestOAuthClientHelpers(t *testing.T) {
	redirects := map[string]string{"web": "https://web.example.com/callback", "ios": "lifebase://oauth"}
	c := NewOAuthClient("cid", "secret", redirects)
	redirects["web"] = "https://mutated"
	if c.redirects["web"] != "https://web.example.com/callback" {
		t.Fatal("redirect map must be cloned")
	}

	if !strings.Contains(c.AuthURL("state1"), "state1") {
		t.Fatal("auth url should include state")
	}
	if !strings.Contains(c.AuthURLForApp("state2", "ios"), url.QueryEscape("lifebase://oauth")) {
		t.Fatal("auth url for app should include app redirect")
	}
	if cfg := c.oauthConfig("unknown"); cfg.RedirectURL != "https://web.example.com/callback" {
		t.Fatalf("unknown app should fallback to web redirect, got %q", cfg.RedirectURL)
	}

	now := time.Now().UTC().Truncate(time.Second)
	if parsed := parseOptionalRFC3339(now.Format(time.RFC3339)); parsed == nil || !parsed.Equal(now) {
		t.Fatalf("parseOptionalRFC3339 valid failed: %#v", parsed)
	}
	if parseOptionalRFC3339("bad") != nil {
		t.Fatal("invalid RFC3339 should return nil")
	}
	if parseOptionalRFC3339("") != nil {
		t.Fatal("empty should return nil")
	}

	if kind, ro, special := classifyGoogleCalendar("primary", "main", true, "owner"); kind != "primary" || ro || special {
		t.Fatalf("unexpected primary classification: %s %v %v", kind, ro, special)
	}
	if kind, _, special := classifyGoogleCalendar("ko.south_korea#holiday@group.v.calendar.google.com", "공휴일", false, "reader"); kind != "holiday" || !special {
		t.Fatalf("unexpected holiday classification: %s", kind)
	}
	if kind, _, special := classifyGoogleCalendar("#contacts@", "birthdays", false, "reader"); kind != "birthday" || !special {
		t.Fatalf("unexpected birthday classification: %s", kind)
	}
	if kind, ro, _ := classifyGoogleCalendar("x", "x", false, "reader"); kind != "subscribed" || !ro {
		t.Fatalf("unexpected subscribed classification: %s", kind)
	}
	if kind, ro, _ := classifyGoogleCalendar("x", "x", false, "owner"); kind != "custom" || ro {
		t.Fatalf("unexpected custom classification: %s", kind)
	}

	due := "2026-03-06"
	body := buildGoogleTaskBody(portout.TodoUpsertInput{Title: "t", Notes: "n", DueDate: &due, IsDone: true})
	if body["status"] != "completed" || body["due"] != "2026-03-06T00:00:00.000Z" {
		t.Fatalf("unexpected task body: %#v", body)
	}
	body = buildGoogleTaskBody(portout.TodoUpsertInput{Title: "t", Notes: "n"})
	if body["status"] != "needsAction" || body["due"] != nil {
		t.Fatalf("unexpected default task body: %#v", body)
	}

	errResp := jsonResp(http.StatusBadRequest, `{"error":{"code":400,"message":"bad","errors":[{"domain":"global","reason":"invalid"}]}}`)
	apiErr := parseGoogleAPIError(errResp, "test action")
	if gerr, ok := apiErr.(*portout.GoogleAPIError); !ok || gerr.Reason != "invalid" || !strings.Contains(gerr.Message, "bad") {
		t.Fatalf("unexpected api error parse: %#v", apiErr)
	}

	errRespFallback := jsonResp(http.StatusTeapot, `{}`)
	apiErr = parseGoogleAPIError(errRespFallback, "test action")
	if gerr, ok := apiErr.(*portout.GoogleAPIError); !ok || !strings.Contains(gerr.Message, "returned 418") {
		t.Fatalf("unexpected fallback error parse: %#v", apiErr)
	}
}

func TestOAuthClientAPIFlows(t *testing.T) {
	c := NewOAuthClient("cid", "secret", map[string]string{"web": "https://web.example.com/callback"})
	hits := map[string]int{}
	ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
		hits[req.URL.Path]++
		switch {
		case req.URL.Host == "oauth2.googleapis.com" && req.URL.Path == "/token":
			return jsonResp(http.StatusOK, `{"access_token":"at","refresh_token":"rt","expires_in":3600,"token_type":"Bearer"}`), nil
		case req.URL.Path == "/oauth2/v3/userinfo":
			return jsonResp(http.StatusOK, `{"sub":"gid","email":"u@example.com","name":"User","picture":"p"}`), nil
		case req.URL.Path == "/calendar/v3/users/me/calendarList":
			return jsonResp(http.StatusOK, `{"items":[{"id":"cal1","summary":"Main","primary":true,"selected":true,"accessRole":"owner"}]}`), nil
		case req.URL.Path == "/tasks/v1/users/@me/lists" && req.Method == http.MethodGet:
			return jsonResp(http.StatusOK, `{"items":[{"id":"list1","title":"Tasks"}]}`), nil
		case req.URL.Path == "/tasks/v1/users/@me/lists" && req.Method == http.MethodPost:
			return jsonResp(http.StatusOK, `{"id":"new-list"}`), nil
		case strings.HasPrefix(req.URL.Path, "/tasks/v1/users/@me/lists/") && req.Method == http.MethodDelete:
			return jsonResp(http.StatusNoContent, ``), nil
		case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events") && req.Method == http.MethodGet:
			return jsonResp(http.StatusOK, `{"items":[{"id":"evt1","status":"confirmed","summary":"S","start":{"dateTime":"2026-03-05T01:00:00Z"},"end":{"dateTime":"2026-03-05T02:00:00Z"}}]}`), nil
		case strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks") && req.Method == http.MethodGet:
			return jsonResp(http.StatusOK, `{"items":[{"id":"task1","parent":"parent-1","title":"T","status":"completed","completed":"2026-03-05T00:00:00Z","due":"2026-03-06T00:00:00.000Z"}]}`), nil
		case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events") && req.Method == http.MethodPost:
			return jsonResp(http.StatusCreated, `{"id":"evt-created","etag":"etag1"}`), nil
		case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events/evt1") && req.Method == http.MethodPatch:
			return jsonResp(http.StatusOK, `{"etag":"etag2"}`), nil
		case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events/evt1") && req.Method == http.MethodDelete:
			return jsonResp(http.StatusNoContent, ``), nil
		case strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1/move") && req.Method == http.MethodPost:
			if req.URL.Query().Get("parent") != "parent-1" || req.URL.Query().Get("previous") != "prev-1" {
				t.Fatalf("unexpected move query: %s", req.URL.RawQuery)
			}
			return jsonResp(http.StatusOK, `{}`), nil
		case strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks") && req.Method == http.MethodPost:
			return jsonResp(http.StatusOK, `{"id":"task-created"}`), nil
		case strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1") && req.Method == http.MethodPatch:
			return jsonResp(http.StatusOK, `{}`), nil
		case strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1") && req.Method == http.MethodDelete:
			return jsonResp(http.StatusNoContent, ``), nil
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})

	token, err := c.ExchangeCodeForApp(ctx, "code", "web")
	if err != nil || token.AccessToken == "" {
		t.Fatalf("exchange code failed: %v %#v", err, token)
	}
	if _, err := c.ExchangeCode(ctx, "code"); err != nil {
		t.Fatalf("exchange code default app failed: %v", err)
	}

	info, err := c.FetchUserInfo(ctx, *token)
	if err != nil || info.GoogleID != "gid" {
		t.Fatalf("fetch user info failed: %v %#v", err, info)
	}
	cals, err := c.ListCalendars(ctx, *token)
	if err != nil || len(cals) != 1 || cals[0].Kind != "primary" {
		t.Fatalf("list calendars failed: %v %#v", err, cals)
	}
	lists, err := c.ListTaskLists(ctx, *token)
	if err != nil || len(lists) != 1 {
		t.Fatalf("list task lists failed: %v %#v", err, lists)
	}
	taskListID, err := c.CreateTaskList(ctx, *token, "new")
	if err != nil || taskListID != "new-list" {
		t.Fatalf("create task list failed: %v id=%s", err, taskListID)
	}
	if err := c.DeleteTaskList(ctx, *token, "list1"); err != nil {
		t.Fatalf("delete task list failed: %v", err)
	}

	eventsPage, err := c.ListCalendarEvents(ctx, *token, "cal1", "", "", nil, nil)
	if err != nil || len(eventsPage.Events) != 1 {
		t.Fatalf("list calendar events failed: %v %#v", err, eventsPage)
	}
	tasksPage, err := c.ListTasks(ctx, *token, "list1", "")
	if err != nil || len(tasksPage.Items) != 1 || tasksPage.Items[0].DueDate == nil {
		t.Fatalf("list tasks failed: %v %#v", err, tasksPage)
	}
	if tasksPage.Items[0].ParentGoogleID == nil || *tasksPage.Items[0].ParentGoogleID != "parent-1" {
		t.Fatalf("expected task parent parsed, got %#v", tasksPage.Items[0])
	}

	color := "2"
	rrule := "FREQ=DAILY"
	etag := "etag-in"
	start := time.Date(2026, 3, 5, 1, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 5, 2, 0, 0, 0, time.UTC)
	eventInput := portout.CalendarEventUpsertInput{
		Title: "A", StartTime: start, EndTime: end, Timezone: "UTC",
		ColorID: &color, RecurrenceRule: &rrule, ETag: &etag,
	}
	newEventID, newETag, err := c.CreateCalendarEvent(ctx, *token, "cal1", eventInput)
	if err != nil || newEventID != "evt-created" || newETag == nil {
		t.Fatalf("create calendar event failed: %v id=%s etag=%v", err, newEventID, newETag)
	}
	if updatedETag, err := c.UpdateCalendarEvent(ctx, *token, "cal1", "evt1", eventInput); err != nil || updatedETag == nil {
		t.Fatalf("update calendar event failed: %v etag=%v", err, updatedETag)
	}
	if err := c.DeleteCalendarEvent(ctx, *token, "cal1", "evt1"); err != nil {
		t.Fatalf("delete calendar event failed: %v", err)
	}

	todoInput := portout.TodoUpsertInput{Title: "T", Notes: "N"}
	newTaskID, err := c.CreateTask(ctx, *token, "list1", todoInput)
	if err != nil || newTaskID != "task-created" {
		t.Fatalf("create task failed: %v id=%s", err, newTaskID)
	}
	if err := c.UpdateTask(ctx, *token, "list1", "task1", todoInput); err != nil {
		t.Fatalf("update task failed: %v", err)
	}
	parentID := "parent-1"
	previousID := "prev-1"
	if err := c.MoveTask(ctx, *token, "list1", "task1", &parentID, &previousID); err != nil {
		t.Fatalf("move task failed: %v", err)
	}
	if err := c.DeleteTask(ctx, *token, "list1", "task1"); err != nil {
		t.Fatalf("delete task failed: %v", err)
	}

	if _, ok := hits["/oauth2/v3/userinfo"]; !ok {
		t.Fatalf("expected userinfo call, hits=%v", hits)
	}
}

func TestGoogleClientErrorBranches(t *testing.T) {
	c := NewOAuthClient("cid", "secret", map[string]string{"web": "https://web.example.com/callback"})
	ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/oauth2/v3/userinfo":
			return jsonResp(http.StatusUnauthorized, `{}`), nil
		case "/calendar/v3/users/me/calendarList":
			return jsonResp(http.StatusBadGateway, `{}`), nil
		case "/tasks/v1/users/@me/lists":
			if req.Method == http.MethodPost {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"fail","errors":[{"reason":"invalid"}]}}`), nil
			}
			return jsonResp(http.StatusBadGateway, `{}`), nil
		case "/calendar/v3/calendars/cal1/events":
			if req.Method == http.MethodPost {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			return jsonResp(http.StatusBadGateway, `{}`), nil
		case "/tasks/v1/lists/list1/tasks":
			if req.Method == http.MethodPost {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			return jsonResp(http.StatusBadGateway, `{}`), nil
		default:
			if strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events/evt1") && req.Method == http.MethodPatch {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			if strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal1/events/evt1") && req.Method == http.MethodDelete {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			if strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1") && req.Method == http.MethodPatch {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			if strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1/move") && req.Method == http.MethodPost {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			if strings.Contains(req.URL.Path, "/tasks/v1/lists/list1/tasks/task1") && req.Method == http.MethodDelete {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			if strings.Contains(req.URL.Path, "/tasks/v1/users/@me/lists/list1") && req.Method == http.MethodDelete {
				return jsonResp(http.StatusBadRequest, `{"error":{"message":"bad"}}`), nil
			}
			return jsonResp(http.StatusInternalServerError, `{}`), nil
		}
	})
	token := portout.OAuthToken{AccessToken: "at"}

	if _, err := c.FetchUserInfo(ctx, token); err == nil {
		t.Fatal("expected fetch userinfo error")
	}
	if _, err := c.ListCalendars(ctx, token); err == nil {
		t.Fatal("expected list calendars error")
	}
	if _, err := c.ListTaskLists(ctx, token); err == nil {
		t.Fatal("expected list task lists error")
	}
	if _, err := c.CreateTaskList(ctx, token, "x"); err == nil {
		t.Fatal("expected create task list error")
	}
	if err := c.DeleteTaskList(ctx, token, "list1"); err == nil {
		t.Fatal("expected delete task list error")
	}
	if _, err := c.ListCalendarEvents(ctx, token, "cal1", "", "", nil, nil); err == nil {
		t.Fatal("expected list calendar events error")
	}
	if _, err := c.ListTasks(ctx, token, "list1", ""); err == nil {
		t.Fatal("expected list tasks error")
	}
	if _, _, err := c.CreateCalendarEvent(ctx, token, "cal1", portout.CalendarEventUpsertInput{}); err == nil {
		t.Fatal("expected create calendar event error")
	}
	if _, err := c.UpdateCalendarEvent(ctx, token, "cal1", "evt1", portout.CalendarEventUpsertInput{}); err == nil {
		t.Fatal("expected update calendar event error")
	}
	if err := c.DeleteCalendarEvent(ctx, token, "cal1", "evt1"); err == nil {
		t.Fatal("expected delete calendar event error")
	}
	if _, err := c.CreateTask(ctx, token, "list1", portout.TodoUpsertInput{}); err == nil {
		t.Fatal("expected create task error")
	}
	if err := c.UpdateTask(ctx, token, "list1", "task1", portout.TodoUpsertInput{}); err == nil {
		t.Fatal("expected update task error")
	}
	if err := c.MoveTask(ctx, token, "list1", "task1", nil, nil); err == nil {
		t.Fatal("expected move task error")
	}
	if err := c.DeleteTask(ctx, token, "list1", "task1"); err == nil {
		t.Fatal("expected delete task error")
	}

	if _, _, _, err := parseGoogleEventDateTime("", "", "", "", "UTC"); err == nil {
		t.Fatal("expected invalid datetime error")
	}
	if _, _, _, err := parseGoogleEventDateTime("bad", "", "bad", "", "UTC"); err == nil {
		t.Fatal("expected all-day parse error")
	}
	if _, _, _, err := parseGoogleEventDateTime("", "bad", "", "bad", "UTC"); err == nil {
		t.Fatal("expected datetime parse error")
	}

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		var payload map[string]any
		_ = json.NewDecoder(req.Body).Decode(&payload)
		return jsonResp(http.StatusOK, `{}`), nil
	})}
	if _, err := doGoogleJSONRequest(context.Background(), client, http.MethodPost, "://bad", map[string]any{"x": "y"}); err == nil {
		t.Fatal("expected bad url error")
	}
}

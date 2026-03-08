package google

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	portout "lifebase/internal/auth/port/out"
)

func TestOAuthClientAdditionalBranches(t *testing.T) {
	c := NewOAuthClient("cid", "secret", map[string]string{"web": "https://web.example.com/callback"})
	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}

	t.Run("exchange code error", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if req.URL.Host == "oauth2.googleapis.com" && req.URL.Path == "/token" {
				return jsonResp(http.StatusBadRequest, `{"error":"invalid_grant"}`), nil
			}
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		})
		if _, err := c.ExchangeCodeForApp(ctx, "bad-code", "web"); err == nil {
			t.Fatal("expected exchange error")
		}
	})

	t.Run("userinfo decode error", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			return jsonResp(http.StatusOK, `{invalid`), nil
		})
		if _, err := c.FetchUserInfo(ctx, token); err == nil {
			t.Fatal("expected decode error")
		}
	})

	t.Run("calendar and task list pagination defaults", func(t *testing.T) {
		hits := 0
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			switch req.URL.Path {
			case "/calendar/v3/users/me/calendarList":
				if hits == 0 {
					hits++
					if req.URL.RawQuery != "" {
						t.Fatalf("first request should not have page token: %q", req.URL.RawQuery)
					}
					return jsonResp(http.StatusOK, `{"items":[{"id":"cal-1","summary":"","accessRole":"owner"}],"nextPageToken":"p2"}`), nil
				}
				if !strings.Contains(req.URL.RawQuery, "pageToken=p2") {
					t.Fatalf("expected page token on second request: %q", req.URL.RawQuery)
				}
				return jsonResp(http.StatusOK, `{"items":[{"id":"cal-2","summary":"Work","colorId":"7","selected":false,"accessRole":"reader"}]}`), nil
			case "/tasks/v1/users/@me/lists":
				if req.URL.RawQuery == "" {
					return jsonResp(http.StatusOK, `{"items":[{"id":"list-1","title":""}],"nextPageToken":"next"}`), nil
				}
				if !strings.Contains(req.URL.RawQuery, "pageToken=next") {
					t.Fatalf("expected next page token, got %q", req.URL.RawQuery)
				}
				return jsonResp(http.StatusOK, `{"items":[{"id":"list-2","title":"Personal"}]}`), nil
			default:
				t.Fatalf("unexpected request: %s", req.URL.String())
				return nil, nil
			}
		})

		calendars, err := c.ListCalendars(ctx, token)
		if err != nil {
			t.Fatalf("list calendars err: %v", err)
		}
		if len(calendars) != 2 || calendars[0].Name != "Google Calendar" || calendars[1].ColorID == nil || calendars[1].IsVisible {
			t.Fatalf("unexpected calendars: %#v", calendars)
		}

		lists, err := c.ListTaskLists(ctx, token)
		if err != nil {
			t.Fatalf("list task lists err: %v", err)
		}
		if len(lists) != 2 || lists[0].Name != "Google Tasks" {
			t.Fatalf("unexpected task lists: %#v", lists)
		}
	})

	t.Run("calendar and task list decode errors", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/calendar/v3/users/me/calendarList" || req.URL.Path == "/tasks/v1/users/@me/lists" {
				return jsonResp(http.StatusOK, `{bad`), nil
			}
			t.Fatalf("unexpected request: %s", req.URL.String())
			return nil, nil
		})
		if _, err := c.ListCalendars(ctx, token); err == nil {
			t.Fatal("expected calendar decode error")
		}
		if _, err := c.ListTaskLists(ctx, token); err == nil {
			t.Fatal("expected task list decode error")
		}
	})

	t.Run("list calendar events branches", func(t *testing.T) {
		timeMin := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		timeMax := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
		calls := 0
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal/events") {
				t.Fatalf("unexpected request path: %s", req.URL.String())
			}
			calls++
			if calls == 1 {
				query := req.URL.Query()
				if query.Get("syncToken") != "sync-1" || query.Get("orderBy") != "" {
					t.Fatalf("unexpected syncToken query: %q", req.URL.RawQuery)
				}
				return jsonResp(http.StatusOK, `{"items":[{"id":"skip","status":"confirmed","start":{"dateTime":"bad"},"end":{"dateTime":"bad"}}],"nextSyncToken":"sync-2"}`), nil
			}
			query := req.URL.Query()
			if query.Get("timeMin") == "" || query.Get("timeMax") == "" || query.Get("pageToken") != "p2" || query.Get("orderBy") != "startTime" {
				t.Fatalf("unexpected range query: %q", req.URL.RawQuery)
			}
			return jsonResp(http.StatusOK, `{"items":[
				{"id":"evt-1","status":"confirmed","summary":"","description":"d","location":"loc","colorId":"2","recurrence":["EXDATE:20260301","RRULE:FREQ=DAILY"],"etag":"etag-1","start":{"date":"2026-03-03"},"end":{"date":"2026-03-03"}},
				{"id":"evt-2","status":"confirmed","summary":"Meeting","start":{"dateTime":"2026-03-05T01:00:00Z"},"end":{"dateTime":"2026-03-05T02:00:00Z","timeZone":"UTC"}}
			],"nextPageToken":"next-page","nextSyncToken":"sync-3"}`), nil
		})

		page, err := c.ListCalendarEvents(ctx, token, "cal", "", "sync-1", nil, nil)
		if err != nil {
			t.Fatalf("sync token call err: %v", err)
		}
		if len(page.Events) != 0 || page.NextSyncToken != "sync-2" {
			t.Fatalf("unexpected sync token page: %#v", page)
		}

		page, err = c.ListCalendarEvents(ctx, token, "cal", "p2", "", &timeMin, &timeMax)
		if err != nil {
			t.Fatalf("range call err: %v", err)
		}
		if len(page.Events) != 2 || page.Events[0].Title != "제목 없음" || page.Events[0].RecurrenceRule == nil || *page.Events[0].RecurrenceRule != "FREQ=DAILY" {
			t.Fatalf("unexpected events page: %#v", page)
		}
	})

	t.Run("list tasks decode and due parsing branches", func(t *testing.T) {
		ctxDecode := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			return jsonResp(http.StatusOK, `{bad`), nil
		})
		if _, err := c.ListTasks(ctxDecode, token, "list-1", ""); err == nil {
			t.Fatal("expected list tasks decode error")
		}

		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if req.URL.Query().Get("pageToken") != "p1" {
				t.Fatalf("expected page token query, got %q", req.URL.RawQuery)
			}
			return jsonResp(http.StatusOK, `{"items":[{"id":"task-1","title":"T","notes":"N","status":"needsAction","due":"bad","completed":""}]}`), nil
		})
		page, err := c.ListTasks(ctx, token, "list-1", "p1")
		if err != nil {
			t.Fatalf("unexpected list tasks err: %v", err)
		}
		if len(page.Items) != 1 || page.Items[0].DueDate != nil || page.Items[0].IsDone {
			t.Fatalf("unexpected tasks page: %#v", page)
		}
	})

	t.Run("create and update decode or missing id branches", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/tasks/v1/users/@me/lists":
				return jsonResp(http.StatusOK, `{}`), nil
			case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal/events") && req.Method == http.MethodPost:
				return jsonResp(http.StatusCreated, `{}`), nil
			case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal/events/evt") && req.Method == http.MethodPatch:
				return jsonResp(http.StatusOK, `{bad`), nil
			case strings.Contains(req.URL.Path, "/tasks/v1/lists/list/tasks/task/move") && req.Method == http.MethodPost:
				return nil, errors.New("move transport failed")
			case strings.Contains(req.URL.Path, "/tasks/v1/lists/list/tasks") && req.Method == http.MethodPost:
				return jsonResp(http.StatusOK, `{}`), nil
			case strings.Contains(req.URL.Path, "/tasks/v1/lists/list/tasks/task") && req.Method == http.MethodPatch:
				return nil, errors.New("transport failed")
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}
		})

		if _, err := c.CreateTaskList(ctx, token, "x"); err == nil {
			t.Fatal("expected missing id error on create task list")
		}
		if _, _, err := c.CreateCalendarEvent(ctx, token, "cal", portout.CalendarEventUpsertInput{}); err == nil {
			t.Fatal("expected missing id error on create calendar event")
		}
		if _, err := c.UpdateCalendarEvent(ctx, token, "cal", "evt", portout.CalendarEventUpsertInput{}); err == nil {
			t.Fatal("expected update calendar decode error")
		}
		if _, err := c.CreateTask(ctx, token, "list", portout.TodoUpsertInput{}); err == nil {
			t.Fatal("expected missing id error on create task")
		}
		if err := c.UpdateTask(ctx, token, "list", "task", portout.TodoUpsertInput{}); err == nil {
			t.Fatal("expected update task transport error")
		}
		if err := c.MoveTask(ctx, token, "list", "task", nil, nil); err == nil {
			t.Fatal("expected move task transport error")
		}
	})

	t.Run("create decode and success without optional fields", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			switch {
			case req.URL.Path == "/tasks/v1/users/@me/lists":
				return jsonResp(http.StatusOK, `{bad`), nil
			case strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal-no-etag/events"):
				return jsonResp(http.StatusCreated, `{"id":"evt-no-etag"}`), nil
			case strings.Contains(req.URL.Path, "/tasks/v1/lists/list-decode/tasks"):
				return jsonResp(http.StatusOK, `{bad`), nil
			default:
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
				return nil, nil
			}
		})

		if _, err := c.CreateTaskList(ctx, token, "decode"); err == nil {
			t.Fatal("expected create task list decode error")
		}
		id, etag, err := c.CreateCalendarEvent(ctx, token, "cal-no-etag", portout.CalendarEventUpsertInput{})
		if err != nil || id != "evt-no-etag" || etag != nil {
			t.Fatalf("expected calendar event success without etag, id=%q etag=%v err=%v", id, etag, err)
		}
		if _, err := c.CreateTask(ctx, token, "list-decode", portout.TodoUpsertInput{}); err == nil {
			t.Fatal("expected create task decode error")
		}
	})

	t.Run("create calendar event ok status success", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal-ok/events") {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			}
			return jsonResp(http.StatusOK, `{"id":"evt-ok","etag":"etag-ok"}`), nil
		})

		id, etag, err := c.CreateCalendarEvent(ctx, token, "cal-ok", portout.CalendarEventUpsertInput{})
		if err != nil || id != "evt-ok" || etag == nil || *etag != "etag-ok" {
			t.Fatalf("expected ok status create calendar event success, id=%q etag=%v err=%v", id, etag, err)
		}
	})

	t.Run("list calendar events decode error", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal-decode/events") {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			}
			return jsonResp(http.StatusOK, `{bad`), nil
		})

		if _, err := c.ListCalendarEvents(ctx, token, "cal-decode", "", "", nil, nil); err == nil {
			t.Fatal("expected list calendar events decode error")
		}
	})

	t.Run("create calendar event decode error", func(t *testing.T) {
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			if !strings.Contains(req.URL.Path, "/calendar/v3/calendars/cal-decode/events") {
				t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			}
			return jsonResp(http.StatusCreated, `{bad`), nil
		})

		if _, _, err := c.CreateCalendarEvent(ctx, token, "cal-decode", portout.CalendarEventUpsertInput{}); err == nil {
			t.Fatal("expected create calendar event decode error")
		}
	})

	t.Run("parse and request helper branches", func(t *testing.T) {
		start, end, isAllDay, err := parseGoogleEventDateTime("2026-03-03", "", "2026-03-03", "", "Invalid/Timezone")
		if err != nil || !isAllDay || start.IsZero() || end.Before(start) {
			t.Fatalf("expected all-day fallback success, got start=%v end=%v allDay=%v err=%v", start, end, isAllDay, err)
		}
		if _, _, _, err := parseGoogleEventDateTime("", "2026-03-03T00:00:00Z", "", "", "UTC"); err == nil {
			t.Fatal("expected invalid partial datetime input")
		}
		if _, _, _, err := parseGoogleEventDateTime(
			"2026-03-03", "", "bad-end", "", "UTC",
		); err == nil {
			t.Fatal("expected all-day end parse error")
		}
		if _, _, _, err := parseGoogleEventDateTime(
			"", "2026-03-03T00:00:00Z", "", "bad-end", "UTC",
		); err == nil {
			t.Fatal("expected timed end parse error")
		}

		body := buildGoogleCalendarEventBody(portout.CalendarEventUpsertInput{
			Title:     "x",
			StartTime: time.Date(2026, 3, 5, 5, 0, 0, 0, time.UTC),
			EndTime:   time.Date(2026, 3, 4, 1, 0, 0, 0, time.UTC),
			IsAllDay:  true,
			Timezone:  "Invalid/Timezone",
		})
		endPayload := body["end"].(map[string]string)
		if endPayload["date"] != "2026-03-06" {
			t.Fatalf("expected same-day clamped exclusive end, got %#v", body)
		}

		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected json content-type, got %q", req.Header.Get("Content-Type"))
			}
			var payload map[string]any
			if req.Body != nil {
				_ = json.NewDecoder(req.Body).Decode(&payload)
			}
			return jsonResp(http.StatusOK, `{}`), nil
		})}
		if _, err := doGoogleJSONRequest(context.Background(), client, http.MethodPost, "https://example.com", nil); err != nil {
			t.Fatalf("nil body request should succeed: %v", err)
		}
		if _, err := doGoogleJSONRequest(context.Background(), client, http.MethodPost, "https://example.com", map[string]any{"bad": make(chan int)}); err == nil {
			t.Fatal("expected marshal error")
		}
	})

	t.Run("delete request build error branches", func(t *testing.T) {
		prev := requestWithContext
		requestWithContext = func(context.Context, string, string, io.Reader) (*http.Request, error) {
			return nil, errors.New("request build fail")
		}
		t.Cleanup(func() { requestWithContext = prev })

		ctx := context.Background()
		if err := c.DeleteTaskList(ctx, token, "list1"); err == nil {
			t.Fatal("expected delete task list request build error")
		}
		if err := c.DeleteCalendarEvent(ctx, token, "cal1", "evt1"); err == nil {
			t.Fatal("expected delete calendar event request build error")
		}
		if err := c.DeleteTask(ctx, token, "list1", "task1"); err == nil {
			t.Fatal("expected delete task request build error")
		}
	})

	t.Run("transport errors across google methods", func(t *testing.T) {
		transportErr := errors.New("transport failed")
		ctx := ctxWithTransport(func(req *http.Request) (*http.Response, error) {
			return nil, transportErr
		})

		if _, err := c.FetchUserInfo(ctx, token); !errors.Is(err, transportErr) {
			t.Fatalf("expected fetch user info transport err, got %v", err)
		}
		if _, err := c.ListCalendars(ctx, token); !errors.Is(err, transportErr) {
			t.Fatalf("expected list calendars transport err, got %v", err)
		}
		if _, err := c.ListTaskLists(ctx, token); !errors.Is(err, transportErr) {
			t.Fatalf("expected list task lists transport err, got %v", err)
		}
		if _, err := c.CreateTaskList(ctx, token, "x"); !errors.Is(err, transportErr) {
			t.Fatalf("expected create task list transport err, got %v", err)
		}
		if err := c.DeleteTaskList(ctx, token, "list1"); !errors.Is(err, transportErr) {
			t.Fatalf("expected delete task list transport err, got %v", err)
		}
		if _, err := c.ListCalendarEvents(ctx, token, "cal1", "", "", nil, nil); !errors.Is(err, transportErr) {
			t.Fatalf("expected list calendar events transport err, got %v", err)
		}
		if _, err := c.ListTasks(ctx, token, "list1", ""); !errors.Is(err, transportErr) {
			t.Fatalf("expected list tasks transport err, got %v", err)
		}
		if _, _, err := c.CreateCalendarEvent(ctx, token, "cal1", portout.CalendarEventUpsertInput{}); !errors.Is(err, transportErr) {
			t.Fatalf("expected create calendar event transport err, got %v", err)
		}
		if _, err := c.UpdateCalendarEvent(ctx, token, "cal1", "evt1", portout.CalendarEventUpsertInput{}); !errors.Is(err, transportErr) {
			t.Fatalf("expected update calendar event transport err, got %v", err)
		}
		if err := c.DeleteCalendarEvent(ctx, token, "cal1", "evt1"); !errors.Is(err, transportErr) {
			t.Fatalf("expected delete calendar event transport err, got %v", err)
		}
		if _, err := c.CreateTask(ctx, token, "list1", portout.TodoUpsertInput{}); !errors.Is(err, transportErr) {
			t.Fatalf("expected create task transport err, got %v", err)
		}
		if err := c.MoveTask(ctx, token, "list1", "task1", nil, nil); !errors.Is(err, transportErr) {
			t.Fatalf("expected move task transport err, got %v", err)
		}
		if err := c.DeleteTask(ctx, token, "list1", "task1"); !errors.Is(err, transportErr) {
			t.Fatalf("expected delete task transport err, got %v", err)
		}
	})
}

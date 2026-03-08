package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	portout "lifebase/internal/auth/port/out"
)

type oauthClient struct {
	clientID     string
	clientSecret string
	redirects    map[string]string
}

var requestWithContext = http.NewRequestWithContext

func NewOAuthClient(clientID, clientSecret string, redirects map[string]string) *oauthClient {
	cloned := map[string]string{}
	for k, v := range redirects {
		cloned[k] = v
	}
	return &oauthClient{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirects:    cloned,
	}
}

func (c *oauthClient) AuthURL(state string) string {
	return c.AuthURLForApp(state, "web")
}

func (c *oauthClient) AuthURLForApp(state, app string) string {
	cfg := c.oauthConfig(app)
	return cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
}

func (c *oauthClient) ExchangeCode(ctx context.Context, code string) (*portout.OAuthToken, error) {
	return c.ExchangeCodeForApp(ctx, code, "web")
}

func (c *oauthClient) ExchangeCodeForApp(ctx context.Context, code, app string) (*portout.OAuthToken, error) {
	token, err := c.oauthConfig(app).Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	return &portout.OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}, nil
}

func (c *oauthClient) FetchUserInfo(ctx context.Context, token portout.OAuthToken) (*portout.OAuthUserInfo, error) {
	client := c.apiClient(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned %d", resp.StatusCode)
	}

	var info struct {
		Sub     string `json:"sub"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &portout.OAuthUserInfo{
		GoogleID: info.Sub,
		Email:    info.Email,
		Name:     info.Name,
		Picture:  info.Picture,
	}, nil
}

func (c *oauthClient) ListCalendars(ctx context.Context, token portout.OAuthToken) ([]portout.OAuthCalendar, error) {
	client := c.apiClient(ctx, token)
	pageToken := ""
	calendars := make([]portout.OAuthCalendar, 0, 8)

	for {
		url := "https://www.googleapis.com/calendar/v3/users/me/calendarList"
		if pageToken != "" {
			url += "?pageToken=" + pageToken
		}

		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Items []struct {
				ID         string `json:"id"`
				Summary    string `json:"summary"`
				ColorID    string `json:"colorId"`
				Primary    bool   `json:"primary"`
				Selected   *bool  `json:"selected"`
				AccessRole string `json:"accessRole"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("google calendar list returned %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, item := range payload.Items {
			isVisible := true
			if item.Selected != nil {
				isVisible = *item.Selected
			}
			name := item.Summary
			if name == "" {
				name = "Google Calendar"
			}

			var colorID *string
			if item.ColorID != "" {
				colorID = &item.ColorID
			}
			kind, isReadOnly, isSpecial := classifyGoogleCalendar(item.ID, name, item.Primary, item.AccessRole)

			calendars = append(calendars, portout.OAuthCalendar{
				GoogleID:   item.ID,
				Name:       name,
				ColorID:    colorID,
				IsPrimary:  item.Primary,
				IsVisible:  isVisible,
				Kind:       kind,
				IsReadOnly: isReadOnly,
				IsSpecial:  isSpecial,
				AccessRole: item.AccessRole,
			})
		}

		if payload.NextPageToken == "" {
			break
		}
		pageToken = payload.NextPageToken
	}

	return calendars, nil
}

func (c *oauthClient) ListTaskLists(ctx context.Context, token portout.OAuthToken) ([]portout.OAuthTaskList, error) {
	client := c.apiClient(ctx, token)
	pageToken := ""
	lists := make([]portout.OAuthTaskList, 0, 8)

	for {
		url := "https://tasks.googleapis.com/tasks/v1/users/@me/lists"
		if pageToken != "" {
			url += "?pageToken=" + pageToken
		}

		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}

		var payload struct {
			Items []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}

		decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("google task list returned %d", resp.StatusCode)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}

		for _, item := range payload.Items {
			name := item.Title
			if name == "" {
				name = "Google Tasks"
			}
			lists = append(lists, portout.OAuthTaskList{
				GoogleID: item.ID,
				Name:     name,
			})
		}

		if payload.NextPageToken == "" {
			break
		}
		pageToken = payload.NextPageToken
	}

	return lists, nil
}

func (c *oauthClient) CreateTaskList(
	ctx context.Context,
	token portout.OAuthToken,
	title string,
) (taskListID string, err error) {
	client := c.apiClient(ctx, token)
	body := map[string]any{
		"title": title,
	}

	resp, err := doGoogleJSONRequest(
		ctx,
		client,
		http.MethodPost,
		"https://tasks.googleapis.com/tasks/v1/users/@me/lists",
		body,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", parseGoogleAPIError(resp, "google create task list")
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ID == "" {
		return "", fmt.Errorf("google create task list: missing id")
	}
	return payload.ID, nil
}

func (c *oauthClient) DeleteTaskList(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID string,
) error {
	client := c.apiClient(ctx, token)
	req, err := requestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(
			"https://tasks.googleapis.com/tasks/v1/users/@me/lists/%s",
			url.PathEscape(taskListID),
		),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return parseGoogleAPIError(resp, "google delete task list")
	}
	return nil
}

func (c *oauthClient) ListCalendarEvents(
	ctx context.Context,
	token portout.OAuthToken,
	calendarID, pageToken, syncToken string,
	timeMin, timeMax *time.Time,
) (*portout.OAuthCalendarEventsPage, error) {
	client := c.apiClient(ctx, token)
	params := url.Values{}
	params.Set("singleEvents", "true")
	params.Set("showDeleted", "true")
	params.Set("maxResults", "250")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	if syncToken != "" {
		params.Set("syncToken", syncToken)
	} else {
		if timeMin != nil {
			params.Set("timeMin", timeMin.Format(time.RFC3339))
		}
		if timeMax != nil {
			params.Set("timeMax", timeMax.Format(time.RFC3339))
		}
		params.Set("orderBy", "startTime")
	}

	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/calendar/v3/calendars/%s/events?%s",
		url.PathEscape(calendarID),
		params.Encode(),
	)
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Items []struct {
			ID          string   `json:"id"`
			Status      string   `json:"status"`
			Summary     string   `json:"summary"`
			Description string   `json:"description"`
			Location    string   `json:"location"`
			ColorID     string   `json:"colorId"`
			Recurrence  []string `json:"recurrence"`
			ETag        string   `json:"etag"`
			Start       struct {
				Date     string `json:"date"`
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			} `json:"start"`
			End struct {
				Date     string `json:"date"`
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			} `json:"end"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
		NextSyncToken string `json:"nextSyncToken"`
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google calendar events returned %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}

	events := make([]portout.OAuthCalendarEvent, 0, len(payload.Items))
	for _, item := range payload.Items {
		timezone := item.Start.TimeZone
		if timezone == "" {
			timezone = item.End.TimeZone
		}
		if timezone == "" {
			timezone = "Asia/Seoul"
		}
		startTime, endTime, isAllDay, parseErr := parseGoogleEventDateTime(
			item.Start.Date,
			item.Start.DateTime,
			item.End.Date,
			item.End.DateTime,
			timezone,
		)
		if parseErr != nil {
			continue
		}

		var colorID *string
		if item.ColorID != "" {
			colorID = &item.ColorID
		}

		var recurrenceRule *string
		for _, line := range item.Recurrence {
			if strings.HasPrefix(line, "RRULE:") {
				rule := strings.TrimPrefix(line, "RRULE:")
				recurrenceRule = &rule
				break
			}
		}

		var etag *string
		if item.ETag != "" {
			etag = &item.ETag
		}

		title := item.Summary
		if title == "" {
			title = "제목 없음"
		}
		events = append(events, portout.OAuthCalendarEvent{
			GoogleID:       item.ID,
			Status:         item.Status,
			Title:          title,
			Description:    item.Description,
			Location:       item.Location,
			StartTime:      &startTime,
			EndTime:        &endTime,
			Timezone:       timezone,
			IsAllDay:       isAllDay,
			ColorID:        colorID,
			RecurrenceRule: recurrenceRule,
			ETag:           etag,
		})
	}

	return &portout.OAuthCalendarEventsPage{
		Events:        events,
		NextPageToken: payload.NextPageToken,
		NextSyncToken: payload.NextSyncToken,
	}, nil
}

func (c *oauthClient) ListTasks(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID, pageToken string,
) (*portout.OAuthTasksPage, error) {
	client := c.apiClient(ctx, token)
	params := url.Values{}
	params.Set("maxResults", "100")
	params.Set("showCompleted", "true")
	params.Set("showHidden", "true")
	params.Set("showDeleted", "true")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}
	endpoint := fmt.Sprintf(
		"https://tasks.googleapis.com/tasks/v1/lists/%s/tasks?%s",
		url.PathEscape(taskListID),
		params.Encode(),
	)

	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Items []struct {
			ID        string `json:"id"`
			Parent    string `json:"parent"`
			Title     string `json:"title"`
			Notes     string `json:"notes"`
			Status    string `json:"status"`
			Deleted   bool   `json:"deleted"`
			Hidden    bool   `json:"hidden"`
			Due       string `json:"due"`
			Completed string `json:"completed"`
		} `json:"items"`
		NextPageToken string `json:"nextPageToken"`
	}

	decodeErr := json.NewDecoder(resp.Body).Decode(&payload)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google tasks returned %d", resp.StatusCode)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}

	items := make([]portout.OAuthTask, 0, len(payload.Items))
	for _, item := range payload.Items {
		var dueDate *string
		if item.Due != "" && len(item.Due) >= 10 {
			day := item.Due[:10]
			dueDate = &day
		}
		var parentGoogleID *string
		if item.Parent != "" {
			parentGoogleID = &item.Parent
		}
		items = append(items, portout.OAuthTask{
			GoogleID:       item.ID,
			ParentGoogleID: parentGoogleID,
			Title:          item.Title,
			Notes:          item.Notes,
			DueDate:        dueDate,
			IsDone:         item.Status == "completed" || item.Completed != "",
			IsDeleted:      item.Deleted,
			CompletedAt:    parseOptionalRFC3339(item.Completed),
		})
	}

	return &portout.OAuthTasksPage{
		Items:         items,
		NextPageToken: payload.NextPageToken,
	}, nil
}

func (c *oauthClient) CreateCalendarEvent(
	ctx context.Context,
	token portout.OAuthToken,
	calendarID string,
	input portout.CalendarEventUpsertInput,
) (googleID string, etag *string, err error) {
	client := c.apiClient(ctx, token)
	body := buildGoogleCalendarEventBody(input)

	resp, err := doGoogleJSONRequest(
		ctx,
		client,
		http.MethodPost,
		fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events", url.PathEscape(calendarID)),
		body,
	)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", nil, parseGoogleAPIError(resp, "google create calendar event")
	}

	var payload struct {
		ID   string `json:"id"`
		ETag string `json:"etag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", nil, err
	}
	if payload.ID == "" {
		return "", nil, fmt.Errorf("google create calendar event: missing id")
	}

	if payload.ETag != "" {
		etag = &payload.ETag
	}
	return payload.ID, etag, nil
}

func (c *oauthClient) UpdateCalendarEvent(
	ctx context.Context,
	token portout.OAuthToken,
	calendarID, eventID string,
	input portout.CalendarEventUpsertInput,
) (etag *string, err error) {
	client := c.apiClient(ctx, token)
	body := buildGoogleCalendarEventBody(input)

	resp, err := doGoogleJSONRequest(
		ctx,
		client,
		http.MethodPatch,
		fmt.Sprintf(
			"https://www.googleapis.com/calendar/v3/calendars/%s/events/%s",
			url.PathEscape(calendarID),
			url.PathEscape(eventID),
		),
		body,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseGoogleAPIError(resp, "google update calendar event")
	}

	var payload struct {
		ETag string `json:"etag"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if payload.ETag != "" {
		etag = &payload.ETag
	}
	return etag, nil
}

func (c *oauthClient) DeleteCalendarEvent(
	ctx context.Context,
	token portout.OAuthToken,
	calendarID, eventID string,
) error {
	client := c.apiClient(ctx, token)
	req, err := requestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(
			"https://www.googleapis.com/calendar/v3/calendars/%s/events/%s",
			url.PathEscape(calendarID),
			url.PathEscape(eventID),
		),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return parseGoogleAPIError(resp, "google delete calendar event")
	}
	return nil
}

func (c *oauthClient) CreateTask(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID string,
	input portout.TodoUpsertInput,
) (googleID string, err error) {
	client := c.apiClient(ctx, token)
	body := buildGoogleTaskBody(input)

	resp, err := doGoogleJSONRequest(
		ctx,
		client,
		http.MethodPost,
		fmt.Sprintf("https://tasks.googleapis.com/tasks/v1/lists/%s/tasks", url.PathEscape(taskListID)),
		body,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", parseGoogleAPIError(resp, "google create task")
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ID == "" {
		return "", fmt.Errorf("google create task: missing id")
	}
	return payload.ID, nil
}

func (c *oauthClient) UpdateTask(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID, taskID string,
	input portout.TodoUpsertInput,
) error {
	client := c.apiClient(ctx, token)
	body := buildGoogleTaskBody(input)

	resp, err := doGoogleJSONRequest(
		ctx,
		client,
		http.MethodPatch,
		fmt.Sprintf(
			"https://tasks.googleapis.com/tasks/v1/lists/%s/tasks/%s",
			url.PathEscape(taskListID),
			url.PathEscape(taskID),
		),
		body,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseGoogleAPIError(resp, "google update task")
	}
	return nil
}

func (c *oauthClient) MoveTask(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID, taskID string,
	parentTaskID, previousTaskID *string,
) error {
	client := c.apiClient(ctx, token)
	params := url.Values{}
	if parentTaskID != nil && strings.TrimSpace(*parentTaskID) != "" {
		params.Set("parent", strings.TrimSpace(*parentTaskID))
	}
	if previousTaskID != nil && strings.TrimSpace(*previousTaskID) != "" {
		params.Set("previous", strings.TrimSpace(*previousTaskID))
	}

	endpoint := fmt.Sprintf(
		"https://tasks.googleapis.com/tasks/v1/lists/%s/tasks/%s/move",
		url.PathEscape(taskListID),
		url.PathEscape(taskID),
	)
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	resp, err := doGoogleJSONRequest(ctx, client, http.MethodPost, endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseGoogleAPIError(resp, "google move task")
	}
	return nil
}

func (c *oauthClient) DeleteTask(
	ctx context.Context,
	token portout.OAuthToken,
	taskListID, taskID string,
) error {
	client := c.apiClient(ctx, token)
	req, err := requestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf(
			"https://tasks.googleapis.com/tasks/v1/lists/%s/tasks/%s",
			url.PathEscape(taskListID),
			url.PathEscape(taskID),
		),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return parseGoogleAPIError(resp, "google delete task")
	}
	return nil
}

func (c *oauthClient) apiClient(ctx context.Context, token portout.OAuthToken) *http.Client {
	ot := &oauth2.Token{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
	}
	return c.oauthConfig("web").Client(ctx, ot)
}

func parseGoogleEventDateTime(
	startDate, startDateTime, endDate, endDateTime string,
	timezone string,
) (time.Time, time.Time, bool, error) {
	if startDateTime != "" && endDateTime != "" {
		start, err := time.Parse(time.RFC3339, startDateTime)
		if err != nil {
			return time.Time{}, time.Time{}, false, err
		}
		end, err := time.Parse(time.RFC3339, endDateTime)
		if err != nil {
			return time.Time{}, time.Time{}, false, err
		}
		return start, end, false, nil
	}

	if startDate != "" && endDate != "" {
		loc := time.UTC
		if strings.TrimSpace(timezone) != "" {
			if loaded, err := time.LoadLocation(timezone); err == nil {
				loc = loaded
			}
		}

		start, err := time.ParseInLocation("2006-01-02", startDate, loc)
		if err != nil {
			return time.Time{}, time.Time{}, true, err
		}
		endExclusive, err := time.ParseInLocation("2006-01-02", endDate, loc)
		if err != nil {
			return time.Time{}, time.Time{}, true, err
		}
		// Google all-day "end.date" is exclusive. Convert to inclusive local end-of-day.
		end := endExclusive.Add(-time.Nanosecond)
		if !end.After(start) {
			end = time.Date(
				start.Year(),
				start.Month(),
				start.Day(),
				23,
				59,
				59,
				int(time.Second-time.Nanosecond),
				loc,
			)
		}
		return start, end, true, nil
	}

	return time.Time{}, time.Time{}, false, fmt.Errorf("invalid event datetime")
}

func parseOptionalRFC3339(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func (c *oauthClient) oauthConfig(app string) *oauth2.Config {
	redirectURL, ok := c.redirects[app]
	if !ok || redirectURL == "" {
		redirectURL = c.redirects["web"]
	}

	return &oauth2.Config{
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/tasks",
		},
		Endpoint:    google.Endpoint,
		RedirectURL: redirectURL,
	}
}

func doGoogleJSONRequest(
	ctx context.Context,
	client *http.Client,
	method, endpoint string,
	body any,
) (*http.Response, error) {
	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

func parseGoogleAPIError(resp *http.Response, action string) error {
	var payload struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Errors  []struct {
				Domain  string `json:"domain"`
				Reason  string `json:"reason"`
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&payload)

	status := resp.StatusCode
	if payload.Error.Code != 0 {
		status = payload.Error.Code
	}
	domain := ""
	reason := ""
	if len(payload.Error.Errors) > 0 {
		domain = payload.Error.Errors[0].Domain
		reason = payload.Error.Errors[0].Reason
	}
	message := payload.Error.Message
	if message == "" {
		message = fmt.Sprintf("%s returned %d", action, resp.StatusCode)
	}
	if reason != "" {
		message = fmt.Sprintf("%s (%s)", message, reason)
	}
	return &portout.GoogleAPIError{
		StatusCode: status,
		Domain:     domain,
		Reason:     reason,
		Message:    message,
	}
}

func classifyGoogleCalendar(id, summary string, primary bool, accessRole string) (kind string, isReadOnly bool, isSpecial bool) {
	lowerID := strings.ToLower(id)
	lowerSummary := strings.ToLower(summary)

	isReadOnly = accessRole == "reader" || accessRole == "freeBusyReader"
	if primary {
		return "primary", isReadOnly, false
	}

	if strings.Contains(lowerID, "holiday.calendar.google.com") || strings.Contains(lowerID, "#holiday@") || strings.Contains(lowerSummary, "holiday") || strings.Contains(lowerSummary, "공휴일") {
		return "holiday", isReadOnly, true
	}
	if strings.Contains(lowerID, "#contacts@") || strings.Contains(lowerSummary, "birthday") || strings.Contains(lowerSummary, "생일") {
		return "birthday", isReadOnly, true
	}
	if isReadOnly {
		return "subscribed", true, false
	}
	return "custom", false, false
}

func buildGoogleCalendarEventBody(input portout.CalendarEventUpsertInput) map[string]any {
	body := map[string]any{
		"summary":     input.Title,
		"description": input.Description,
		"location":    input.Location,
	}

	loc := time.UTC
	if input.Timezone != "" {
		if loaded, err := time.LoadLocation(input.Timezone); err == nil {
			loc = loaded
		}
	}
	start := input.StartTime.In(loc)
	end := input.EndTime.In(loc)

	if input.IsAllDay {
		startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
		endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
		if endDay.Before(startDay) {
			endDay = startDay
		}

		// Google all-day payload expects exclusive end date.
		endExclusive := endDay.AddDate(0, 0, 1)
		body["start"] = map[string]string{"date": startDay.Format("2006-01-02")}
		body["end"] = map[string]string{"date": endExclusive.Format("2006-01-02")}
	} else {
		tz := input.Timezone
		if tz == "" {
			tz = "UTC"
		}
		body["start"] = map[string]string{
			"dateTime": input.StartTime.Format(time.RFC3339),
			"timeZone": tz,
		}
		body["end"] = map[string]string{
			"dateTime": input.EndTime.Format(time.RFC3339),
			"timeZone": tz,
		}
	}

	if input.ColorID != nil && *input.ColorID != "" {
		body["colorId"] = *input.ColorID
	}
	if input.RecurrenceRule != nil && *input.RecurrenceRule != "" {
		body["recurrence"] = []string{"RRULE:" + *input.RecurrenceRule}
	}
	return body
}

func buildGoogleTaskBody(input portout.TodoUpsertInput) map[string]any {
	status := "needsAction"
	if input.IsDone {
		status = "completed"
	}

	body := map[string]any{
		"title":  input.Title,
		"notes":  input.Notes,
		"status": status,
	}
	if input.DueDate != nil && *input.DueDate != "" {
		body["due"] = *input.DueDate + "T00:00:00.000Z"
	} else {
		body["due"] = nil
	}
	return body
}

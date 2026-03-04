package google

import (
	"testing"
	"time"

	portout "lifebase/internal/auth/port/out"
)

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

package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lifebase/internal/calendar/domain"
	portin "lifebase/internal/calendar/port/in"
	portout "lifebase/internal/calendar/port/out"
)

type mockCalendarRepo struct {
	cal     *domain.Calendar
	createErr error
	findErr error
	listRes []*domain.Calendar
	listErr error
	updateErr error
	deleteErr error
	updated *domain.Calendar
}

func (m *mockCalendarRepo) Create(_ context.Context, cal *domain.Calendar) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.cal = cal
	return nil
}
func (m *mockCalendarRepo) FindByID(context.Context, string, string) (*domain.Calendar, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.cal, nil
}
func (m *mockCalendarRepo) ListByUser(context.Context, string) ([]*domain.Calendar, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listRes, nil
}
func (m *mockCalendarRepo) Update(_ context.Context, cal *domain.Calendar) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = cal
	return nil
}
func (m *mockCalendarRepo) Delete(context.Context, string) error { return m.deleteErr }

type mockEventRepo struct {
	event *domain.Event
	createErr error
	findErr error
	listRes []*domain.Event
	listErr error
	updateErr error
	softDeleteErr error
	created *domain.Event
	updated *domain.Event
}

func (m *mockEventRepo) Create(_ context.Context, event *domain.Event) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = event
	return nil
}
func (m *mockEventRepo) FindByID(context.Context, string, string) (*domain.Event, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.event, nil
}
func (m *mockEventRepo) ListByRange(context.Context, string, []string, string, string) ([]*domain.Event, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listRes, nil
}
func (m *mockEventRepo) Update(_ context.Context, event *domain.Event) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = event
	return nil
}
func (m *mockEventRepo) SoftDelete(context.Context, string, string) error { return m.softDeleteErr }

type mockReminderRepo struct {
	createErr error
	listRes []domain.EventReminder
	listErr error
	deleteErr error
	created []domain.EventReminder
}

func (m *mockReminderRepo) CreateBatch(_ context.Context, reminders []domain.EventReminder) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = reminders
	return nil
}
func (m *mockReminderRepo) ListByEvent(context.Context, string) ([]domain.EventReminder, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listRes, nil
}
func (m *mockReminderRepo) DeleteByEvent(context.Context, string) error { return m.deleteErr }

type mockOutbox struct {
	created int
	updated int
	deleted int
}

func (m *mockOutbox) EnqueueCreate(context.Context, string, string, time.Time) error { m.created++; return nil }
func (m *mockOutbox) EnqueueUpdate(context.Context, string, string, time.Time) error { m.updated++; return nil }
func (m *mockOutbox) EnqueueDelete(context.Context, string, string, time.Time) error { m.deleted++; return nil }

type mockBackfill struct {
	result *portout.CalendarBackfillResult
	err error
}

func (m *mockBackfill) BackfillEvents(context.Context, string, time.Time, time.Time, []string) (*portout.CalendarBackfillResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func strPtr(v string) *string { return &v }
func boolPtr(v bool) *bool { return &v }

func TestCalendarCRUDFlows(t *testing.T) {
	calRepo := &mockCalendarRepo{
		cal: &domain.Calendar{ID: "c1", UserID: "u1", Name: "My", IsPrimary: false},
		listRes: []*domain.Calendar{{ID: "c1"}},
	}
	uc := &calendarUseCase{calendars: calRepo}

	cal, err := uc.CreateCalendar(context.Background(), "u1", "Work", nil)
	if err != nil || cal.ID == "" {
		t.Fatalf("expected calendar create, got cal=%#v err=%v", cal, err)
	}
	if _, err := uc.ListCalendars(context.Background(), "u1"); err != nil {
		t.Fatalf("expected list success: %v", err)
	}

	if err := uc.UpdateCalendar(context.Background(), "u1", "c1", "Renamed", strPtr("3"), boolPtr(false)); err != nil {
		t.Fatalf("expected update success: %v", err)
	}
	if calRepo.updated == nil || calRepo.updated.Name != "Renamed" || calRepo.updated.IsVisible {
		t.Fatalf("unexpected updated calendar: %#v", calRepo.updated)
	}

	if err := uc.DeleteCalendar(context.Background(), "u1", "c1"); err != nil {
		t.Fatalf("expected delete success: %v", err)
	}
}

func TestCalendarCRUDErrors(t *testing.T) {
	uc := &calendarUseCase{calendars: &mockCalendarRepo{createErr: errors.New("create failed")}}
	if _, err := uc.CreateCalendar(context.Background(), "u1", "X", nil); err == nil {
		t.Fatal("expected create error")
	}

	uc = &calendarUseCase{calendars: &mockCalendarRepo{findErr: errors.New("not found")}}
	if err := uc.UpdateCalendar(context.Background(), "u1", "c1", "", nil, nil); err == nil {
		t.Fatal("expected update find error")
	}
	if err := uc.DeleteCalendar(context.Background(), "u1", "c1"); err == nil {
		t.Fatal("expected delete find error")
	}

	uc = &calendarUseCase{calendars: &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", IsPrimary: true}}}
	if err := uc.DeleteCalendar(context.Background(), "u1", "c1"); err == nil {
		t.Fatal("expected primary delete block")
	}
}

func TestCreateEventFlows(t *testing.T) {
	calRepo := &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", UserID: "u1"}}
	eventRepo := &mockEventRepo{}
	reminderRepo := &mockReminderRepo{}
	outbox := &mockOutbox{}
	uc := &calendarUseCase{calendars: calRepo, events: eventRepo, reminders: reminderRepo, outbox: outbox}

	input := portin.CreateEventInput{
		CalendarID: "c1",
		Title:      "event",
		StartTime:  "2026-03-05T10:00:00Z",
		EndTime:    "2026-03-05T11:00:00Z",
		Reminders:  []portin.ReminderInput{{Minutes: 15}},
	}
	ev, err := uc.CreateEvent(context.Background(), "u1", input)
	if err != nil {
		t.Fatalf("expected create event success: %v", err)
	}
	if ev.Timezone != "Asia/Seoul" {
		t.Fatalf("expected default timezone Asia/Seoul, got %s", ev.Timezone)
	}
	if len(reminderRepo.created) != 1 || reminderRepo.created[0].Method != "popup" {
		t.Fatalf("unexpected reminder create: %#v", reminderRepo.created)
	}
	if outbox.created != 1 {
		t.Fatalf("expected outbox create enqueue, got %d", outbox.created)
	}
}

func TestCreateEventErrorBranches(t *testing.T) {
	base := &calendarUseCase{
		calendars: &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", IsReadOnly: true}},
		events:    &mockEventRepo{},
		reminders: &mockReminderRepo{},
	}
	_, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "2026-03-05T10:00:00Z", EndTime: "2026-03-05T11:00:00Z",
	})
	if !errors.Is(err, domain.ErrReadOnlyCalendar) {
		t.Fatalf("expected readonly error, got %v", err)
	}

	base.calendars = &mockCalendarRepo{findErr: errors.New("missing")}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{CalendarID: "c1"}); err == nil {
		t.Fatal("expected missing calendar error")
	}

	base.calendars = &mockCalendarRepo{cal: &domain.Calendar{ID: "c1"}}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "bad", EndTime: "2026-03-05T11:00:00Z",
	}); err == nil {
		t.Fatal("expected invalid start format")
	}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "2026-03-05T11:00:00Z", EndTime: "bad",
	}); err == nil {
		t.Fatal("expected invalid end format")
	}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "2026-03-05T12:00:00Z", EndTime: "2026-03-05T11:00:00Z",
	}); err == nil {
		t.Fatal("expected end-before-start error")
	}

	base.events = &mockEventRepo{createErr: errors.New("db fail")}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "2026-03-05T10:00:00Z", EndTime: "2026-03-05T11:00:00Z",
	}); err == nil {
		t.Fatal("expected event create error")
	}

	base.events = &mockEventRepo{}
	base.reminders = &mockReminderRepo{createErr: errors.New("reminder fail")}
	if _, err := base.CreateEvent(context.Background(), "u1", portin.CreateEventInput{
		CalendarID: "c1", StartTime: "2026-03-05T10:00:00Z", EndTime: "2026-03-05T11:00:00Z",
		Reminders: []portin.ReminderInput{{Minutes: 5}},
	}); err == nil {
		t.Fatal("expected reminder create error")
	}
}

func TestGetListBackfillUpdateDeleteEventFlows(t *testing.T) {
	now := time.Now()
	calRepo := &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", UserID: "u1"}}
	eventRepo := &mockEventRepo{
		event: &domain.Event{ID: "e1", CalendarID: "c1", Title: "old", StartTime: now, EndTime: now.Add(time.Hour)},
		listRes: []*domain.Event{{ID: "e1", CalendarID: "c1", StartTime: now, EndTime: now.Add(time.Hour)}},
	}
	reminderRepo := &mockReminderRepo{listRes: []domain.EventReminder{{ID: "r1", EventID: "e1"}}}
	outbox := &mockOutbox{}
	backfill := &mockBackfill{result: &portout.CalendarBackfillResult{FetchedEvents: 1, CoveredStart: now, CoveredEnd: now}}
	uc := &calendarUseCase{
		calendars: calRepo, events: eventRepo, reminders: reminderRepo, outbox: outbox, backfill: backfill,
	}

	ev, err := uc.GetEvent(context.Background(), "u1", "e1")
	if err != nil || len(ev.Reminders) != 1 {
		t.Fatalf("expected get event with reminders, ev=%#v err=%v", ev, err)
	}
	if _, err := uc.ListEvents(context.Background(), "u1", nil, "start", "end"); err != nil {
		t.Fatalf("expected list events success: %v", err)
	}

	_, err = uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
		Start: "2026-03-01T00:00:00Z", End: "2026-03-02T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("expected backfill success: %v", err)
	}

	title := "new"
	start := "2026-03-05T10:00:00Z"
	end := "2026-03-05T11:00:00Z"
	_, err = uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{
		Title: &title, StartTime: &start, EndTime: &end, Reminders: []portin.ReminderInput{{Minutes: 10}},
	})
	if err != nil {
		t.Fatalf("expected update success: %v", err)
	}
	if outbox.updated != 1 {
		t.Fatalf("expected outbox update enqueue, got %d", outbox.updated)
	}

	if err := uc.DeleteEvent(context.Background(), "u1", "e1"); err != nil {
		t.Fatalf("expected delete success: %v", err)
	}
	if outbox.deleted != 1 {
		t.Fatalf("expected outbox delete enqueue, got %d", outbox.deleted)
	}
}

func TestEventFlowErrors(t *testing.T) {
	uc := &calendarUseCase{
		calendars: &mockCalendarRepo{findErr: errors.New("missing")},
		events:    &mockEventRepo{findErr: errors.New("missing")},
		reminders: &mockReminderRepo{},
	}
	if _, err := uc.GetEvent(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected get event error")
	}

	if _, err := uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
		Start: "2026-03-01T00:00:00Z", End: "2026-03-02T00:00:00Z",
	}); err == nil {
		t.Fatal("expected backfill not configured error")
	}
	uc.backfill = &mockBackfill{err: errors.New("bf fail")}
	if _, err := uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
		Start: "bad", End: "2026-03-02T00:00:00Z",
	}); err == nil {
		t.Fatal("expected invalid start error")
	}
	if _, err := uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
		Start: "2026-03-03T00:00:00Z", End: "2026-03-02T00:00:00Z",
	}); err == nil {
		t.Fatal("expected end-before-start error")
	}

	uc.events = &mockEventRepo{findErr: errors.New("missing")}
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{}); err == nil {
		t.Fatal("expected update event not found")
	}
	uc.events = &mockEventRepo{event: &domain.Event{ID: "e1", CalendarID: "c1"}}
	uc.calendars = &mockCalendarRepo{findErr: errors.New("missing")}
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{}); err == nil {
		t.Fatal("expected update calendar not found")
	}
	uc.calendars = &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", IsReadOnly: true}}
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{}); !errors.Is(err, domain.ErrReadOnlyCalendar) {
		t.Fatalf("expected readonly error, got %v", err)
	}

	uc.calendars = &mockCalendarRepo{cal: &domain.Calendar{ID: "c1"}}
	invalid := "bad"
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{StartTime: &invalid}); err == nil {
		t.Fatal("expected invalid start update")
	}
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{EndTime: &invalid}); err == nil {
		t.Fatal("expected invalid end update")
	}

	uc.events = &mockEventRepo{event: &domain.Event{ID: "e1", CalendarID: "c1"}, updateErr: errors.New("db fail")}
	if _, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{}); err == nil {
		t.Fatal("expected update db error")
	}

	uc.events = &mockEventRepo{findErr: errors.New("missing")}
	if err := uc.DeleteEvent(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected delete event not found")
	}
	uc.events = &mockEventRepo{event: &domain.Event{ID: "e1", CalendarID: "c1"}}
	uc.calendars = &mockCalendarRepo{findErr: errors.New("missing")}
	if err := uc.DeleteEvent(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected delete calendar not found")
	}
	uc.calendars = &mockCalendarRepo{cal: &domain.Calendar{ID: "c1", IsReadOnly: true}}
	if err := uc.DeleteEvent(context.Background(), "u1", "e1"); !errors.Is(err, domain.ErrReadOnlyCalendar) {
		t.Fatalf("expected readonly delete error, got %v", err)
	}
	uc.calendars = &mockCalendarRepo{cal: &domain.Calendar{ID: "c1"}}
	uc.events = &mockEventRepo{event: &domain.Event{ID: "e1", CalendarID: "c1"}, softDeleteErr: errors.New("delete fail")}
	if err := uc.DeleteEvent(context.Background(), "u1", "e1"); err == nil {
		t.Fatal("expected soft delete error")
	}
}

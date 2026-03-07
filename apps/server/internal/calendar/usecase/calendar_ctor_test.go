package usecase

import "testing"

func TestNewCalendarUseCase(t *testing.T) {
	calRepo := &mockCalendarRepo{}
	eventRepo := &mockEventRepo{}
	reminderRepo := &mockReminderRepo{}
	outbox := &mockOutbox{}
	backfill := &mockBackfill{}

	uc := NewCalendarUseCase(calRepo, eventRepo, reminderRepo, outbox, backfill, nil, nil)
	impl, ok := uc.(*calendarUseCase)
	if !ok {
		t.Fatalf("expected concrete calendar use case, got %T", uc)
	}
	if impl.calendars != calRepo || impl.events != eventRepo || impl.reminders != reminderRepo || impl.outbox != outbox || impl.backfill != backfill {
		t.Fatalf("constructor did not wire dependencies correctly: %#v", impl)
	}
}

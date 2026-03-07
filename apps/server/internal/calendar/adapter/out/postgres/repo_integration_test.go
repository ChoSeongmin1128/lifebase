package postgres

import (
	"context"
	"testing"
	"time"

	"lifebase/internal/calendar/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestCalendarEventReminderAndDaySummaryReposIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"

	calendarRepo := NewCalendarRepo(db)
	eventRepo := NewEventRepo(db)
	reminderRepo := NewReminderRepo(db)
	holidayRepo := NewDaySummaryHolidayRepo(db)
	todoRepo := NewDaySummaryTodoRepo(db)

	c1 := &domain.Calendar{
		ID:              "cal-1",
		UserID:          userID,
		GoogleID:        strPtr("g-cal-1"),
		GoogleAccountID: strPtr("22222222-2222-2222-2222-222222222222"),
		Name:            "Primary",
		Kind:            "google",
		ColorID:         strPtr("1"),
		IsPrimary:       true,
		IsVisible:       true,
		IsReadOnly:      false,
		IsSpecial:       false,
		SyncToken:       strPtr("sync-1"),
		SyncedStart:     timePtr(now.Add(-24 * time.Hour)),
		SyncedEnd:       timePtr(now.Add(24 * time.Hour)),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := calendarRepo.Create(ctx, c1); err != nil {
		t.Fatalf("create calendar c1: %v", err)
	}

	c2 := &domain.Calendar{
		ID:         "cal-2",
		UserID:     userID,
		Name:       "Work",
		Kind:       "custom",
		IsPrimary:  false,
		IsVisible:  true,
		IsReadOnly: false,
		IsSpecial:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := calendarRepo.Create(ctx, c2); err != nil {
		t.Fatalf("create calendar c2: %v", err)
	}

	gotC1, err := calendarRepo.FindByID(ctx, userID, c1.ID)
	if err != nil || gotC1.Name != "Primary" {
		t.Fatalf("find c1 failed: err=%v cal=%#v", err, gotC1)
	}
	if _, err := calendarRepo.FindByID(ctx, userID, "missing"); err == nil {
		t.Fatal("expected missing calendar error")
	}

	cals, err := calendarRepo.ListByUser(ctx, userID)
	if err != nil || len(cals) != 2 {
		t.Fatalf("list calendars failed: err=%v len=%d", err, len(cals))
	}
	if cals[0].ID != "cal-1" {
		t.Fatalf("expected primary first, got %#v", cals)
	}
	emptyCals, err := calendarRepo.ListByUser(ctx, "other-user")
	if err != nil {
		t.Fatalf("list calendars empty failed: %v", err)
	}
	if len(emptyCals) != 0 {
		t.Fatalf("expected no calendars for other user, got %#v", emptyCals)
	}

	c2.Name = "Work Updated"
	c2.ColorID = strPtr("6")
	c2.IsVisible = false
	c2.UpdatedAt = now.Add(time.Minute)
	if err := calendarRepo.Update(ctx, c2); err != nil {
		t.Fatalf("update c2: %v", err)
	}
	gotC2, err := calendarRepo.FindByID(ctx, userID, c2.ID)
	if err != nil || gotC2.Name != "Work Updated" || gotC2.IsVisible {
		t.Fatalf("updated c2 mismatch: err=%v cal=%#v", err, gotC2)
	}

	e1 := &domain.Event{
		ID:          "evt-1",
		CalendarID:  c1.ID,
		UserID:      userID,
		GoogleID:    strPtr("g-evt-1"),
		Title:       "Team sync",
		Description: "daily",
		Location:    "room-a",
		StartTime:   now.Add(1 * time.Hour),
		EndTime:     now.Add(2 * time.Hour),
		Timezone:    "Asia/Seoul",
		IsAllDay:    false,
		ColorID:     strPtr("7"),
		ETag:        strPtr("etag-1"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := eventRepo.Create(ctx, e1); err != nil {
		t.Fatalf("create e1: %v", err)
	}

	e2 := &domain.Event{
		ID:         "evt-2",
		CalendarID: c2.ID,
		UserID:     userID,
		Title:      "All hands",
		StartTime:  now.Add(3 * time.Hour),
		EndTime:    now.Add(4 * time.Hour),
		Timezone:   "Asia/Seoul",
		IsAllDay:   false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := eventRepo.Create(ctx, e2); err != nil {
		t.Fatalf("create e2: %v", err)
	}

	gotE1, err := eventRepo.FindByID(ctx, userID, e1.ID)
	if err != nil || gotE1.Title != "Team sync" {
		t.Fatalf("find e1 failed: err=%v event=%#v", err, gotE1)
	}
	if _, err := eventRepo.FindByID(ctx, userID, "missing"); err == nil {
		t.Fatal("expected missing event error")
	}

	start := now.Format(time.RFC3339)
	end := now.Add(6 * time.Hour).Format(time.RFC3339)

	allEvents, err := eventRepo.ListByRange(ctx, userID, nil, start, end)
	if err != nil || len(allEvents) != 2 {
		t.Fatalf("list by range (all) failed: err=%v len=%d", err, len(allEvents))
	}

	c1Events, err := eventRepo.ListByRange(ctx, userID, []string{c1.ID}, start, end)
	if err != nil || len(c1Events) != 1 || c1Events[0].ID != e1.ID {
		t.Fatalf("list by range (calendar filter) failed: err=%v events=%#v", err, c1Events)
	}

	e1.Title = "Team sync updated"
	e1.Description = "updated"
	e1.Location = "room-b"
	e1.IsAllDay = true
	e1.RecurrenceRule = strPtr("FREQ=DAILY")
	e1.UpdatedAt = now.Add(2 * time.Minute)
	if err := eventRepo.Update(ctx, e1); err != nil {
		t.Fatalf("update e1: %v", err)
	}
	updatedE1, err := eventRepo.FindByID(ctx, userID, e1.ID)
	if err != nil || updatedE1.Title != "Team sync updated" || !updatedE1.IsAllDay {
		t.Fatalf("updated e1 mismatch: err=%v event=%#v", err, updatedE1)
	}

	if err := eventRepo.SoftDelete(ctx, userID, e2.ID); err != nil {
		t.Fatalf("soft delete e2: %v", err)
	}
	if _, err := eventRepo.FindByID(ctx, userID, e2.ID); err == nil {
		t.Fatal("expected deleted e2 not found")
	}

	reminders := []domain.EventReminder{
		{ID: "rem-1", EventID: e1.ID, Method: "popup", Minutes: 30, CreatedAt: now},
		{ID: "rem-2", EventID: e1.ID, Method: "email", Minutes: 10, CreatedAt: now},
	}
	if err := reminderRepo.CreateBatch(ctx, reminders); err != nil {
		t.Fatalf("create reminders: %v", err)
	}
	gotReminders, err := reminderRepo.ListByEvent(ctx, e1.ID)
	if err != nil || len(gotReminders) != 2 {
		t.Fatalf("list reminders failed: err=%v reminders=%#v", err, gotReminders)
	}
	if gotReminders[0].Minutes != 10 {
		t.Fatalf("expected minute asc order, got %#v", gotReminders)
	}
	if err := reminderRepo.DeleteByEvent(ctx, e1.ID); err != nil {
		t.Fatalf("delete reminders by event: %v", err)
	}
	gotReminders, err = reminderRepo.ListByEvent(ctx, e1.ID)
	if err != nil || len(gotReminders) != 0 {
		t.Fatalf("expected no reminders after delete: err=%v reminders=%#v", err, gotReminders)
	}
	noReminders, err := reminderRepo.ListByEvent(ctx, "missing-event")
	if err != nil {
		t.Fatalf("list missing reminders failed: %v", err)
	}
	if len(noReminders) != 0 {
		t.Fatalf("expected no reminders for missing event, got %#v", noReminders)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO public_holidays_kr (locdate, name, year, month, date_kind, is_holiday, fetched_at, created_at, updated_at)
		 VALUES
		   ($1, 'Holiday A', 2026, 3, '01', true, $4, $4, $4),
		   ($2, 'Holiday B', 2026, 3, '01', true, $4, $4, $4),
		   ($3, 'Holiday C', 2026, 4, '01', true, $4, $4, $4)`,
		now, now.AddDate(0, 0, 1), now.AddDate(0, 1, 0), now,
	)
	if err != nil {
		t.Fatalf("insert holidays: %v", err)
	}
	holidays, err := holidayRepo.ListByDateRange(ctx, now, now.AddDate(0, 0, 10))
	if err != nil || len(holidays) != 2 {
		t.Fatalf("list holidays failed: err=%v holidays=%#v", err, holidays)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1', $1, 'Default', 0, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, title, due, priority, is_done, is_pinned, sort_order, created_at, updated_at)
		 VALUES
		   ('todo-1', 'list-1', $1, 'Urgent', $2::date, 'urgent', false, false, 0, $3, $3),
		   ('todo-2', 'list-1', $1, 'Done high', $2::date, 'high', true, false, 1, $3, $3)`,
		userID, now.Format("2006-01-02"), now,
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	undone, err := todoRepo.ListByDueDate(ctx, userID, now.Format("2006-01-02"), false)
	if err != nil || len(undone) != 1 || undone[0].ID != "todo-1" {
		t.Fatalf("list due todos includeDone=false failed: err=%v todos=%#v", err, undone)
	}
	allDue, err := todoRepo.ListByDueDate(ctx, userID, now.Format("2006-01-02"), true)
	if err != nil || len(allDue) != 2 {
		t.Fatalf("list due todos includeDone=true failed: err=%v todos=%#v", err, allDue)
	}

	if err := calendarRepo.Delete(ctx, c2.ID); err != nil {
		t.Fatalf("delete calendar c2: %v", err)
	}
}

func TestEventPushOutboxRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "22222222-2222-2222-2222-222222222222"

	outboxRepo := NewEventPushOutboxRepo(db)

	// Missing event should be ignored.
	if err := outboxRepo.EnqueueCreate(ctx, userID, "missing", now); err != nil {
		t.Fatalf("enqueue create on missing event should be nil: %v", err)
	}

	_, err := db.Exec(ctx,
		`INSERT INTO calendars (id, user_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'Primary', 'google', true, true, false, false, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert calendar: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, title, start_time, end_time, timezone, is_all_day, created_at, updated_at)
		 VALUES ('evt-1', 'cal-1', $1, 'Title', $2, $3, 'Asia/Seoul', false, $2, $2)`,
		userID, now, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert event: %v", err)
	}

	// No google account id should be ignored.
	if err := outboxRepo.EnqueueUpdate(ctx, userID, "evt-1", now); err != nil {
		t.Fatalf("enqueue update without account should be nil: %v", err)
	}

	_, err = db.Exec(ctx, `UPDATE calendars SET google_account_id = $1 WHERE id = 'cal-1'`, accountID)
	if err != nil {
		t.Fatalf("set calendar google_account_id: %v", err)
	}

	if err := outboxRepo.EnqueueCreate(ctx, userID, "evt-1", now); err != nil {
		t.Fatalf("enqueue create: %v", err)
	}
	if err := outboxRepo.EnqueueUpdate(ctx, userID, "evt-1", now.Add(time.Second)); err != nil {
		t.Fatalf("enqueue update: %v", err)
	}
	if err := outboxRepo.EnqueueDelete(ctx, userID, "evt-1", now.Add(2*time.Second)); err != nil {
		t.Fatalf("enqueue delete: %v", err)
	}

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox`).Scan(&count); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 outbox rows, got %d", count)
	}

	if err := outboxRepo.EnqueueCreate(ctx, userID, "evt-1", now); err != nil {
		t.Fatalf("enqueue duplicate create: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox`).Scan(&count); err != nil {
		t.Fatalf("count outbox rows after dedup: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected dedup row count 3, got %d", count)
	}
}

func TestCalendarReposErrorBranchesOnClosedPool(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	calendarRepo := NewCalendarRepo(db)
	eventRepo := NewEventRepo(db)
	reminderRepo := NewReminderRepo(db)
	holidayRepo := NewDaySummaryHolidayRepo(db)
	todoRepo := NewDaySummaryTodoRepo(db)
	outboxRepo := NewEventPushOutboxRepo(db)
	db.Close()

	if _, err := calendarRepo.ListByUser(ctx, "u1"); err == nil {
		t.Fatal("expected calendar list error on closed pool")
	}
	if err := reminderRepo.CreateBatch(ctx, []domain.EventReminder{{ID: "r1", EventID: "e1", Method: "popup", Minutes: 10, CreatedAt: time.Now()}}); err == nil {
		t.Fatal("expected reminder create batch error on closed pool")
	}
	if _, err := reminderRepo.ListByEvent(ctx, "e1"); err == nil {
		t.Fatal("expected reminder list error on closed pool")
	}
	if _, err := holidayRepo.ListByDateRange(ctx, time.Now(), time.Now()); err == nil {
		t.Fatal("expected holiday list error on closed pool")
	}
	if _, err := todoRepo.ListByDueDate(ctx, "u1", time.Now().Format("2006-01-02"), true); err == nil {
		t.Fatal("expected due-date todo list error on closed pool")
	}
	if _, err := eventRepo.ListByRange(ctx, "u1", nil, time.Now().Format(time.RFC3339), time.Now().Add(time.Hour).Format(time.RFC3339)); err == nil {
		t.Fatal("expected event range list error on closed pool")
	}
	if err := outboxRepo.EnqueueCreate(ctx, "u1", "evt-1", time.Now()); err == nil {
		t.Fatal("expected event outbox enqueue error on closed pool")
	}
}

func strPtr(v string) *string { return &v }

func timePtr(v time.Time) *time.Time { return &v }

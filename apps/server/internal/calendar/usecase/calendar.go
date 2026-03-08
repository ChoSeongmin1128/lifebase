package usecase

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"lifebase/internal/calendar/domain"
	portin "lifebase/internal/calendar/port/in"
	portout "lifebase/internal/calendar/port/out"
)

type calendarUseCase struct {
	calendars portout.CalendarRepository
	events    portout.EventRepository
	reminders portout.ReminderRepository
	outbox    portout.EventPushOutbox
	backfill  portout.CalendarBackfillService
	holidays  portout.DaySummaryHolidayRepository
	todos     portout.DaySummaryTodoRepository
}

func NewCalendarUseCase(
	calendars portout.CalendarRepository,
	events portout.EventRepository,
	reminders portout.ReminderRepository,
	outbox portout.EventPushOutbox,
	backfill portout.CalendarBackfillService,
	holidays portout.DaySummaryHolidayRepository,
	todos portout.DaySummaryTodoRepository,
) portin.CalendarUseCase {
	return &calendarUseCase{
		calendars: calendars,
		events:    events,
		reminders: reminders,
		outbox:    outbox,
		backfill:  backfill,
		holidays:  holidays,
		todos:     todos,
	}
}

// Calendars

func (uc *calendarUseCase) CreateCalendar(ctx context.Context, userID, name string, colorID *string) (*domain.Calendar, error) {
	now := time.Now()
	cal := &domain.Calendar{
		ID:         uuid.New().String(),
		UserID:     userID,
		Name:       name,
		Kind:       "custom",
		ColorID:    colorID,
		IsPrimary:  false,
		IsVisible:  true,
		IsReadOnly: false,
		IsSpecial:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := uc.calendars.Create(ctx, cal); err != nil {
		return nil, fmt.Errorf("create calendar: %w", err)
	}
	return cal, nil
}

func (uc *calendarUseCase) ListCalendars(ctx context.Context, userID string) ([]*domain.Calendar, error) {
	return uc.calendars.ListByUser(ctx, userID)
}

func (uc *calendarUseCase) UpdateCalendar(ctx context.Context, userID, calID, name string, colorID *string, isVisible *bool) error {
	cal, err := uc.calendars.FindByID(ctx, userID, calID)
	if err != nil {
		return fmt.Errorf("calendar not found")
	}

	if name != "" {
		cal.Name = name
	}
	if colorID != nil {
		cal.ColorID = colorID
	}
	if isVisible != nil {
		cal.IsVisible = *isVisible
	}
	cal.UpdatedAt = time.Now()
	return uc.calendars.Update(ctx, cal)
}

func (uc *calendarUseCase) DeleteCalendar(ctx context.Context, userID, calID string) error {
	cal, err := uc.calendars.FindByID(ctx, userID, calID)
	if err != nil {
		return fmt.Errorf("calendar not found")
	}
	if cal.IsPrimary {
		return fmt.Errorf("cannot delete primary calendar")
	}
	return uc.calendars.Delete(ctx, calID)
}

// Events

func (uc *calendarUseCase) CreateEvent(ctx context.Context, userID string, input portin.CreateEventInput) (*domain.Event, error) {
	// Verify calendar belongs to user
	cal, err := uc.calendars.FindByID(ctx, userID, input.CalendarID)
	if err != nil {
		return nil, fmt.Errorf("calendar not found")
	}
	if cal.IsReadOnly {
		return nil, domain.ErrReadOnlyCalendar
	}

	startTime, err := time.Parse(time.RFC3339, input.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format")
	}
	endTime, err := time.Parse(time.RFC3339, input.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format")
	}

	if endTime.Before(startTime) {
		return nil, fmt.Errorf("end_time must be after start_time")
	}

	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Seoul"
	}

	now := time.Now()
	event := &domain.Event{
		ID:             uuid.New().String(),
		CalendarID:     input.CalendarID,
		UserID:         userID,
		Title:          input.Title,
		Description:    input.Description,
		Location:       input.Location,
		StartTime:      startTime,
		EndTime:        endTime,
		Timezone:       tz,
		IsAllDay:       input.IsAllDay,
		ColorID:        input.ColorID,
		RecurrenceRule: input.RecurrenceRule,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.events.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("create event: %w", err)
	}
	if uc.outbox != nil {
		_ = uc.outbox.EnqueueCreate(ctx, userID, event.ID, event.UpdatedAt)
	}

	// Create reminders
	if len(input.Reminders) > 0 {
		reminders := make([]domain.EventReminder, len(input.Reminders))
		for i, r := range input.Reminders {
			method := r.Method
			if method == "" {
				method = "popup"
			}
			reminders[i] = domain.EventReminder{
				ID:        uuid.New().String(),
				EventID:   event.ID,
				Method:    method,
				Minutes:   r.Minutes,
				CreatedAt: now,
			}
		}
		if err := uc.reminders.CreateBatch(ctx, reminders); err != nil {
			return nil, fmt.Errorf("create reminders: %w", err)
		}
		event.Reminders = reminders
	}

	return event, nil
}

func (uc *calendarUseCase) GetEvent(ctx context.Context, userID, eventID string) (*domain.Event, error) {
	event, err := uc.events.FindByID(ctx, userID, eventID)
	if err != nil {
		return nil, err
	}

	reminders, err := uc.reminders.ListByEvent(ctx, eventID)
	if err == nil {
		event.Reminders = reminders
	}

	return event, nil
}

func (uc *calendarUseCase) ListEvents(ctx context.Context, userID string, calendarIDs []string, start, end string) ([]*domain.Event, error) {
	return uc.events.ListByRange(ctx, userID, calendarIDs, start, end)
}

func (uc *calendarUseCase) BackfillEvents(ctx context.Context, userID string, input portin.BackfillEventsInput) (*portin.BackfillEventsResult, error) {
	if uc.backfill == nil {
		return nil, fmt.Errorf("calendar backfill is not configured")
	}
	start, err := time.Parse(time.RFC3339, input.Start)
	if err != nil {
		return nil, fmt.Errorf("invalid start format")
	}
	end, err := time.Parse(time.RFC3339, input.End)
	if err != nil {
		return nil, fmt.Errorf("invalid end format")
	}
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}
	result, err := uc.backfill.BackfillEvents(ctx, userID, start, end, input.CalendarIDs)
	if err != nil {
		return nil, err
	}
	return &portin.BackfillEventsResult{
		FetchedEvents: result.FetchedEvents,
		UpdatedEvents: result.UpdatedEvents,
		DeletedEvents: result.DeletedEvents,
		CoveredStart:  result.CoveredStart,
		CoveredEnd:    result.CoveredEnd,
	}, nil
}

func (uc *calendarUseCase) GetDaySummary(
	ctx context.Context,
	userID string,
	input portin.DaySummaryInput,
) (*portin.DaySummaryResult, error) {
	day, err := time.Parse("2006-01-02", input.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format")
	}

	tz := input.Timezone
	if tz == "" {
		tz = "Asia/Seoul"
	}
	location, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone")
	}

	startLocal := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, location)
	endLocal := startLocal.AddDate(0, 0, 1)

	events, err := uc.events.ListByRange(
		ctx,
		userID,
		input.CalendarIDs,
		startLocal.UTC().Format(time.RFC3339),
		endLocal.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return nil, err
	}

	holidayItems := make([]portin.DaySummaryHoliday, 0)
	if uc.holidays != nil {
		holidayRows, err := uc.holidays.ListByDateRange(ctx, day, day)
		if err != nil {
			return nil, err
		}
		seenHoliday := make(map[string]struct{}, len(holidayRows))
		for _, item := range holidayRows {
			dateKey := item.Date.Format("2006-01-02")
			key := dateKey + "|" + item.Name
			if _, exists := seenHoliday[key]; exists {
				continue
			}
			seenHoliday[key] = struct{}{}
			holidayItems = append(holidayItems, portin.DaySummaryHoliday{
				Date: dateKey,
				Name: item.Name,
			})
		}
		sort.SliceStable(holidayItems, func(i, j int) bool {
			if holidayItems[i].Date == holidayItems[j].Date {
				return holidayItems[i].Name < holidayItems[j].Name
			}
			return holidayItems[i].Date < holidayItems[j].Date
		})
	}

	todoItems := make([]portin.DaySummaryTodo, 0)
	if uc.todos != nil {
		rows, err := uc.todos.ListByDueDate(ctx, userID, input.Date, input.IncludeDoneTodos)
		if err != nil {
			return nil, err
		}
		todoItems = make([]portin.DaySummaryTodo, 0, len(rows))
		for _, row := range rows {
			todoItems = append(todoItems, portin.DaySummaryTodo{
				ID:       row.ID,
				ListID:   row.ListID,
				Title:    row.Title,
				DueDate:  row.DueDate,
				DueTime:  row.DueTime,
				Priority: row.Priority,
				IsDone:   row.IsDone,
			})
		}
	}

	return &portin.DaySummaryResult{
		Date:     input.Date,
		Timezone: tz,
		Holidays: holidayItems,
		Events:   events,
		Todos:    todoItems,
	}, nil
}

func (uc *calendarUseCase) UpdateEvent(ctx context.Context, userID, eventID string, input portin.UpdateEventInput) (*domain.Event, error) {
	event, err := uc.events.FindByID(ctx, userID, eventID)
	if err != nil {
		return nil, fmt.Errorf("event not found")
	}
	cal, err := uc.calendars.FindByID(ctx, userID, event.CalendarID)
	if err != nil {
		return nil, fmt.Errorf("calendar not found")
	}
	if cal.IsReadOnly {
		return nil, domain.ErrReadOnlyCalendar
	}

	if input.Title != nil {
		event.Title = *input.Title
	}
	if input.Description != nil {
		event.Description = *input.Description
	}
	if input.Location != nil {
		event.Location = *input.Location
	}
	if input.StartTime != nil {
		t, err := time.Parse(time.RFC3339, *input.StartTime)
		if err != nil {
			return nil, fmt.Errorf("invalid start_time format")
		}
		event.StartTime = t
	}
	if input.EndTime != nil {
		t, err := time.Parse(time.RFC3339, *input.EndTime)
		if err != nil {
			return nil, fmt.Errorf("invalid end_time format")
		}
		event.EndTime = t
	}
	if input.Timezone != nil {
		event.Timezone = *input.Timezone
	}
	if input.IsAllDay != nil {
		event.IsAllDay = *input.IsAllDay
	}
	if input.ColorID != nil {
		event.ColorID = input.ColorID
	}
	if input.RecurrenceRule != nil {
		event.RecurrenceRule = input.RecurrenceRule
	}
	event.UpdatedAt = time.Now()

	if err := uc.events.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("update event: %w", err)
	}
	if uc.outbox != nil {
		_ = uc.outbox.EnqueueUpdate(ctx, userID, event.ID, event.UpdatedAt)
	}

	// Update reminders if provided
	if input.Reminders != nil {
		_ = uc.reminders.DeleteByEvent(ctx, eventID)
		if len(input.Reminders) > 0 {
			reminders := make([]domain.EventReminder, len(input.Reminders))
			now := time.Now()
			for i, r := range input.Reminders {
				method := r.Method
				if method == "" {
					method = "popup"
				}
				reminders[i] = domain.EventReminder{
					ID:        uuid.New().String(),
					EventID:   eventID,
					Method:    method,
					Minutes:   r.Minutes,
					CreatedAt: now,
				}
			}
			_ = uc.reminders.CreateBatch(ctx, reminders)
			event.Reminders = reminders
		}
	}

	return event, nil
}

func (uc *calendarUseCase) DeleteEvent(ctx context.Context, userID, eventID string) error {
	event, err := uc.events.FindByID(ctx, userID, eventID)
	if err != nil {
		return fmt.Errorf("event not found")
	}
	cal, err := uc.calendars.FindByID(ctx, userID, event.CalendarID)
	if err != nil {
		return fmt.Errorf("calendar not found")
	}
	if cal.IsReadOnly {
		return domain.ErrReadOnlyCalendar
	}
	if err := uc.events.SoftDelete(ctx, userID, eventID); err != nil {
		return err
	}
	if uc.outbox != nil {
		_ = uc.outbox.EnqueueDelete(ctx, userID, eventID, time.Now())
	}
	return nil
}

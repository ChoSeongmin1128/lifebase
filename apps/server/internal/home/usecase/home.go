package usecase

import (
	"context"
	"fmt"
	"math"

	"lifebase/internal/home/domain"
	portin "lifebase/internal/home/port/in"
	portout "lifebase/internal/home/port/out"
)

type homeUseCase struct {
	repo portout.HomeRepository
}

func NewHomeUseCase(repo portout.HomeRepository) portin.HomeUseCase {
	return &homeUseCase{repo: repo}
}

func (uc *homeUseCase) GetSummary(ctx context.Context, userID string, input portin.GetSummaryInput) (*domain.Summary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if input.Start.IsZero() || input.End.IsZero() {
		return nil, fmt.Errorf("start and end are required")
	}
	if !input.Start.Before(input.End) {
		return nil, fmt.Errorf("start must be before end")
	}

	eventLimit := clampLimit(input.EventLimit, 5, 20)
	todoLimit := clampLimit(input.TodoLimit, 7, 30)
	recentLimit := clampLimit(input.RecentLimit, 8, 30)

	todayDate := input.Start.Format("2006-01-02")
	startISO := input.Start.Format("2006-01-02T15:04:05-07:00")
	endISO := input.End.Format("2006-01-02T15:04:05-07:00")

	events, eventTotal, err := uc.repo.ListEventsInRange(ctx, userID, startISO, endISO, eventLimit)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	overdueTodos, overdueCount, err := uc.repo.ListOverdueTodos(ctx, userID, todayDate, todoLimit)
	if err != nil {
		return nil, fmt.Errorf("list overdue todos: %w", err)
	}
	todayTodos, todayCount, err := uc.repo.ListTodayTodos(ctx, userID, todayDate, todoLimit)
	if err != nil {
		return nil, fmt.Errorf("list today todos: %w", err)
	}
	recentFiles, fileTotal, err := uc.repo.ListRecentFiles(ctx, userID, recentLimit)
	if err != nil {
		return nil, fmt.Errorf("list recent files: %w", err)
	}
	storage, err := uc.repo.GetStorageSummary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get storage summary: %w", err)
	}

	if storage.QuotaBytes <= 0 {
		storage.UsagePercent = 0
	} else {
		usage := float64(storage.UsedBytes) / float64(storage.QuotaBytes) * 100
		storage.UsagePercent = math.Round(usage*100) / 100
	}

	out := &domain.Summary{}
	out.Window = domain.TimeWindow{Start: input.Start, End: input.End}
	out.Events.Items = events
	out.Events.TotalCount = eventTotal
	out.Todos.Overdue = overdueTodos
	out.Todos.Today = todayTodos
	out.Todos.OverdueCount = overdueCount
	out.Todos.TodayCount = todayCount
	out.Files.Recent = recentFiles
	out.Files.TotalCount = fileTotal
	out.Storage = storage
	return out, nil
}

func clampLimit(input, def, max int) int {
	if input <= 0 {
		return def
	}
	if input > max {
		return max
	}
	return input
}

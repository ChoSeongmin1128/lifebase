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
	typeUsage, err := uc.repo.ListStorageTypeUsage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list storage type usage: %w", err)
	}

	if storage.QuotaBytes <= 0 {
		storage.UsagePercent = 0
	} else {
		usage := float64(storage.UsedBytes) / float64(storage.QuotaBytes) * 100
		storage.UsagePercent = math.Round(usage*100) / 100
	}
	storage.Breakdown = fillStorageBreakdown(typeUsage, storage.UsedBytes)

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

func fillStorageBreakdown(raw []domain.StorageTypeUsage, usedBytes int64) []domain.StorageTypeUsage {
	byType := map[string]int64{
		"image":    0,
		"video":    0,
		"document": 0,
		"other":    0,
	}

	for _, item := range raw {
		if _, ok := byType[item.Type]; !ok {
			byType["other"] += item.Bytes
			continue
		}
		byType[item.Type] += item.Bytes
	}

	denominator := usedBytes
	if denominator <= 0 {
		denominator = byType["image"] + byType["video"] + byType["document"] + byType["other"]
	}

	types := []string{"image", "video", "document", "other"}
	out := make([]domain.StorageTypeUsage, 0, len(types))
	for _, t := range types {
		bytes := byType[t]
		percent := 0.0
		if denominator > 0 {
			percent = math.Round((float64(bytes)/float64(denominator)*100)*100) / 100
		}
		out = append(out, domain.StorageTypeUsage{
			Type:    t,
			Bytes:   bytes,
			Percent: percent,
		})
	}
	return out
}

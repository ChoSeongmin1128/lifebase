package out

import (
	"context"

	"lifebase/internal/home/domain"
)

type HomeRepository interface {
	ListEventsInRange(ctx context.Context, userID string, startISO, endISO string, limit int) ([]domain.EventSummary, int, error)
	ListOverdueTodos(ctx context.Context, userID, todayDate string, limit int) ([]domain.TodoSummary, int, error)
	ListTodayTodos(ctx context.Context, userID, todayDate string, limit int) ([]domain.TodoSummary, int, error)
	ListRecentFiles(ctx context.Context, userID string, limit int) ([]domain.RecentFileSummary, int, error)
	GetStorageSummary(ctx context.Context, userID string) (domain.StorageSummary, error)
	ListStorageTypeUsage(ctx context.Context, userID string) ([]domain.StorageTypeUsage, error)
}

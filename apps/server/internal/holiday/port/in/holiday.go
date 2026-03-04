package in

import (
	"context"
	"time"

	"lifebase/internal/holiday/domain"
)

type RefreshRangeInput struct {
	FromYear *int `json:"from_year"`
	ToYear   *int `json:"to_year"`
}

type RefreshRangeResult struct {
	MonthsTotal     int       `json:"months_total"`
	MonthsRefreshed int       `json:"months_refreshed"`
	ItemsUpserted   int       `json:"items_upserted"`
	RefreshedAt     time.Time `json:"refreshed_at"`
}

type HolidayUseCase interface {
	ListRange(ctx context.Context, start, end time.Time) ([]domain.Holiday, error)
	RefreshRange(ctx context.Context, input RefreshRangeInput) (*RefreshRangeResult, error)
}

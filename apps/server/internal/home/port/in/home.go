package in

import (
	"context"
	"time"

	"lifebase/internal/home/domain"
)

type GetSummaryInput struct {
	Start       time.Time
	End         time.Time
	EventLimit  int
	TodoLimit   int
	RecentLimit int
}

type HomeUseCase interface {
	GetSummary(ctx context.Context, userID string, input GetSummaryInput) (*domain.Summary, error)
}

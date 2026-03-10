package out

import (
	"context"

	"lifebase/internal/gallery/domain"
)

type MediaRepository interface {
	ListMedia(ctx context.Context, userID string, mimePrefix string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.Media, error)
	GetMedia(ctx context.Context, userID, fileID string) (*domain.Media, error)
}

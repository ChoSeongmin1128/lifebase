package in

import (
	"context"

	"lifebase/internal/cloud/domain"
)

type GalleryUseCase interface {
	ListMedia(ctx context.Context, userID string, mediaType string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.File, string, error)
}

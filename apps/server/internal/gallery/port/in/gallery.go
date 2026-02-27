package in

import (
	"context"

	"lifebase/internal/gallery/domain"
)

type GalleryUseCase interface {
	ListMedia(ctx context.Context, userID string, mediaType string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.Media, string, error)
}

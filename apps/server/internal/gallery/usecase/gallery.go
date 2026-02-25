package usecase

import (
	"context"

	"lifebase/internal/cloud/domain"
	portin "lifebase/internal/gallery/port/in"
	portout "lifebase/internal/gallery/port/out"
)

type galleryUseCase struct {
	media portout.MediaRepository
}

func NewGalleryUseCase(media portout.MediaRepository) portin.GalleryUseCase {
	return &galleryUseCase{media: media}
}

func (uc *galleryUseCase) ListMedia(ctx context.Context, userID string, mediaType string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.File, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	files, err := uc.media.ListMedia(ctx, userID, mediaType, sortBy, sortDir, cursor, limit+1)
	if err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(files) > limit {
		nextCursor = files[limit-1].ID
		files = files[:limit]
	}

	return files, nextCursor, nil
}

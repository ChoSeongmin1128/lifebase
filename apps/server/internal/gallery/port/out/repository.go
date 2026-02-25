package out

import (
	"context"

	"lifebase/internal/cloud/domain"
)

type MediaRepository interface {
	ListMedia(ctx context.Context, userID string, mimePrefix string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.File, error)
}

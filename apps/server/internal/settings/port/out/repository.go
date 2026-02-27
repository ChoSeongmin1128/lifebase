package out

import (
	"context"

	"lifebase/internal/settings/domain"
)

type SettingsRepository interface {
	ListByUser(ctx context.Context, userID string) ([]domain.Setting, error)
	Set(ctx context.Context, userID, key, value string) error
}

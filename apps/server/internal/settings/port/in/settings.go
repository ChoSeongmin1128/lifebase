package in

import "context"

type SettingsUseCase interface {
	GetAll(ctx context.Context, userID string) (map[string]string, error)
	Update(ctx context.Context, userID string, values map[string]string) error
}

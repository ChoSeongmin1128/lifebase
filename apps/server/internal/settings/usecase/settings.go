package usecase

import (
	"context"

	portin "lifebase/internal/settings/port/in"
	portout "lifebase/internal/settings/port/out"
)

type settingsUseCase struct {
	repo portout.SettingsRepository
}

func NewSettingsUseCase(repo portout.SettingsRepository) portin.SettingsUseCase {
	return &settingsUseCase{repo: repo}
}

func (uc *settingsUseCase) GetAll(ctx context.Context, userID string) (map[string]string, error) {
	settings, err := uc.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	out := make(map[string]string, len(settings))
	for _, s := range settings {
		out[s.Key] = s.Value
	}
	return out, nil
}

func (uc *settingsUseCase) Update(ctx context.Context, userID string, values map[string]string) error {
	for key, value := range values {
		if err := uc.repo.Set(ctx, userID, key, value); err != nil {
			return err
		}
	}
	return nil
}

package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/settings/domain"
)

type SettingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) ListByUser(ctx context.Context, userID string) ([]domain.Setting, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key, value FROM user_settings WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]domain.Setting, 0)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		settings = append(settings, domain.Setting{
			UserID: userID,
			Key:    k,
			Value:  v,
		})
	}
	return settings, nil
}

func (r *SettingsRepo) Set(ctx context.Context, userID, key, value string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, key) DO UPDATE SET value = $3, updated_at = $4`,
		userID, key, value, time.Now(),
	)
	return err
}

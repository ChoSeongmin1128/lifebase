package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	portout "lifebase/internal/cloud/port/out"
)

const starPrefix = "cloud_star:"

type starRepo struct {
	db *pgxpool.Pool
}

func NewStarRepo(db *pgxpool.Pool) *starRepo {
	return &starRepo{db: db}
}

func (r *starRepo) List(ctx context.Context, userID string) ([]portout.StarRef, error) {
	rows, err := r.db.Query(ctx,
		`SELECT key
		 FROM user_settings
		 WHERE user_id = $1 AND key LIKE $2
		 ORDER BY updated_at DESC`, userID, starPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStarRows(rows)
}

func (r *starRepo) Set(ctx context.Context, userID, itemID, itemType string) error {
	key := buildStarKey(itemType, itemID)
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, key) DO UPDATE
		 SET value = $3, updated_at = $4`, userID, key, "1", time.Now())
	return err
}

func (r *starRepo) Unset(ctx context.Context, userID, itemID, itemType string) error {
	key := buildStarKey(itemType, itemID)
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_settings WHERE user_id = $1 AND key = $2`, userID, key)
	return err
}

func buildStarKey(itemType, itemID string) string {
	return starPrefix + itemType + ":" + itemID
}

func parseStarKey(key string) (portout.StarRef, bool) {
	if !strings.HasPrefix(key, starPrefix) {
		return portout.StarRef{}, false
	}
	parts := strings.Split(key, ":")
	if len(parts) != 3 {
		return portout.StarRef{}, false
	}
	itemType := parts[1]
	itemID := parts[2]
	if itemType != "file" && itemType != "folder" {
		return portout.StarRef{}, false
	}
	if itemID == "" {
		return portout.StarRef{}, false
	}
	return portout.StarRef{
		ItemID:   itemID,
		ItemType: itemType,
	}, true
}

func scanStarRows(rows pgx.Rows) ([]portout.StarRef, error) {
	refs := make([]portout.StarRef, 0)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		ref, ok := parseStarKey(key)
		if !ok {
			continue
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return refs, nil
}

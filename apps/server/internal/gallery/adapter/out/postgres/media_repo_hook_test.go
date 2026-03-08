package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"lifebase/internal/gallery/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestMediaRepoScanHookAndDefaultSortBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewMediaRepo(db)

	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now().UTC().Truncate(time.Second)
	folderID := "33333333-3333-3333-3333-333333333333"

	if _, err := db.Exec(ctx,
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, created_at, updated_at)
		 VALUES ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1', $1, $2, 'x.bin', 'application/octet-stream', 10, 'u1/f1', 'done', $3, $3)`,
		userID, folderID, now,
	); err != nil {
		t.Fatalf("insert file: %v", err)
	}

	// Unknown sortBy/sortDir should fall back to created_at DESC.
	items, err := repo.ListMedia(ctx, userID, "unknown-type", "unknown-sort", "UP", "", 10)
	if err != nil {
		t.Fatalf("list media with default sort branches: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("default mime branch should filter to image/video only, got %d", len(items))
	}

	prev := scanMediaFilesFn
	scanMediaFilesFn = func(pgx.Rows) ([]*domain.Media, error) {
		return nil, errors.New("scan fail")
	}
	t.Cleanup(func() { scanMediaFilesFn = prev })

	if _, err := repo.ListMedia(ctx, userID, "", "created_at", "desc", "", 10); err == nil {
		t.Fatal("expected ListMedia scan hook error")
	}
}

package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"lifebase/internal/testutil/dbtest"
)

func ensureGalleryTestDSN(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgres://seongmin@localhost:5432/lifebase_test?sslmode=disable"
	}
	t.Setenv("LIFEBASE_TEST_DATABASE_URL", dsn)
}

func TestMediaRepoCoverageWithForcedDBEnv(t *testing.T) {
	ensureGalleryTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	repo := NewMediaRepo(db)
	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "22222222-2222-2222-2222-222222222222"
	folderID := "33333333-3333-3333-3333-333333333333"
	now := time.Now().UTC().Truncate(time.Second)

	insert := func(id, uid, name, mime string, size int64, takenAt *time.Time) {
		_, err := db.Exec(ctx,
			`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,'done',$8,$9,$9)`,
			id, uid, folderID, name, mime, size, "storage/"+id, takenAt, now,
		)
		if err != nil {
			t.Fatalf("insert file %s: %v", id, err)
		}
	}

	taken := now.Add(-time.Hour)
	insert("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", userID, "a-image.png", "image/png", 100, &taken)
	insert("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2", userID, "b-video.mp4", "video/mp4", 200, nil)
	insert("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3", userID, "c-doc.txt", "text/plain", 300, nil)
	insert("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1", otherUserID, "x-image.png", "image/png", 100, nil)

	allMedia, err := repo.ListMedia(ctx, userID, "", "created_at", "desc", "", 20)
	if err != nil || len(allMedia) != 2 {
		t.Fatalf("default filter should return image/video only: len=%d err=%v", len(allMedia), err)
	}

	images, err := repo.ListMedia(ctx, userID, "image", "name", "asc", "", 20)
	if err != nil || len(images) != 1 || images[0].MimeType != "image/png" {
		t.Fatalf("image filter mismatch: rows=%#v err=%v", images, err)
	}

	videos, err := repo.ListMedia(ctx, userID, "video", "size", "desc", "", 20)
	if err != nil || len(videos) != 1 || videos[0].MimeType != "video/mp4" {
		t.Fatalf("video filter mismatch: rows=%#v err=%v", videos, err)
	}

	withCursor, err := repo.ListMedia(ctx, userID, "", "taken_at", "asc", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1", 20)
	if err != nil {
		t.Fatalf("cursor listing failed: %v", err)
	}
	if len(withCursor) == 0 {
		t.Fatal("expected at least one row after cursor")
	}

	db.Close()
	if _, err := repo.ListMedia(context.Background(), userID, "", "created_at", "desc", "", 20); err == nil {
		t.Fatal("expected list media query error on closed pool")
	}
}

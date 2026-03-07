package postgres

import (
	"context"
	"testing"
	"time"

	"lifebase/internal/testutil/dbtest"
)

func TestMediaRepoListMediaIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	userID := "11111111-1111-1111-1111-111111111111"
	otherUserID := "22222222-2222-2222-2222-222222222222"
	now := time.Now().UTC().Truncate(time.Second)
	folderID := "33333333-3333-3333-3333-333333333333"

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

	repo := NewMediaRepo(db)

	allMedia, err := repo.ListMedia(ctx, userID, "", "created_at", "desc", "", 20)
	if err != nil {
		t.Fatalf("list all media: %v", err)
	}
	if len(allMedia) != 2 {
		t.Fatalf("default media filter should include image/video only, got %d", len(allMedia))
	}

	images, err := repo.ListMedia(ctx, userID, "image", "name", "asc", "", 20)
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if len(images) != 1 || images[0].MimeType != "image/png" {
		t.Fatalf("unexpected image filter result: %#v", images)
	}

	videos, err := repo.ListMedia(ctx, userID, "video", "size", "desc", "", 20)
	if err != nil {
		t.Fatalf("list videos: %v", err)
	}
	if len(videos) != 1 || videos[0].MimeType != "video/mp4" {
		t.Fatalf("unexpected video filter result: %#v", videos)
	}

	// Cursor path and taken_at sort path.
	cursor := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1"
	withCursor, err := repo.ListMedia(ctx, userID, "", "taken_at", "asc", cursor, 20)
	if err != nil {
		t.Fatalf("list with cursor: %v", err)
	}
	if len(withCursor) == 0 {
		t.Fatal("expected at least one row after cursor")
	}
}

func TestMediaRepoListMediaClosedPoolError(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	db.Close()

	repo := NewMediaRepo(db)
	if _, err := repo.ListMedia(context.Background(), "u1", "", "created_at", "desc", "", 20); err == nil {
		t.Fatal("expected list media query error on closed pool")
	}
}

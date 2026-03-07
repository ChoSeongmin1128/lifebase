package worker

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/testutil/dbtest"
)

func TestThumbnailHandlerProcessTaskSuccessPaths(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	dataPath := t.TempDir()
	thumbPath := t.TempDir()
	installFakeMediaTools(t, map[string]string{
		"vipsthumbnail": "#!/bin/sh\nout=''\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = \"--output\" ]; then\n    shift\n    out=\"$1\"\n  fi\n  shift\ndone\nout=${out%%[*}\nmkdir -p \"$(dirname \"$out\")\"\n: > \"$out\"\n",
		"ffmpeg":        "#!/bin/sh\nfor last; do :; done\nmkdir -p \"$(dirname \"$last\")\"\n: > \"$last\"\n",
		"exiftool":      "#!/bin/sh\necho \"2025:01:15 14:30:00\"\n",
	})

	imageFileID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	imageStoragePath := "media/image-src.jpg"
	seedFileRow(t, db, imageFileID, userID, imageStoragePath, now)
	writeDummyFile(t, filepath.Join(dataPath, imageStoragePath))

	handler := NewThumbnailHandler(db, dataPath, thumbPath)
	imageTask, err := NewThumbnailTask(imageFileID, userID, imageStoragePath, "image/jpeg")
	if err != nil {
		t.Fatalf("new image task: %v", err)
	}
	if err := handler.ProcessTask(ctx, imageTask); err != nil {
		t.Fatalf("process image task: %v", err)
	}
	assertThumbStatus(t, db, imageFileID, "done")
	assertThumbExists(t, filepath.Join(thumbPath, userID, imageFileID+"_small.webp"))
	assertThumbExists(t, filepath.Join(thumbPath, userID, imageFileID+"_medium.webp"))

	var takenAt *time.Time
	if err := db.QueryRow(ctx, `SELECT taken_at FROM files WHERE id = $1`, imageFileID).Scan(&takenAt); err != nil {
		t.Fatalf("query taken_at: %v", err)
	}
	if takenAt == nil {
		t.Fatal("expected taken_at to be extracted for image")
	}

	videoFileID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	videoStoragePath := "media/video-src.mp4"
	seedFileRow(t, db, videoFileID, userID, videoStoragePath, now)
	writeDummyFile(t, filepath.Join(dataPath, videoStoragePath))

	videoTask, err := NewThumbnailTask(videoFileID, userID, videoStoragePath, "video/mp4")
	if err != nil {
		t.Fatalf("new video task: %v", err)
	}
	if err := handler.ProcessTask(ctx, videoTask); err != nil {
		t.Fatalf("process video task: %v", err)
	}
	assertThumbStatus(t, db, videoFileID, "done")
	assertThumbExists(t, filepath.Join(thumbPath, userID, videoFileID+"_small.webp"))
	assertThumbExists(t, filepath.Join(thumbPath, userID, videoFileID+"_medium.webp"))

	docFileID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	docStoragePath := "docs/doc.txt"
	seedFileRow(t, db, docFileID, userID, docStoragePath, now)
	writeDummyFile(t, filepath.Join(dataPath, docStoragePath))

	docTask, err := NewThumbnailTask(docFileID, userID, docStoragePath, "text/plain")
	if err != nil {
		t.Fatalf("new doc task: %v", err)
	}
	if err := handler.ProcessTask(ctx, docTask); err != nil {
		t.Fatalf("process non-media task: %v", err)
	}
	assertThumbStatus(t, db, docFileID, "done")
}

func TestThumbnailHandlerProcessTaskFailurePaths(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	dataPath := t.TempDir()
	thumbRoot := t.TempDir()

	// Force thumbnail generation failure.
	installFakeMediaTools(t, map[string]string{
		"vipsthumbnail": "#!/bin/sh\nexit 1\n",
		"ffmpeg":        "#!/bin/sh\nexit 1\n",
		"exiftool":      "#!/bin/sh\nexit 0\n",
	})

	fileID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	storagePath := "media/fail-src.jpg"
	seedFileRow(t, db, fileID, userID, storagePath, now)
	writeDummyFile(t, filepath.Join(dataPath, storagePath))

	handler := NewThumbnailHandler(db, dataPath, thumbRoot)
	task, err := NewThumbnailTask(fileID, userID, storagePath, "image/jpeg")
	if err != nil {
		t.Fatalf("new task: %v", err)
	}
	if err := handler.ProcessTask(ctx, task); err == nil {
		t.Fatal("expected thumbnail generation failure")
	}
	assertThumbStatus(t, db, fileID, "failed")

	// Force create thumb dir failure.
	thumbPathAsFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(thumbPathAsFile, []byte("x"), 0644); err != nil {
		t.Fatalf("write thumb path file: %v", err)
	}
	okToolsDir := installFakeMediaTools(t, map[string]string{
		"vipsthumbnail": "#!/bin/sh\nexit 0\n",
		"ffmpeg":        "#!/bin/sh\nexit 0\n",
		"exiftool":      "#!/bin/sh\nexit 0\n",
	})
	prevPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", okToolsDir+string(os.PathListSeparator)+prevPath); err != nil {
		t.Fatalf("set PATH for mkdir failure test: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", prevPath)
	})

	fileID2 := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	storagePath2 := "media/dir-fail.jpg"
	seedFileRow(t, db, fileID2, userID, storagePath2, now)
	writeDummyFile(t, filepath.Join(dataPath, storagePath2))

	handler2 := NewThumbnailHandler(db, dataPath, thumbPathAsFile)
	task2, err := NewThumbnailTask(fileID2, userID, storagePath2, "image/jpeg")
	if err != nil {
		t.Fatalf("new task2: %v", err)
	}
	if err := handler2.ProcessTask(ctx, task2); err == nil {
		t.Fatal("expected create thumb dir failure")
	}
}

func TestWorkerServerAndClientValidRedisURI(t *testing.T) {
	client := NewAsynqClient("redis://127.0.0.1:6379/0")
	if client == nil {
		t.Fatal("expected non-nil asynq client for valid redis URI")
	}
	client.Close()

	srv := StartWorkerServer("redis://127.0.0.1:6379/0", nil, t.TempDir(), t.TempDir())
	if srv == nil {
		t.Fatal("expected non-nil worker server for valid redis URI")
	}
	srv.Shutdown()
}

func seedFileRow(t *testing.T, db *pgxpool.Pool, fileID, userID, storagePath string, now time.Time) {
	t.Helper()
	_, err := db.Exec(context.Background(),
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, created_at, updated_at)
		 VALUES ($1, $2, NULL, 'f', 'application/octet-stream', 1, $3, 'pending', $4, $4)`,
		fileID, userID, storagePath, now,
	)
	if err != nil {
		t.Fatalf("insert file row %s: %v", fileID, err)
	}
}

func assertThumbStatus(t *testing.T, db *pgxpool.Pool, fileID, expected string) {
	t.Helper()
	var status string
	if err := db.QueryRow(context.Background(), `SELECT thumb_status FROM files WHERE id = $1`, fileID).Scan(&status); err != nil {
		t.Fatalf("read thumb_status: %v", err)
	}
	if status != expected {
		t.Fatalf("expected thumb_status=%s got=%s", expected, status)
	}
}

func assertThumbExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("thumbnail file does not exist: %s err=%v", path, err)
	}
}

func writeDummyFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir for source file: %v", err)
	}
	if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
		t.Fatalf("write source file: %v", err)
	}
}

func installFakeMediaTools(t *testing.T, scripts map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range scripts {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0755); err != nil {
			t.Fatalf("write fake tool %s: %v", name, err)
		}
	}
	prevPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+prevPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("PATH", prevPath)
	})
	return dir
}

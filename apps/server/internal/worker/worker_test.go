package worker

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/testutil/dbtest"
)

func TestWorkerBasics(t *testing.T) {
	task, err := NewThumbnailTask("f1", "u1", "u1/f1", "image/png")
	if err != nil {
		t.Fatalf("new thumbnail task: %v", err)
	}
	if task.Type() != TypeThumbnailGenerate {
		t.Fatalf("unexpected task type: %s", task.Type())
	}

	h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
	if h == nil {
		t.Fatal("expected thumbnail handler")
	}

	// invalid payload exits before DB access.
	bad := asynq.NewTask(TypeThumbnailGenerate, []byte("{"))
	if err := h.ProcessTask(context.Background(), bad); err == nil {
		t.Fatal("expected unmarshal payload error")
	}

	if err := runVipsThumbnail("/no/such/src", "/tmp/out", 100); err == nil {
		t.Fatal("expected vipsthumbnail command error")
	}
	if err := h.generateImageThumbnails("/no/such/src", "/tmp/s", "/tmp/m"); err == nil {
		t.Fatal("expected image thumbnail generation error")
	}
	if err := h.generateVideoThumbnails("/no/such/src", "/tmp/s", "/tmp/m"); err == nil {
		t.Fatal("expected video thumbnail generation error")
	}

	// exiftool not found or no output path should be no-op.
	h.extractExifTakenAt(context.Background(), "/no/such/src", "f1")
}

func TestWorkerServerHelpers(t *testing.T) {
	if srv := StartWorkerServer("://invalid", nil, "/tmp/data", "/tmp/thumbs"); srv != nil {
		t.Fatal("expected nil worker server for invalid redis URL")
	}
	if client := NewAsynqClient("://invalid"); client != nil {
		t.Fatal("expected nil client for invalid redis URL")
	}
}

func TestWorkerInjectedErrorBranches(t *testing.T) {
	t.Run("new_thumbnail_task_marshal_error", func(t *testing.T) {
		prev := marshalThumbnailPayload
		marshalThumbnailPayload = func(any) ([]byte, error) { return nil, errors.New("marshal failed") }
		t.Cleanup(func() { marshalThumbnailPayload = prev })

		if _, err := NewThumbnailTask("f1", "u1", "u1/f1", "image/png"); err == nil {
			t.Fatal("expected marshal error")
		}
	})

	t.Run("generate_image_medium_error", func(t *testing.T) {
		prev := runThumbnailFn
		calls := 0
		runThumbnailFn = func(string, string, int) error {
			calls++
			if calls == 2 {
				return errors.New("medium fail")
			}
			return nil
		}
		t.Cleanup(func() { runThumbnailFn = prev })

		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		if err := h.generateImageThumbnails("src", "small", "medium"); err == nil {
			t.Fatal("expected medium thumbnail error")
		}
	})

	t.Run("generate_video_medium_error", func(t *testing.T) {
		prevRun := runThumbnailFn
		prevCmd := execCommand
		calls := 0
		runThumbnailFn = func(string, string, int) error {
			calls++
			if calls == 2 {
				return errors.New("medium fail")
			}
			return nil
		}
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 0")
		}
		t.Cleanup(func() {
			runThumbnailFn = prevRun
			execCommand = prevCmd
		})

		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		if err := h.generateVideoThumbnails("src", "small", "medium"); err == nil {
			t.Fatal("expected medium video thumbnail error")
		}
	})

	t.Run("generate_video_small_error", func(t *testing.T) {
		prevRun := runThumbnailFn
		prevCmd := execCommand
		runThumbnailFn = func(string, string, int) error { return errors.New("small fail") }
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 0")
		}
		t.Cleanup(func() {
			runThumbnailFn = prevRun
			execCommand = prevCmd
		})

		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		if err := h.generateVideoThumbnails("src", "small", "medium"); err == nil {
			t.Fatal("expected small video thumbnail error")
		}
	})

	t.Run("extract_exif_empty_output", func(t *testing.T) {
		prevCmd := execCommand
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "printf ''")
		}
		t.Cleanup(func() { execCommand = prevCmd })

		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		h.extractExifTakenAt(context.Background(), "src", "f1")
	})

	t.Run("extract_exif_whitespace_output", func(t *testing.T) {
		prevCmd := execCommand
		execCommand = func(name string, args ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "printf '   '")
		}
		t.Cleanup(func() { execCommand = prevCmd })

		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		h.extractExifTakenAt(context.Background(), "src", "f1")
	})

	t.Run("marshal_default_contract", func(t *testing.T) {
		payload, err := marshalThumbnailPayload(ThumbnailPayload{FileID: "f1"})
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		var decoded ThumbnailPayload
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if decoded.FileID != "f1" {
			t.Fatalf("unexpected payload: %#v", decoded)
		}
	})
}

func TestThumbnailHandlerProcessTaskStatusErrors(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	db.Close()
	h := NewThumbnailHandler(db, t.TempDir(), t.TempDir())

	task, err := NewThumbnailTask("f1", "u1", "u1/f1", "image/png")
	if err != nil {
		t.Fatalf("new thumbnail task: %v", err)
	}
	if err := h.ProcessTask(context.Background(), task); err == nil {
		t.Fatal("expected update status error on invalid pool")
	}
}

func TestThumbnailHandlerProcessTaskHookedBranches(t *testing.T) {
	t.Run("final_done_status_error", func(t *testing.T) {
		prevExec := execThumbnailSQL
		prevRun := runThumbnailFn
		execCalls := 0
		execThumbnailSQL = func(context.Context, *pgxpool.Pool, string, ...any) error {
			execCalls++
			if execCalls == 2 {
				return errors.New("done update fail")
			}
			return nil
		}
		runThumbnailFn = func(string, string, int) error { return nil }
		t.Cleanup(func() {
			execThumbnailSQL = prevExec
			runThumbnailFn = prevRun
		})

		task, err := NewThumbnailTask("f1", "u1", "u1/f1", "image/png")
		if err != nil {
			t.Fatalf("new thumbnail task: %v", err)
		}
		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		if err := h.ProcessTask(context.Background(), task); err == nil || !strings.Contains(err.Error(), "update status done") {
			t.Fatalf("expected final done status error, got %v", err)
		}
	})

	t.Run("non_media_marks_done_without_error", func(t *testing.T) {
		prevExec := execThumbnailSQL
		execCalls := 0
		execThumbnailSQL = func(context.Context, *pgxpool.Pool, string, ...any) error {
			execCalls++
			return nil
		}
		t.Cleanup(func() { execThumbnailSQL = prevExec })

		task, err := NewThumbnailTask("f1", "u1", "u1/f1", "application/pdf")
		if err != nil {
			t.Fatalf("new thumbnail task: %v", err)
		}
		h := NewThumbnailHandler(nil, t.TempDir(), t.TempDir())
		if err := h.ProcessTask(context.Background(), task); err != nil {
			t.Fatalf("expected non-media success, got %v", err)
		}
		if execCalls != 2 {
			t.Fatalf("expected processing+done status updates, got %d", execCalls)
		}
	})
}

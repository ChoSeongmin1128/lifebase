package asynq

import (
	"context"
	"errors"
	"testing"

	asynqlib "github.com/hibiken/asynq"

	portout "lifebase/internal/cloud/port/out"
)

func TestThumbnailQueueEnqueue(t *testing.T) {
	q := NewThumbnailQueue(nil)
	if err := q.EnqueueThumbnail(context.Background(), portout.ThumbnailTask{
		FileID:      "f1",
		UserID:      "u1",
		StoragePath: "u1/f1",
		MimeType:    "image/png",
	}); err != nil {
		t.Fatalf("nil client should noop without error: %v", err)
	}

	client := asynqlib.NewClient(asynqlib.RedisClientOpt{Addr: "127.0.0.1:0"})
	defer client.Close()
	q = NewThumbnailQueue(client)
	if err := q.EnqueueThumbnail(context.Background(), portout.ThumbnailTask{
		FileID:      "f2",
		UserID:      "u1",
		StoragePath: "u1/f2",
		MimeType:    "image/png",
	}); err == nil {
		t.Fatal("expected enqueue error with invalid redis addr")
	}
}

func TestThumbnailQueueEnqueueTaskBuildError(t *testing.T) {
	prev := newThumbnailTask
	newThumbnailTask = func(string, string, string, string) (*asynqlib.Task, error) {
		return nil, errors.New("build fail")
	}
	t.Cleanup(func() { newThumbnailTask = prev })

	client := asynqlib.NewClient(asynqlib.RedisClientOpt{Addr: "127.0.0.1:0"})
	defer client.Close()

	q := NewThumbnailQueue(client)
	if err := q.EnqueueThumbnail(context.Background(), portout.ThumbnailTask{
		FileID:      "f3",
		UserID:      "u1",
		StoragePath: "u1/f3",
		MimeType:    "image/png",
	}); err == nil {
		t.Fatal("expected task build error")
	}
}

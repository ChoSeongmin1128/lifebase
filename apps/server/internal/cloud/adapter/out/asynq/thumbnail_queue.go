package asynq

import (
	"context"

	"github.com/hibiken/asynq"

	portout "lifebase/internal/cloud/port/out"
	"lifebase/internal/worker"
)

var newThumbnailTask = worker.NewThumbnailTask

type ThumbnailQueue struct {
	client *asynq.Client
}

func NewThumbnailQueue(client *asynq.Client) *ThumbnailQueue {
	return &ThumbnailQueue{client: client}
}

func (q *ThumbnailQueue) EnqueueThumbnail(_ context.Context, task portout.ThumbnailTask) error {
	if q.client == nil {
		return nil
	}

	t, err := newThumbnailTask(task.FileID, task.UserID, task.StoragePath, task.MimeType)
	if err != nil {
		return err
	}

	_, err = q.client.Enqueue(t, asynq.Queue("thumbnails"))
	return err
}

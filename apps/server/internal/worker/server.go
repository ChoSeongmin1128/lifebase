package worker

import (
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
)

func StartWorkerServer(redisURL string, db *pgxpool.Pool, dataPath, thumbPath string) *asynq.Server {
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		slog.Error("failed to parse redis URL for worker", "error", err)
		return nil
	}

	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency: 4,
		Queues: map[string]int{
			"default":    6,
			"thumbnails": 4,
		},
	})

	mux := asynq.NewServeMux()
	thumbHandler := NewThumbnailHandler(db, dataPath, thumbPath)
	mux.HandleFunc(TypeThumbnailGenerate, thumbHandler.ProcessTask)

	go func() {
		if err := srv.Run(mux); err != nil {
			slog.Error("worker server error", "error", err)
		}
	}()

	slog.Info("worker server started", "concurrency", 4)
	return srv
}

func NewAsynqClient(redisURL string) *asynq.Client {
	redisOpt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		slog.Error("failed to parse redis URL for asynq client", "error", err)
		return nil
	}
	return asynq.NewClient(redisOpt)
}

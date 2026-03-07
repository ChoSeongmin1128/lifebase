package worker

import (
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

func TestStartWorkerServerValidRedisURL(t *testing.T) {
	srv := StartWorkerServer("redis://127.0.0.1:0/0", nil, t.TempDir(), t.TempDir())
	if srv == nil {
		t.Fatal("expected non-nil server for valid redis url")
	}
	time.Sleep(150 * time.Millisecond)
	srv.Shutdown()
}

func TestNewAsynqClientValidRedisURL(t *testing.T) {
	client := NewAsynqClient("redis://127.0.0.1:6379/0")
	if client == nil {
		t.Fatal("expected non-nil asynq client for valid redis url")
	}
	_ = client.Close()
}

func TestStartWorkerServerRunErrorBranch(t *testing.T) {
	prev := runAsynqServer
	t.Cleanup(func() { runAsynqServer = prev })

	done := make(chan struct{}, 1)
	runAsynqServer = func(srv *asynq.Server, mux *asynq.ServeMux) error {
		done <- struct{}{}
		return errors.New("run failed")
	}

	srv := StartWorkerServer("redis://127.0.0.1:0/0", nil, t.TempDir(), t.TempDir())
	if srv == nil {
		t.Fatal("expected non-nil server for valid redis url")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected run hook to be called")
	}

	srv.Shutdown()
}

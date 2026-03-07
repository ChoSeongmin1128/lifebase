package main

import (
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMainBootAndGracefulShutdown(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("REDIS_URL", "://invalid")
	t.Setenv("SERVER_PORT", "0")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("STORAGE_DATA_PATH", t.TempDir())
	t.Setenv("STORAGE_THUMB_PATH", t.TempDir())
	t.Setenv("WEB_URL", "http://localhost:39001")
	t.Setenv("ADMIN_URL", "http://localhost:39002")
	t.Setenv("API_URL", "http://localhost:38117")

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	time.Sleep(200 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGINT); err != nil {
		t.Fatalf("send SIGINT: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("main did not exit after SIGINT")
	}
}

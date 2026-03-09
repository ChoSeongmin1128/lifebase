package main

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"lifebase/internal/shared/config"
)

func TestRunSuccess(t *testing.T) {
	prevLoad, prevExec := loadConfigFn, execCommandFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		execCommandFn = prevExec
	})

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
	}
	execCommandFn = func(name string, args ...string) *exec.Cmd {
		if name != "pg_restore" {
			t.Fatalf("unexpected command: %s", name)
		}
		if len(args) != 6 || args[0] != "--clean" || args[1] != "--if-exists" || args[2] != "--no-owner" || args[3] != "--dbname" || !strings.Contains(args[4], "lifebase_dev") || args[5] != "/tmp/backup.dump" {
			t.Fatalf("unexpected args: %#v", args)
		}
		return exec.Command("sh", "-c", "exit 0")
	}

	if err := run([]string{"--file", "/tmp/backup.dump"}); err != nil {
		t.Fatalf("run restore: %v", err)
	}
}

func TestRunErrorBranches(t *testing.T) {
	prevLoad, prevExec := loadConfigFn, execCommandFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		execCommandFn = prevExec
	})

	t.Run("load_config_error", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return nil, errors.New("load fail") }
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "load config") {
			t.Fatalf("expected load config error, got %v", err)
		}
	})

	t.Run("empty_database_url", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return &config.Config{}, nil }
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "database url is required") {
			t.Fatalf("expected empty database url error, got %v", err)
		}
	})

	t.Run("missing_file", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		if err := run(nil); err == nil || !strings.Contains(err.Error(), "backup file path is required") {
			t.Fatalf("expected missing file error, got %v", err)
		}
	})

	t.Run("flag_parse_error", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		if err := run([]string{"--unknown"}); err == nil {
			t.Fatal("expected flag parse error")
		}
	})

	t.Run("pg_restore_error", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		execCommandFn = func(string, ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 9")
		}
		if err := run([]string{"--file", "/tmp/backup.dump"}); err == nil || !strings.Contains(err.Error(), "pg_restore") {
			t.Fatalf("expected pg_restore error, got %v", err)
		}
	})
}

func TestMainBranches(t *testing.T) {
	prevRun, prevExit, prevStderr := runFn, exitFn, stderrWriter
	t.Cleanup(func() {
		runFn = prevRun
		exitFn = prevExit
		stderrWriter = prevStderr
	})

	t.Run("success", func(t *testing.T) {
		called := false
		runFn = func(args []string) error {
			called = true
			return nil
		}
		exitFn = func(int) { t.Fatal("exit should not be called") }
		main()
		if !called {
			t.Fatal("expected run to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		capturedCode := 0
		var stderr strings.Builder
		runFn = func(args []string) error { return errors.New("boom") }
		exitFn = func(code int) { capturedCode = code }
		stderrWriter = &stderr
		main()
		if capturedCode != 1 {
			t.Fatalf("expected exit code 1, got %d", capturedCode)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Fatalf("expected stderr to contain error, got %q", stderr.String())
		}
	})
}

package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/dbbackup"
)

func TestRunSuccess(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevFind := recentBackupFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		recentBackupFn = prevFind
	})

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase?sslmode=disable"}}, nil
	}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }
	called := false
	recentBackupFn = func(root, dbName string, at time.Time, window time.Duration) (string, time.Time, bool, error) {
		called = true
		if root != dbbackup.DefaultBackupRoot || dbName != "lifebase" || at != now || window != dbbackup.DefaultRecentWindow {
			t.Fatalf("unexpected args root=%s db=%s at=%s window=%s", root, dbName, at, window)
		}
		return "/Volumes/WDRedPlus/LifeBase/backups/hourly/lifebase-20260309-120000.dump", now.Add(-time.Hour), true, nil
	}
	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if !called {
		t.Fatal("expected recent backup lookup")
	}
}

func TestRunErrorBranches(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevFind := recentBackupFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		recentBackupFn = prevFind
	})

	t.Run("load_config_error", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return nil, errors.New("load fail") }
		if err := run(); err == nil || !strings.Contains(err.Error(), "load config") {
			t.Fatalf("expected load error, got %v", err)
		}
	})

	t.Run("empty_database_url", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return &config.Config{}, nil }
		if err := run(); err == nil || !strings.Contains(err.Error(), "database url is required") {
			t.Fatalf("expected empty db url error, got %v", err)
		}
	})

	t.Run("parse_error_and_skip_non_operational", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "://bad"}}, nil
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "parse database url") {
			t.Fatalf("expected parse error, got %v", err)
		}
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		if err := run(); err != nil {
			t.Fatalf("expected non-operational db skip, got %v", err)
		}
	})

	t.Run("find_recent_errors_and_missing", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase?sslmode=disable"}}, nil
		}
		recentBackupFn = func(string, string, time.Time, time.Duration) (string, time.Time, bool, error) {
			return "", time.Time{}, false, errors.New("find fail")
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "find fail") {
			t.Fatalf("expected find error, got %v", err)
		}
		recentBackupFn = func(string, string, time.Time, time.Duration) (string, time.Time, bool, error) {
			return "", time.Time{}, false, nil
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "recent operational backup not found") {
			t.Fatalf("expected missing backup error, got %v", err)
		}
		recentBackupFn = func(string, string, time.Time, time.Duration) (string, time.Time, bool, error) {
			return "/x/lifebase-20260309-000000.dump", time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), false, nil
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "recent operational backup missing") {
			t.Fatalf("expected stale backup error, got %v", err)
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
		runFn = func() error {
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
		runFn = func() error { return errors.New("boom") }
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

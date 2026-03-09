package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lifebase/internal/shared/config"
	"lifebase/internal/shared/dbbackup"
)

type exitCapture struct {
	code int
}

func TestRunSuccess(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevDump := backupDumpFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		backupDumpFn = prevDump
	})

	tmpDir := t.TempDir()
	t.Setenv("DB_BACKUP_DIR", tmpDir)

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{
			Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"},
		}, nil
	}
	nowFn = func() time.Time { return time.Date(2026, 3, 9, 12, 0, 0, 0, time.FixedZone("KST", 9*60*60)) }
	backupDumpFn = func(databaseURL, backupDir string, now time.Time, stdout, stderr io.Writer) (string, error) {
		if databaseURL != "postgres://user@localhost:5432/lifebase_dev?sslmode=disable" {
			t.Fatalf("unexpected database url: %s", databaseURL)
		}
		if backupDir != tmpDir {
			t.Fatalf("unexpected backup dir: %s", backupDir)
		}
		return filepath.Join(tmpDir, "lifebase_dev-20260309-120000.dump"), nil
	}

	path, err := run()
	if err != nil {
		t.Fatalf("run backup: %v", err)
	}
	if path != filepath.Join(tmpDir, "lifebase_dev-20260309-120000.dump") {
		t.Fatalf("unexpected backup path: %s", path)
	}
}

func TestRunErrorBranches(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevDump := backupDumpFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		backupDumpFn = prevDump
	})

	t.Run("load_config_error", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return nil, errors.New("load fail") }
		if _, err := run(); err == nil || !strings.Contains(err.Error(), "load config") {
			t.Fatalf("expected load config error, got %v", err)
		}
	})

	t.Run("empty_database_url", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) { return &config.Config{}, nil }
		if _, err := run(); err == nil || !strings.Contains(err.Error(), "database url is required") {
			t.Fatalf("expected empty database url error, got %v", err)
		}
	})

	t.Run("invalid_database_url", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "://bad"}}, nil
		}
		backupDumpFn = func(string, string, time.Time, io.Writer, io.Writer) (string, error) {
			return "", errors.New("parse database url: bad")
		}
		if _, err := run(); err == nil || !strings.Contains(err.Error(), "parse database url") {
			t.Fatalf("expected parse database url error, got %v", err)
		}
	})

	t.Run("dump_error_and_default_manual_dir", func(t *testing.T) {
		_ = os.Unsetenv("DB_BACKUP_DIR")
		tmpDir := t.TempDir()
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		nowFn = func() time.Time { return time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC) }
		backupDumpFn = func(databaseURL, backupDir string, now time.Time, stdout, stderr io.Writer) (string, error) {
			if backupDir != filepath.Join(dbbackup.DefaultBackupRoot, dbbackup.ManualDirName) {
				t.Fatalf("expected manual backup dir, got %s", backupDir)
			}
			_ = tmpDir
			return "", errors.New("pg_dump: exit status 12")
		}
		if _, err := run(); err == nil || !strings.Contains(err.Error(), "pg_dump") {
			t.Fatalf("expected pg_dump error, got %v", err)
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
		runFn = func() (string, error) {
			called = true
			return "/tmp/ok.dump", nil
		}
		exitFn = func(int) { t.Fatal("exit should not be called") }
		main()
		if !called {
			t.Fatal("expected run to be called")
		}
	})

	t.Run("error", func(t *testing.T) {
		capture := &exitCapture{}
		var stderr strings.Builder
		runFn = func() (string, error) { return "", errors.New("boom") }
		exitFn = func(code int) { capture.code = code }
		stderrWriter = &stderr
		main()
		if capture.code != 1 {
			t.Fatalf("expected exit code 1, got %d", capture.code)
		}
		if !strings.Contains(stderr.String(), "boom") {
			t.Fatalf("expected stderr to contain error, got %q", stderr.String())
		}
	})
}

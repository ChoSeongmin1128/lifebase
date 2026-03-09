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

func TestRunSuccess(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevDump := backupDumpFn
	prevCopy := copyIntoCategoriesFn
	prevRotate := rotateBackupDirFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		backupDumpFn = prevDump
		copyIntoCategoriesFn = prevCopy
		rotateBackupDirFn = prevRotate
	})

	root := t.TempDir()
	t.Setenv("DB_BACKUP_ROOT", root)
	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase?sslmode=disable"}}, nil
	}
	now := time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)
	nowFn = func() time.Time { return now }

	var dumpDir string
	backupDumpFn = func(databaseURL, backupDir string, when time.Time, stdout, stderr io.Writer) (string, error) {
		dumpDir = backupDir
		if databaseURL == "" || when != now {
			t.Fatalf("unexpected dump args: url=%s when=%s", databaseURL, when)
		}
		return filepath.Join(backupDir, "lifebase-20260309-000000.dump"), nil
	}
	var copiedCategories []string
	copyIntoCategoriesFn = func(sourcePath, root string, categories []string) ([]string, error) {
		copiedCategories = categories
		return []string{sourcePath, filepath.Join(root, dbbackup.DailyDirName, "lifebase-20260309-000000.dump"), filepath.Join(root, dbbackup.WeeklyDirName, "lifebase-20260309-000000.dump")}, nil
	}
	rotated := map[string]bool{}
	rotateBackupDirFn = func(dir, dbName string, keep int) ([]string, error) {
		rotated[filepath.Base(dir)] = true
		return nil, nil
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if dumpDir != filepath.Join(root, dbbackup.HourlyDirName) {
		t.Fatalf("unexpected dump dir: %s", dumpDir)
	}
	if strings.Join(copiedCategories, ",") != "hourly,daily,weekly" {
		t.Fatalf("unexpected copied categories: %#v", copiedCategories)
	}
	if !rotated[dbbackup.HourlyDirName] || !rotated[dbbackup.DailyDirName] || !rotated[dbbackup.WeeklyDirName] {
		t.Fatalf("expected rotate to run for all categories, got %#v", rotated)
	}
}

func TestRunErrorBranches(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevDump := backupDumpFn
	prevCopy := copyIntoCategoriesFn
	prevRotate := rotateBackupDirFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		backupDumpFn = prevDump
		copyIntoCategoriesFn = prevCopy
		rotateBackupDirFn = prevRotate
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

	t.Run("invalid_database_name", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "://bad"}}, nil
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "parse database url") {
			t.Fatalf("expected parse error, got %v", err)
		}
	})

	t.Run("non_operational_db", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase_dev?sslmode=disable"}}, nil
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "automatic backup only supports") {
			t.Fatalf("expected operational db error, got %v", err)
		}
	})

	t.Run("dump_copy_rotate_errors", func(t *testing.T) {
		loadConfigFn = func() (*config.Config, error) {
			return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase?sslmode=disable"}}, nil
		}
		nowFn = func() time.Time { return time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC) }
		root := t.TempDir()
		t.Setenv("DB_BACKUP_ROOT", root)
		backupDumpFn = func(string, string, time.Time, io.Writer, io.Writer) (string, error) {
			return "", errors.New("dump fail")
		}
		if err := run(); err == nil || !strings.Contains(err.Error(), "dump fail") {
			t.Fatalf("expected dump error, got %v", err)
		}
		backupDumpFn = func(string, string, time.Time, io.Writer, io.Writer) (string, error) {
			return filepath.Join(root, dbbackup.HourlyDirName, "lifebase-20260309-060000.dump"), nil
		}
		copyIntoCategoriesFn = func(string, string, []string) ([]string, error) { return nil, errors.New("copy fail") }
		if err := run(); err == nil || !strings.Contains(err.Error(), "copy fail") {
			t.Fatalf("expected copy error, got %v", err)
		}
		copyIntoCategoriesFn = func(sourcePath, root string, categories []string) ([]string, error) { return []string{sourcePath}, nil }
		rotateBackupDirFn = func(string, string, int) ([]string, error) { return nil, errors.New("rotate fail") }
		if err := run(); err == nil || !strings.Contains(err.Error(), "rotate fail") {
			t.Fatalf("expected rotate error, got %v", err)
		}
	})
}

func TestJoinPathsAndMainBranches(t *testing.T) {
	if got := joinPaths(nil); got != "" {
		t.Fatalf("expected empty join result, got %q", got)
	}

	prevRun, prevExit, prevStderr := runFn, exitFn, stderrWriter
	t.Cleanup(func() {
		runFn = prevRun
		exitFn = prevExit
		stderrWriter = prevStderr
	})

	t.Run("main_success", func(t *testing.T) {
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

	t.Run("main_error", func(t *testing.T) {
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

func TestRunUsesDefaultRootAndLogsRotations(t *testing.T) {
	prevLoad, prevNow := loadConfigFn, nowFn
	prevDump := backupDumpFn
	prevCopy := copyIntoCategoriesFn
	prevRotate := rotateBackupDirFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		nowFn = prevNow
		backupDumpFn = prevDump
		copyIntoCategoriesFn = prevCopy
		rotateBackupDirFn = prevRotate
	})

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Database: config.DatabaseConfig{URL: "postgres://user@localhost:5432/lifebase?sslmode=disable"}}, nil
	}
	nowFn = func() time.Time { return time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC) }
	_ = os.Unsetenv("DB_BACKUP_ROOT")
	backupDumpFn = func(databaseURL, backupDir string, when time.Time, stdout, stderr io.Writer) (string, error) {
		if backupDir != filepath.Join(dbbackup.DefaultBackupRoot, dbbackup.HourlyDirName) {
			t.Fatalf("expected default hourly dir, got %s", backupDir)
		}
		return filepath.Join(backupDir, "lifebase-20260309-060000.dump"), nil
	}
	copyIntoCategoriesFn = func(sourcePath, root string, categories []string) ([]string, error) {
		return []string{sourcePath}, nil
	}
	rotateCalls := 0
	rotateBackupDirFn = func(dir, dbName string, keep int) ([]string, error) {
		rotateCalls++
		if filepath.Base(dir) == dbbackup.HourlyDirName {
			return []string{filepath.Join(dir, "lifebase-20260308-000000.dump")}, nil
		}
		return nil, nil
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if rotateCalls != 3 {
		t.Fatalf("expected rotate calls for 3 categories, got %d", rotateCalls)
	}
}

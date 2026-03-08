package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"lifebase/internal/shared/config"
)

func TestMainSuccess(t *testing.T) {
	dataDir := t.TempDir()
	thumbDir := t.TempDir()

	// Prepare one empty nested dir under each root to ensure "removed" output is printed.
	if err := os.MkdirAll(filepath.Join(dataDir, "u1", "empty"), 0o755); err != nil {
		t.Fatalf("mkdir data empty dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(thumbDir, "u1", "empty"), 0o755); err != nil {
		t.Fatalf("mkdir thumb empty dir: %v", err)
	}

	code, stdout, stderr := runMainSubprocess(t, map[string]string{
		"STORAGE_DATA_PATH":  dataDir,
		"STORAGE_THUMB_PATH": thumbDir,
	})
	if code != 0 {
		t.Fatalf("expected success exit code, got %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "data_empty_dirs_removed=") {
		t.Fatalf("missing data removed output: %s", stdout)
	}
	if !strings.Contains(stdout, "thumb_empty_dirs_removed=") {
		t.Fatalf("missing thumb removed output: %s", stdout)
	}
}

func TestRunSuccess(t *testing.T) {
	prevLoad := loadConfigFn
	prevRemove := removeEmptyDirsFn
	prevStdout := os.Stdout
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		removeEmptyDirsFn = prevRemove
		os.Stdout = prevStdout
	})

	var buf bytes.Buffer
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Storage: config.StorageConfig{DataPath: "/data", ThumbPath: "/thumb"}}, nil
	}
	calls := []string{}
	removeEmptyDirsFn = func(path string) (int, error) {
		calls = append(calls, path)
		if path == "/data" {
			return 2, nil
		}
		return 3, nil
	}

	if err := run(); err != nil {
		t.Fatalf("run: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if got := strings.Join(calls, ","); got != "/data,/thumb" {
		t.Fatalf("unexpected cleanup order: %s", got)
	}
	output := buf.String()
	if !strings.Contains(output, "data_empty_dirs_removed=2") || !strings.Contains(output, "thumb_empty_dirs_removed=3") {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestRunLoadConfigError(t *testing.T) {
	prevLoad := loadConfigFn
	t.Cleanup(func() { loadConfigFn = prevLoad })

	loadConfigFn = func() (*config.Config, error) {
		return nil, errors.New("boom")
	}

	if err := run(); err == nil || !strings.Contains(err.Error(), "load config: boom") {
		t.Fatalf("expected wrapped config error, got %v", err)
	}
}

func TestRunDataCleanupError(t *testing.T) {
	prevLoad := loadConfigFn
	prevRemove := removeEmptyDirsFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		removeEmptyDirsFn = prevRemove
	})

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Storage: config.StorageConfig{DataPath: "/data", ThumbPath: "/thumb"}}, nil
	}
	removeEmptyDirsFn = func(path string) (int, error) {
		if path == "/data" {
			return 0, errors.New("data fail")
		}
		return 0, nil
	}

	if err := run(); err == nil || !strings.Contains(err.Error(), "cleanup data dirs: data fail") {
		t.Fatalf("expected wrapped data cleanup error, got %v", err)
	}
}

func TestRunThumbCleanupError(t *testing.T) {
	prevLoad := loadConfigFn
	prevRemove := removeEmptyDirsFn
	t.Cleanup(func() {
		loadConfigFn = prevLoad
		removeEmptyDirsFn = prevRemove
	})

	loadConfigFn = func() (*config.Config, error) {
		return &config.Config{Storage: config.StorageConfig{DataPath: "/data", ThumbPath: "/thumb"}}, nil
	}
	removeEmptyDirsFn = func(path string) (int, error) {
		if path == "/thumb" {
			return 0, errors.New("thumb fail")
		}
		return 1, nil
	}

	if err := run(); err == nil || !strings.Contains(err.Error(), "cleanup thumb dirs: thumb fail") {
		t.Fatalf("expected wrapped thumb cleanup error, got %v", err)
	}
}

func TestMainFailsWhenDataCleanupErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on windows")
	}

	dataDir := t.TempDir()
	thumbDir := t.TempDir()
	locked := filepath.Join(dataDir, "locked")
	child := filepath.Join(locked, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir locked child: %v", err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	code, _, stderr := runMainSubprocess(t, map[string]string{
		"STORAGE_DATA_PATH":  dataDir,
		"STORAGE_THUMB_PATH": thumbDir,
	})
	if code == 0 {
		t.Fatalf("expected non-zero exit code, stderr=%s", stderr)
	}
	if !strings.Contains(stderr, "cleanup data dirs:") {
		t.Fatalf("expected data cleanup error message, stderr=%s", stderr)
	}
}

func TestMainFailsWhenThumbCleanupErrors(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on windows")
	}

	dataDir := t.TempDir()
	thumbDir := t.TempDir()
	locked := filepath.Join(thumbDir, "locked")
	child := filepath.Join(locked, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir locked child: %v", err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

	code, _, stderr := runMainSubprocess(t, map[string]string{
		"STORAGE_DATA_PATH":  dataDir,
		"STORAGE_THUMB_PATH": thumbDir,
	})
	if code == 0 {
		t.Fatalf("expected non-zero exit code, stderr=%s", stderr)
	}
	if !strings.Contains(stderr, "cleanup thumb dirs:") {
		t.Fatalf("expected thumb cleanup error message, stderr=%s", stderr)
	}
}

func TestHelperMainProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	main()
	os.Exit(0)
}

func runMainSubprocess(t *testing.T, env map[string]string) (exitCode int, stdout string, stderr string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=TestHelperMainProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()

	if err == nil {
		return 0, string(out), ""
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("unexpected subprocess error: %v", err)
	}
	return exitErr.ExitCode(), "", string(out)
}

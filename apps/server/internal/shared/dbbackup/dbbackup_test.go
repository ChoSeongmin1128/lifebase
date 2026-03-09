package dbbackup

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type stubDirEntry struct {
	name  string
	isDir bool
	info  fs.FileInfo
	err   error
}

func (s stubDirEntry) Name() string               { return s.name }
func (s stubDirEntry) IsDir() bool                { return s.isDir }
func (s stubDirEntry) Type() fs.FileMode          { return 0 }
func (s stubDirEntry) Info() (fs.FileInfo, error) { return s.info, s.err }

func TestDatabaseNameFromDSNAndBackupFileName(t *testing.T) {
	name, err := DatabaseNameFromDSN("postgres://user@localhost:5432/lifebase?sslmode=disable")
	if err != nil || name != "lifebase" {
		t.Fatalf("expected lifebase, got name=%q err=%v", name, err)
	}
	if _, err := DatabaseNameFromDSN("postgres://user@localhost:5432/?sslmode=disable"); err == nil {
		t.Fatal("expected empty database name error")
	}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	if got := BackupFileName("lifebase", now); got != "lifebase-20260309-120000.dump" {
		t.Fatalf("unexpected backup filename: %s", got)
	}
}

func TestCategoryDirs(t *testing.T) {
	if got := CategoryDirs(time.Date(2026, 3, 9, 6, 0, 0, 0, time.UTC)); len(got) != 1 || got[0] != HourlyDirName {
		t.Fatalf("unexpected hourly categories: %#v", got)
	}
	if got := CategoryDirs(time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)); strings.Join(got, ",") != "hourly,daily" {
		t.Fatalf("unexpected daily categories: %#v", got)
	}
	if got := CategoryDirs(time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC)); strings.Join(got, ",") != "hourly,daily,weekly" {
		t.Fatalf("unexpected weekly categories: %#v", got)
	}
}

func TestDump(t *testing.T) {
	prevExec, prevMkdir, prevRemove := execCommandFn, mkdirAllFn, removeFileFn
	t.Cleanup(func() {
		execCommandFn = prevExec
		mkdirAllFn = prevMkdir
		removeFileFn = prevRemove
	})

	t.Run("success", func(t *testing.T) {
		tmpDir := t.TempDir()
		mkdirAllFn = func(path string, perm os.FileMode) error {
			if path != tmpDir || perm != 0o755 {
				t.Fatalf("unexpected mkdir args path=%s perm=%o", path, perm)
			}
			return nil
		}
		execCommandFn = func(name string, args ...string) *exec.Cmd {
			if name != "pg_dump" {
				t.Fatalf("unexpected command: %s", name)
			}
			wantFile := filepath.Join(tmpDir, "lifebase-20260309-120000.dump")
			if len(args) != 4 || args[2] != wantFile || !strings.Contains(args[3], "lifebase") {
				t.Fatalf("unexpected args: %#v", args)
			}
			return exec.Command("sh", "-c", "exit 0")
		}
		path, err := Dump("postgres://user@localhost:5432/lifebase?sslmode=disable", tmpDir, time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), os.Stdout, os.Stderr)
		if err != nil {
			t.Fatalf("dump success err: %v", err)
		}
		if path != filepath.Join(tmpDir, "lifebase-20260309-120000.dump") {
			t.Fatalf("unexpected dump path: %s", path)
		}
	})

	t.Run("cleanup_partial_file_on_pg_dump_error", func(t *testing.T) {
		tmpDir := t.TempDir()
		var removed string
		mkdirAllFn = func(string, os.FileMode) error { return nil }
		removeFileFn = func(path string) error {
			removed = path
			return nil
		}
		execCommandFn = func(string, ...string) *exec.Cmd {
			return exec.Command("sh", "-c", "exit 9")
		}
		_, err := Dump("postgres://user@localhost:5432/lifebase?sslmode=disable", tmpDir, time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), os.Stdout, os.Stderr)
		if err == nil || !strings.Contains(err.Error(), "pg_dump") {
			t.Fatalf("expected pg_dump error, got %v", err)
		}
		if removed != filepath.Join(tmpDir, "lifebase-20260309-120000.dump") {
			t.Fatalf("expected partial file cleanup, got %s", removed)
		}
	})

	t.Run("validation_and_mkdir_error", func(t *testing.T) {
		if _, err := Dump("", t.TempDir(), time.Now(), os.Stdout, os.Stderr); err == nil {
			t.Fatal("expected empty database url error")
		}
		if _, err := Dump("://bad", t.TempDir(), time.Now(), os.Stdout, os.Stderr); err == nil {
			t.Fatal("expected parse database url error")
		}
		mkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir fail") }
		if _, err := Dump("postgres://user@localhost:5432/lifebase?sslmode=disable", t.TempDir(), time.Now(), os.Stdout, os.Stderr); err == nil || !strings.Contains(err.Error(), "create backup dir") {
			t.Fatalf("expected mkdir error, got %v", err)
		}
	})

	t.Run("default_manual_dir", func(t *testing.T) {
		expectedDir := filepath.Join(DefaultBackupRoot, ManualDirName)
		mkdirAllFn = func(path string, perm os.FileMode) error {
			if path != expectedDir || perm != 0o755 {
				t.Fatalf("unexpected mkdir args path=%s perm=%o", path, perm)
			}
			return nil
		}
		execCommandFn = func(name string, args ...string) *exec.Cmd {
			if args[2] != filepath.Join(expectedDir, "lifebase-20260309-120000.dump") {
				t.Fatalf("unexpected dump output path: %#v", args)
			}
			return exec.Command("sh", "-c", "exit 0")
		}
		if _, err := Dump("postgres://user@localhost:5432/lifebase?sslmode=disable", "", time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), os.Stdout, os.Stderr); err != nil {
			t.Fatalf("expected dump success with default dir, got %v", err)
		}
	})
}

func TestCopyIntoCategoriesAndRotateDir(t *testing.T) {
	prevCopy, prevMkdir, prevRead, prevRemove := copyFileFn, mkdirAllFn, readDirFn, removeFileFn
	t.Cleanup(func() {
		copyFileFn = prevCopy
		mkdirAllFn = prevMkdir
		readDirFn = prevRead
		removeFileFn = prevRemove
	})

	t.Run("copy_categories", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "hourly", "lifebase-20260309-000000.dump")
		var copiedSrc, copiedDst string
		mkdirAllFn = func(string, os.FileMode) error { return nil }
		copyFileFn = func(from, to string) error {
			copiedSrc, copiedDst = from, to
			return nil
		}
		copied, err := CopyIntoCategories(src, root, []string{HourlyDirName, DailyDirName})
		if err != nil {
			t.Fatalf("copy categories: %v", err)
		}
		if len(copied) != 2 || copied[0] != src || copied[1] != filepath.Join(root, DailyDirName, filepath.Base(src)) {
			t.Fatalf("unexpected copied paths: %#v", copied)
		}
		if copiedSrc != src || copiedDst != filepath.Join(root, DailyDirName, filepath.Base(src)) {
			t.Fatalf("unexpected copy args src=%s dst=%s", copiedSrc, copiedDst)
		}
	})

	t.Run("copy_error_branches", func(t *testing.T) {
		root := t.TempDir()
		src := filepath.Join(root, "hourly", "lifebase-20260309-000000.dump")
		mkdirAllFn = func(string, os.FileMode) error { return errors.New("mkdir fail") }
		if _, err := CopyIntoCategories(src, root, []string{DailyDirName}); err == nil || !strings.Contains(err.Error(), "create category dir") {
			t.Fatalf("expected mkdir error, got %v", err)
		}
		mkdirAllFn = func(string, os.FileMode) error { return nil }
		copyFileFn = func(string, string) error { return errors.New("copy fail") }
		if _, err := CopyIntoCategories(src, root, []string{DailyDirName}); err == nil || !strings.Contains(err.Error(), "copy backup to") {
			t.Fatalf("expected copy error, got %v", err)
		}
	})

	t.Run("rotate_dir", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-000000.dump"), []byte("a"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-060000.dump"), []byte("b"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-120000.dump"), []byte("c"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		removed, err := RotateDir(dir, "lifebase", 2)
		if err != nil {
			t.Fatalf("rotate dir: %v", err)
		}
		if len(removed) != 1 || !strings.Contains(removed[0], "000000") {
			t.Fatalf("unexpected removed files: %#v", removed)
		}
	})

	t.Run("rotate_error_branches", func(t *testing.T) {
		if removed, err := RotateDir("/tmp/x", "lifebase", 0); err != nil || removed != nil {
			t.Fatalf("expected keep<=0 no-op, got removed=%#v err=%v", removed, err)
		}
		readDirFn = func(string) ([]os.DirEntry, error) { return nil, os.ErrNotExist }
		if removed, err := RotateDir("/tmp/missing", "lifebase", 1); err != nil || removed != nil {
			t.Fatalf("expected missing dir no-op, got removed=%#v err=%v", removed, err)
		}
		readDirFn = func(string) ([]os.DirEntry, error) { return nil, errors.New("read fail") }
		if _, err := RotateDir("/tmp/x", "lifebase", 1); err == nil || !strings.Contains(err.Error(), "read backup dir") {
			t.Fatalf("expected read dir error, got %v", err)
		}
		readDirFn = prevRead
		removeFileFn = func(string) error { return errors.New("remove fail") }
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-000000.dump"), []byte("a"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-060000.dump"), []byte("b"), 0o644); err != nil {
			t.Fatalf("write file: %v", err)
		}
		if _, err := RotateDir(dir, "lifebase", 1); err == nil || !strings.Contains(err.Error(), "remove old backup") {
			t.Fatalf("expected remove error, got %v", err)
		}
	})

	t.Run("rotate_kept_and_filtered_entries", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o755); err != nil {
			t.Fatalf("mkdir subdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write note: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lifebase-20260309-120000.dump"), []byte("c"), 0o644); err != nil {
			t.Fatalf("write dump: %v", err)
		}
		removed, err := RotateDir(dir, "lifebase", 2)
		if err != nil {
			t.Fatalf("rotate dir: %v", err)
		}
		if removed != nil {
			t.Fatalf("expected no removals when count <= keep, got %#v", removed)
		}
	})
}

func TestFindLatestRecentBackup(t *testing.T) {
	root := t.TempDir()
	dirs := []string{HourlyDirName, DailyDirName, WeeklyDirName, ManualDirName}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	oldPath := filepath.Join(root, ManualDirName, "lifebase-20260309-000000.dump")
	if err := os.WriteFile(oldPath, []byte("old"), 0o644); err != nil {
		t.Fatalf("write old backup: %v", err)
	}
	oldMod := now.Add(-10 * time.Hour)
	if err := os.Chtimes(oldPath, oldMod, oldMod); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}

	latestPath := filepath.Join(root, HourlyDirName, "lifebase-20260309-090000.dump")
	if err := os.WriteFile(latestPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write recent backup: %v", err)
	}
	recentMod := now.Add(-2 * time.Hour)
	if err := os.Chtimes(latestPath, recentMod, recentMod); err != nil {
		t.Fatalf("chtimes recent: %v", err)
	}

	path, modTime, ok, err := FindLatestRecentBackup(root, "lifebase", now, 6*time.Hour)
	if err != nil || !ok || path != latestPath || !modTime.Equal(recentMod) {
		t.Fatalf("unexpected latest recent backup path=%s mod=%s ok=%v err=%v", path, modTime, ok, err)
	}

	path, _, ok, err = FindLatestRecentBackup(root, "lifebase", now.Add(20*time.Hour), 6*time.Hour)
	if err != nil || ok || path != latestPath {
		t.Fatalf("expected stale backup result, got path=%s ok=%v err=%v", path, ok, err)
	}

	emptyRoot := t.TempDir()
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(emptyRoot, dir), 0o755); err != nil {
			t.Fatalf("mkdir empty dir %s: %v", dir, err)
		}
	}
	path, _, ok, err = FindLatestRecentBackup(emptyRoot, "lifebase", now, 6*time.Hour)
	if err != nil || ok || path != "" {
		t.Fatalf("expected no backups, got path=%s ok=%v err=%v", path, ok, err)
	}

	missingRoot := t.TempDir()
	path, _, ok, err = FindLatestRecentBackup(missingRoot, "lifebase", now, 6*time.Hour)
	if err != nil || ok || path != "" {
		t.Fatalf("expected missing directories to behave as no backups, got path=%s ok=%v err=%v", path, ok, err)
	}

	prevRead := readDirFn
	t.Cleanup(func() { readDirFn = prevRead })
	readDirFn = func(string) ([]os.DirEntry, error) { return nil, errors.New("read fail") }
	if _, _, _, err := FindLatestRecentBackup(root, "lifebase", now, 6*time.Hour); err == nil || !strings.Contains(err.Error(), "read backup dir") {
		t.Fatalf("expected read dir error, got %v", err)
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{stubDirEntry{name: "lifebase-20260309-010000.dump", err: errors.New("stat fail")}}, nil
	}
	if _, _, _, err := FindLatestRecentBackup(root, "lifebase", now, 6*time.Hour); err == nil || !strings.Contains(err.Error(), "stat backup file") {
		t.Fatalf("expected stat error, got %v", err)
	}

	readDirFn = func(string) ([]os.DirEntry, error) {
		return []os.DirEntry{
			stubDirEntry{name: "subdir", isDir: true},
			stubDirEntry{name: "note.txt"},
		}, nil
	}
	if path, _, ok, err := FindLatestRecentBackup(root, "lifebase", now, 6*time.Hour); err != nil || ok || path != "" {
		t.Fatalf("expected filtered entries to produce no backups, got path=%s ok=%v err=%v", path, ok, err)
	}
}

func TestCopyFile(t *testing.T) {
	prevRemove := removeFileFn
	t.Cleanup(func() { removeFileFn = prevRemove })

	t.Run("success", func(t *testing.T) {
		dir := t.TempDir()
		src := filepath.Join(dir, "source.txt")
		dst := filepath.Join(dir, "dest.txt")
		if err := os.WriteFile(src, []byte("backup"), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if err := copyFile(src, dst); err != nil {
			t.Fatalf("copy file: %v", err)
		}
		content, err := os.ReadFile(dst)
		if err != nil {
			t.Fatalf("read dest: %v", err)
		}
		if string(content) != "backup" {
			t.Fatalf("unexpected copied content: %q", string(content))
		}
	})

	t.Run("open_create_and_copy_errors", func(t *testing.T) {
		dir := t.TempDir()
		if err := copyFile(filepath.Join(dir, "missing.txt"), filepath.Join(dir, "dest.txt")); err == nil {
			t.Fatal("expected source open error")
		}
		src := filepath.Join(dir, "source.txt")
		if err := os.WriteFile(src, []byte("backup"), 0o644); err != nil {
			t.Fatalf("write source: %v", err)
		}
		if err := copyFile(src, filepath.Join(dir, "missing", "dest.txt")); err == nil {
			t.Fatal("expected destination create error")
		}

		copyDir := t.TempDir()
		removed := ""
		removeFileFn = func(path string) error {
			removed = path
			return nil
		}
		dst := filepath.Join(copyDir, "dest.txt")
		if err := copyFile(copyDir, dst); err == nil {
			t.Fatal("expected copy error when source is directory")
		}
		if removed != dst {
			t.Fatalf("expected partial destination cleanup, got %q", removed)
		}
	})
}

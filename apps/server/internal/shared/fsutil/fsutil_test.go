package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestPruneEmptyParentsRemovesNestedDirs(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "u1", "ab")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatalf("mkdir leaf: %v", err)
	}

	if err := PruneEmptyParents(root, leaf); err != nil {
		t.Fatalf("prune empty parents: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "u1", "ab")); !os.IsNotExist(err) {
		t.Fatalf("expected leaf dir pruned, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "u1")); !os.IsNotExist(err) {
		t.Fatalf("expected user dir pruned, stat err=%v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("expected root preserved, stat err=%v", err)
	}
}

func TestRemoveEmptyDirsRemovesOnlyEmptyDirs(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "empty", "child"), 0o755); err != nil {
		t.Fatalf("mkdir empty child: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "keep"), 0o755); err != nil {
		t.Fatalf("mkdir keep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	removed, err := RemoveEmptyDirs(root)
	if err != nil {
		t.Fatalf("remove empty dirs: %v", err)
	}
	if removed != 2 {
		t.Fatalf("expected 2 removed dirs, got %d", removed)
	}

	if _, err := os.Stat(filepath.Join(root, "empty")); !os.IsNotExist(err) {
		t.Fatalf("expected empty dir removed, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "keep")); err != nil {
		t.Fatalf("expected non-empty dir kept, stat err=%v", err)
	}
}

func TestPruneEmptyParentsOutsideRootNoop(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := PruneEmptyParents(root, outside); err != nil {
		t.Fatalf("prune outside root should no-op, got %v", err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside dir should remain, stat err=%v", err)
	}
}

func TestPruneEmptyParentsMissingStartNoop(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "missing", "child")
	if err := PruneEmptyParents(root, missing); err != nil {
		t.Fatalf("prune missing start should no-op, got %v", err)
	}
}

func TestPruneEmptyParentsPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on windows")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	leaf := filepath.Join(locked, "leaf")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatalf("mkdir leaf: %v", err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(locked, 0o755)
	})

	if err := PruneEmptyParents(root, leaf); err == nil {
		t.Fatal("expected permission error while pruning")
	}
}

func TestRemoveEmptyDirsRootNotExist(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing-root")
	removed, err := RemoveEmptyDirs(root)
	if err != nil {
		t.Fatalf("remove missing root: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed on missing root, got %d", removed)
	}
}

func TestRemoveEmptyDirsPermissionError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission model differs on windows")
	}

	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	child := filepath.Join(locked, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.Chmod(locked, 0o555); err != nil {
		t.Fatalf("chmod locked: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(locked, 0o755)
	})

	if _, err := RemoveEmptyDirs(root); err == nil {
		t.Fatal("expected permission error while removing empty dirs")
	}
}

func TestPruneEmptyParentsStopsAtNonEmptyDirectory(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "u1", "child")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatalf("mkdir leaf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(leaf, "keep.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}

	if err := PruneEmptyParents(root, leaf); err != nil {
		t.Fatalf("prune non-empty leaf should stop cleanly: %v", err)
	}
	if _, err := os.Stat(leaf); err != nil {
		t.Fatalf("leaf should remain because it is non-empty: %v", err)
	}
}

func TestRemoveEmptyDirsInvalidRootError(t *testing.T) {
	if _, err := RemoveEmptyDirs(string([]byte{0})); err == nil {
		t.Fatal("expected invalid root path error")
	}
}

func TestIsWithinRootInvalidPath(t *testing.T) {
	if isWithinRoot(string([]byte{0}), "target") {
		t.Fatal("expected invalid root path to return false")
	}
}

func TestIsWithinRootTrueCases(t *testing.T) {
	root := t.TempDir()
	if !isWithinRoot(root, root) {
		t.Fatal("expected root to be within root")
	}
	child := filepath.Join(root, "a", "b")
	if !isWithinRoot(root, child) {
		t.Fatal("expected child to be within root")
	}
}

func TestRemoveEmptyDirsRootFileNoop(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "not-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file root: %v", err)
	}

	removed, err := RemoveEmptyDirs(filePath)
	if err != nil {
		t.Fatalf("expected file root to return no-op without error: %v", err)
	}
	if removed != 0 {
		t.Fatalf("expected 0 removed for file root, got %d", removed)
	}
}

func TestPruneEmptyParentsRemoveError(t *testing.T) {
	prevRemove := removeDirFn
	prevDir := dirPathFn
	t.Cleanup(func() {
		removeDirFn = prevRemove
		dirPathFn = prevDir
	})

	removeDirFn = func(string) error { return errors.New("remove fail") }
	dirPathFn = filepath.Dir

	if err := PruneEmptyParents("/root", "/root/child"); err == nil || err.Error() != "remove fail" {
		t.Fatalf("expected remove error, got %v", err)
	}
}

func TestPruneEmptyParentsStopsWhenParentEqualsCurrent(t *testing.T) {
	prevRemove := removeDirFn
	prevDir := dirPathFn
	t.Cleanup(func() {
		removeDirFn = prevRemove
		dirPathFn = prevDir
	})

	removeDirFn = func(string) error { return nil }
	dirPathFn = func(string) string { return "/stuck" }

	if err := PruneEmptyParents("/", "/stuck"); err != nil {
		t.Fatalf("expected parent==current branch to return nil, got %v", err)
	}
}

func TestRemoveEmptyDirsStatError(t *testing.T) {
	prevStat := statPathFn
	t.Cleanup(func() { statPathFn = prevStat })

	statPathFn = func(string) (os.FileInfo, error) { return nil, errors.New("stat fail") }

	if _, err := RemoveEmptyDirs("/root"); err == nil || err.Error() != "stat fail" {
		t.Fatalf("expected stat error, got %v", err)
	}
}

func TestRemoveEmptyDirsWalkError(t *testing.T) {
	prevStat := statPathFn
	prevWalk := walkDirFn
	t.Cleanup(func() {
		statPathFn = prevStat
		walkDirFn = prevWalk
	})

	statPathFn = func(string) (os.FileInfo, error) { return fakeFileInfo{name: "root", dir: true}, nil }
	walkDirFn = func(string, fs.WalkDirFunc) error { return errors.New("walk fail") }

	if _, err := RemoveEmptyDirs("/root"); err == nil || err.Error() != "walk fail" {
		t.Fatalf("expected walk error, got %v", err)
	}
}

func TestRemoveEmptyDirsWalkCallbackErrors(t *testing.T) {
	prevStat := statPathFn
	prevWalk := walkDirFn
	t.Cleanup(func() {
		statPathFn = prevStat
		walkDirFn = prevWalk
	})

	statPathFn = func(string) (os.FileInfo, error) { return fakeFileInfo{name: "root", dir: true}, nil }

	walkDirFn = func(root string, fn fs.WalkDirFunc) error {
		return fn(filepath.Join(root, "missing"), fakeDirEntry{name: "missing", dir: true}, os.ErrNotExist)
	}
	if removed, err := RemoveEmptyDirs("/root"); err != nil || removed != 0 {
		t.Fatalf("expected not-exist callback error to be ignored, removed=%d err=%v", removed, err)
	}

	walkDirFn = func(root string, fn fs.WalkDirFunc) error {
		return fn(filepath.Join(root, "boom"), fakeDirEntry{name: "boom", dir: true}, errors.New("callback fail"))
	}
	if _, err := RemoveEmptyDirs("/root"); err == nil || err.Error() != "callback fail" {
		t.Fatalf("expected callback error to propagate, got %v", err)
	}
}

func TestRemoveEmptyDirsRemoveError(t *testing.T) {
	prevStat := statPathFn
	prevWalk := walkDirFn
	prevRemove := removeDirFn
	t.Cleanup(func() {
		statPathFn = prevStat
		walkDirFn = prevWalk
		removeDirFn = prevRemove
	})

	statPathFn = func(string) (os.FileInfo, error) { return fakeFileInfo{name: "root", dir: true}, nil }
	walkDirFn = func(root string, fn fs.WalkDirFunc) error {
		if err := fn(root, fakeDirEntry{name: filepath.Base(root), dir: true}, nil); err != nil {
			return err
		}
		return fn(filepath.Join(root, "child"), fakeDirEntry{name: "child", dir: true}, nil)
	}
	removeDirFn = func(path string) error {
		if filepath.Base(path) == "child" {
			return errors.New("remove fail")
		}
		return nil
	}

	if removed, err := RemoveEmptyDirs("/root"); err == nil || err.Error() != "remove fail" || removed != 0 {
		t.Fatalf("expected remove error with 0 removed, got removed=%d err=%v", removed, err)
	}
}

func TestIsWithinRootRelError(t *testing.T) {
	prevRel := relPathFn
	t.Cleanup(func() { relPathFn = prevRel })

	relPathFn = func(string, string) (string, error) { return "", errors.New("rel fail") }

	if isWithinRoot("/root", "/root/child") {
		t.Fatal("expected rel error to return false")
	}
}

type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string { return f.name }
func (f fakeFileInfo) Size() int64  { return 0 }
func (f fakeFileInfo) Mode() os.FileMode {
	if f.dir {
		return os.ModeDir | 0o755
	}
	return 0o644
}
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

type fakeDirEntry struct {
	name string
	dir  bool
}

func (d fakeDirEntry) Name() string { return d.name }
func (d fakeDirEntry) IsDir() bool  { return d.dir }
func (d fakeDirEntry) Type() fs.FileMode {
	if d.dir {
		return os.ModeDir
	}
	return 0
}
func (d fakeDirEntry) Info() (os.FileInfo, error) { return fakeFileInfo{name: d.name, dir: d.dir}, nil }

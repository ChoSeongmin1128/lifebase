package fsutil

import (
	"os"
	"path/filepath"
	"testing"
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

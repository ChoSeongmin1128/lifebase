package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocalStorageSaveReadDelete(t *testing.T) {
	base := t.TempDir()
	s := NewLocalStorage(base)

	data := []byte("hello")
	storagePath, err := s.Save("u1", "ab1234", data)
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if storagePath == "" {
		t.Fatal("expected storage path")
	}

	read, err := s.Read(storagePath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(read) != "hello" {
		t.Fatalf("unexpected read content: %q", string(read))
	}

	if err := s.Delete(storagePath); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, storagePath)); !os.IsNotExist(err) {
		t.Fatalf("expected deleted file, stat err=%v", err)
	}
}

func TestLocalStorageDeletePrunesEmptyParentDirs(t *testing.T) {
	base := t.TempDir()
	s := NewLocalStorage(base)

	storagePath, err := s.Save("u1", "ab1234", []byte("hello"))
	if err != nil {
		t.Fatalf("save failed: %v", err)
	}

	if err := s.Delete(storagePath); err != nil {
		t.Fatalf("delete failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(base, "u1", "ab")); !os.IsNotExist(err) {
		t.Fatalf("expected empty prefix dir pruned, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "u1")); !os.IsNotExist(err) {
		t.Fatalf("expected empty user dir pruned, stat err=%v", err)
	}
}

func TestLocalStorageSaveCreateDirError(t *testing.T) {
	base := t.TempDir()
	blocker := filepath.Join(base, "u2")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	s := NewLocalStorage(base)
	if _, err := s.Save("u2", "ab1234", []byte("x")); err == nil {
		t.Fatal("expected mkdir error")
	}
}

func TestLocalStorageSaveWriteErrorAndDeleteMissing(t *testing.T) {
	base := t.TempDir()
	userID := "u3"
	fileID := "ab1234"
	prefix := fileID[:2]
	dir := filepath.Join(base, userID, prefix)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	fullPath := filepath.Join(dir, fileID)
	if err := os.MkdirAll(fullPath, 0o755); err != nil {
		t.Fatalf("mkdir full path as dir: %v", err)
	}

	s := NewLocalStorage(base)
	if _, err := s.Save(userID, fileID, []byte("x")); err == nil {
		t.Fatal("expected write error")
	}
	if err := s.Delete(filepath.Join(userID, prefix, "missing")); err == nil {
		t.Fatal("expected delete missing file error")
	}
}

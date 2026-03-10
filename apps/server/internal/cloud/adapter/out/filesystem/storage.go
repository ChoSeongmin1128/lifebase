package filesystem

import (
	"fmt"
	"os"
	"path/filepath"

	"lifebase/internal/shared/fsutil"
)

type localStorage struct {
	basePath string
}

func NewLocalStorage(basePath string) *localStorage {
	return &localStorage{basePath: basePath}
}

func (s *localStorage) Save(userID, fileID string, data []byte) (string, error) {
	prefix := fileID[:2]
	dir := filepath.Join(s.basePath, userID, prefix)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	storagePath := filepath.Join(userID, prefix, fileID)
	fullPath := filepath.Join(s.basePath, storagePath)

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		_ = fsutil.PruneEmptyParents(s.basePath, dir)
		return "", fmt.Errorf("write file: %w", err)
	}

	return storagePath, nil
}

func (s *localStorage) Read(storagePath string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, storagePath)
	return os.ReadFile(fullPath)
}

func (s *localStorage) Delete(storagePath string) error {
	fullPath := filepath.Join(s.basePath, storagePath)
	if err := os.Remove(fullPath); err != nil {
		return err
	}
	return fsutil.PruneEmptyParents(s.basePath, filepath.Dir(fullPath))
}

type localThumbnailStorage struct {
	basePath string
}

func NewLocalThumbnailStorage(basePath string) *localThumbnailStorage {
	return &localThumbnailStorage{basePath: basePath}
}

func (s *localThumbnailStorage) Delete(userID, fileID string) error {
	userDir := filepath.Join(s.basePath, userID)
	names := []string{
		fileID + "_small.webp",
		fileID + "_medium.webp",
	}
	for _, name := range names {
		fullPath := filepath.Join(userDir, name)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return fsutil.PruneEmptyParents(s.basePath, userDir)
}

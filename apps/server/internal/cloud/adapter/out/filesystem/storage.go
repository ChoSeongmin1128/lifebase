package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
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
	return os.Remove(fullPath)
}

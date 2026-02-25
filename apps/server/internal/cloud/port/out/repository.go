package out

import (
	"context"

	"lifebase/internal/cloud/domain"
)

type FolderRepository interface {
	Create(ctx context.Context, folder *domain.Folder) error
	FindByID(ctx context.Context, userID, id string) (*domain.Folder, error)
	ListByParent(ctx context.Context, userID string, parentID *string) ([]*domain.Folder, error)
	Update(ctx context.Context, folder *domain.Folder) error
	SoftDelete(ctx context.Context, userID, id string) error
	Restore(ctx context.Context, userID, id string) error
	HardDelete(ctx context.Context, id string) error
	ListTrashed(ctx context.Context, userID string) ([]*domain.Folder, error)
}

type FileRepository interface {
	Create(ctx context.Context, file *domain.File) error
	FindByID(ctx context.Context, userID, id string) (*domain.File, error)
	ListByFolder(ctx context.Context, userID string, folderID *string, sortBy, sortDir string) ([]*domain.File, error)
	Update(ctx context.Context, file *domain.File) error
	SoftDelete(ctx context.Context, userID, id string) error
	Restore(ctx context.Context, userID, id string) error
	HardDelete(ctx context.Context, id string) error
	ListTrashed(ctx context.Context, userID string) ([]*domain.File, error)
	UpdateStorageUsed(ctx context.Context, userID string, delta int64) error
	Search(ctx context.Context, userID, query string, limit int) ([]*domain.File, error)
}

type FileStorage interface {
	Save(userID, fileID string, data []byte) (storagePath string, err error)
	Read(storagePath string) ([]byte, error)
	Delete(storagePath string) error
}

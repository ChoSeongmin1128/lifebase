package in

import (
	"context"

	"lifebase/internal/cloud/domain"
)

type FolderItem struct {
	Type   string         `json:"type"`
	Folder *domain.Folder `json:"folder,omitempty"`
	File   *domain.File   `json:"file,omitempty"`
	Path   string         `json:"path,omitempty"`
}

type StarItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type CloudUseCase interface {
	// Folders
	CreateFolder(ctx context.Context, userID string, parentID *string, name string) (*domain.Folder, error)
	GetFolder(ctx context.Context, userID, folderID string) (*domain.Folder, error)
	ListFolder(ctx context.Context, userID string, folderID *string, sortBy, sortDir string) ([]FolderItem, error)
	RenameFolder(ctx context.Context, userID, folderID, newName string) error
	MoveFolder(ctx context.Context, userID, folderID string, newParentID *string) error
	CopyFolder(ctx context.Context, userID, folderID string, targetParentID *string) error
	DeleteFolder(ctx context.Context, userID, folderID string) error
	GetTrashFolder(ctx context.Context, userID, folderID string) (*domain.Folder, error)

	// Files
	UploadFile(ctx context.Context, userID string, folderID *string, name string, mimeType string, size int64, data []byte) (*domain.File, error)
	GetFile(ctx context.Context, userID, fileID string) (*domain.File, error)
	DownloadFile(ctx context.Context, userID, fileID string) ([]byte, *domain.File, error)
	GetFileContent(ctx context.Context, userID, fileID string) (string, *domain.File, error)
	RenameFile(ctx context.Context, userID, fileID, newName string) error
	UpdateFileContent(ctx context.Context, userID, fileID, content string) error
	MoveFile(ctx context.Context, userID, fileID string, newFolderID *string) error
	CopyFile(ctx context.Context, userID, fileID string, targetFolderID *string) (*domain.File, error)
	DiscardFile(ctx context.Context, userID, fileID string) error
	DeleteFile(ctx context.Context, userID, fileID string) error

	// Trash
	ListTrash(ctx context.Context, userID string, folderID *string) ([]FolderItem, error)
	RestoreItem(ctx context.Context, userID, itemID, itemType string) error
	EmptyTrash(ctx context.Context, userID string) error

	// Views
	ListRecent(ctx context.Context, userID string) ([]FolderItem, error)
	ListShared(ctx context.Context, userID string) ([]FolderItem, error)
	ListStarred(ctx context.Context, userID string) ([]FolderItem, error)

	// Stars
	ListStars(ctx context.Context, userID string) ([]StarItem, error)
	StarItem(ctx context.Context, userID, itemID, itemType string) error
	UnstarItem(ctx context.Context, userID, itemID, itemType string) error

	// Search
	Search(ctx context.Context, userID, query string) ([]*domain.File, error)
}

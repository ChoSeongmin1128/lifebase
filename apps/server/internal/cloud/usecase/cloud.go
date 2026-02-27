package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"lifebase/internal/cloud/domain"
	portin "lifebase/internal/cloud/port/in"
	portout "lifebase/internal/cloud/port/out"
)

type cloudUseCase struct {
	folders portout.FolderRepository
	files   portout.FileRepository
	shared  portout.SharedRepository
	stars   portout.StarRepository
	storage portout.FileStorage
	queue   portout.ThumbnailQueue
}

func NewCloudUseCase(
	folders portout.FolderRepository,
	files portout.FileRepository,
	shared portout.SharedRepository,
	stars portout.StarRepository,
	storage portout.FileStorage,
	queue portout.ThumbnailQueue,
) portin.CloudUseCase {
	return &cloudUseCase{
		folders: folders,
		files:   files,
		shared:  shared,
		stars:   stars,
		storage: storage,
		queue:   queue,
	}
}

// Folders

func (uc *cloudUseCase) CreateFolder(ctx context.Context, userID string, parentID *string, name string) (*domain.Folder, error) {
	if parentID != nil {
		parent, err := uc.folders.FindByID(ctx, userID, *parentID)
		if err != nil || parent == nil {
			return nil, fmt.Errorf("parent folder not found")
		}
	}

	now := time.Now()
	folder := &domain.Folder{
		ID:        uuid.New().String(),
		UserID:    userID,
		ParentID:  parentID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.folders.Create(ctx, folder); err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	return folder, nil
}

func (uc *cloudUseCase) GetFolder(ctx context.Context, userID, folderID string) (*domain.Folder, error) {
	return uc.folders.FindByID(ctx, userID, folderID)
}

func (uc *cloudUseCase) ListFolder(ctx context.Context, userID string, folderID *string, sortBy, sortDir string) ([]portin.FolderItem, error) {
	folders, err := uc.folders.ListByParent(ctx, userID, folderID)
	if err != nil {
		return nil, err
	}

	files, err := uc.files.ListByFolder(ctx, userID, folderID, sortBy, sortDir)
	if err != nil {
		return nil, err
	}

	items := make([]portin.FolderItem, 0, len(folders)+len(files))
	for _, f := range folders {
		items = append(items, portin.FolderItem{Type: "folder", Folder: f})
	}
	for _, f := range files {
		items = append(items, portin.FolderItem{Type: "file", File: f})
	}
	return items, nil
}

func (uc *cloudUseCase) RenameFolder(ctx context.Context, userID, folderID, newName string) error {
	folder, err := uc.folders.FindByID(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("folder not found")
	}
	folder.Name = newName
	folder.UpdatedAt = time.Now()
	return uc.folders.Update(ctx, folder)
}

func (uc *cloudUseCase) MoveFolder(ctx context.Context, userID, folderID string, newParentID *string) error {
	folder, err := uc.folders.FindByID(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("folder not found")
	}

	if newParentID != nil {
		if *newParentID == folderID {
			return fmt.Errorf("cannot move folder into itself")
		}
		parent, err := uc.folders.FindByID(ctx, userID, *newParentID)
		if err != nil || parent == nil {
			return fmt.Errorf("target folder not found")
		}
	}

	folder.ParentID = newParentID
	folder.UpdatedAt = time.Now()
	return uc.folders.Update(ctx, folder)
}

func (uc *cloudUseCase) DeleteFolder(ctx context.Context, userID, folderID string) error {
	return uc.folders.SoftDelete(ctx, userID, folderID)
}

// Files

// resolveFileName returns a unique name within the folder using Google Drive style:
// "file.txt" → "file (1).txt" → "file (2).txt" → ...
func (uc *cloudUseCase) resolveFileName(ctx context.Context, userID string, folderID *string, name string) (string, error) {
	exists, err := uc.files.ExistsByName(ctx, userID, folderID, name)
	if err != nil {
		return "", err
	}
	if !exists {
		return name, nil
	}

	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)

	for i := 1; i <= 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)%s", stem, i, ext)
		exists, err := uc.files.ExistsByName(ctx, userID, folderID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve unique filename for %q", name)
}

func (uc *cloudUseCase) UploadFile(ctx context.Context, userID string, folderID *string, name string, mimeType string, size int64, data []byte) (*domain.File, error) {
	if folderID != nil {
		folder, err := uc.folders.FindByID(ctx, userID, *folderID)
		if err != nil || folder == nil {
			return nil, fmt.Errorf("folder not found")
		}
	}

	resolvedName, err := uc.resolveFileName(ctx, userID, folderID, name)
	if err != nil {
		return nil, fmt.Errorf("resolve filename: %w", err)
	}
	name = resolvedName

	fileID := uuid.New().String()
	storagePath, err := uc.storage.Save(userID, fileID, data)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	now := time.Now()
	file := &domain.File{
		ID:          fileID,
		UserID:      userID,
		FolderID:    folderID,
		Name:        name,
		MimeType:    mimeType,
		SizeBytes:   size,
		StoragePath: storagePath,
		ThumbStatus: "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := uc.files.Create(ctx, file); err != nil {
		_ = uc.storage.Delete(storagePath)
		return nil, fmt.Errorf("create file record: %w", err)
	}

	if err := uc.files.UpdateStorageUsed(ctx, userID, size); err != nil {
		return nil, fmt.Errorf("update storage used: %w", err)
	}

	// Enqueue thumbnail generation
	if uc.queue != nil {
		_ = uc.queue.EnqueueThumbnail(ctx, portout.ThumbnailTask{
			FileID:      fileID,
			UserID:      userID,
			StoragePath: storagePath,
			MimeType:    mimeType,
		})
	}

	return file, nil
}

func (uc *cloudUseCase) GetFile(ctx context.Context, userID, fileID string) (*domain.File, error) {
	return uc.files.FindByID(ctx, userID, fileID)
}

func (uc *cloudUseCase) DownloadFile(ctx context.Context, userID, fileID string) ([]byte, *domain.File, error) {
	file, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil {
		return nil, nil, fmt.Errorf("file not found")
	}

	data, err := uc.storage.Read(file.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}

	return data, file, nil
}

func (uc *cloudUseCase) RenameFile(ctx context.Context, userID, fileID, newName string) error {
	file, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil {
		return fmt.Errorf("file not found")
	}
	file.Name = newName
	file.UpdatedAt = time.Now()
	return uc.files.Update(ctx, file)
}

func (uc *cloudUseCase) MoveFile(ctx context.Context, userID, fileID string, newFolderID *string) error {
	file, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil {
		return fmt.Errorf("file not found")
	}

	if newFolderID != nil {
		folder, err := uc.folders.FindByID(ctx, userID, *newFolderID)
		if err != nil || folder == nil {
			return fmt.Errorf("target folder not found")
		}
	}

	file.FolderID = newFolderID
	file.UpdatedAt = time.Now()
	return uc.files.Update(ctx, file)
}

func (uc *cloudUseCase) DeleteFile(ctx context.Context, userID, fileID string) error {
	return uc.files.SoftDelete(ctx, userID, fileID)
}

// Trash

func (uc *cloudUseCase) ListTrash(ctx context.Context, userID string) ([]portin.FolderItem, error) {
	folders, err := uc.folders.ListTrashed(ctx, userID)
	if err != nil {
		return nil, err
	}

	files, err := uc.files.ListTrashed(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]portin.FolderItem, 0, len(folders)+len(files))
	for _, f := range folders {
		items = append(items, portin.FolderItem{Type: "folder", Folder: f})
	}
	for _, f := range files {
		items = append(items, portin.FolderItem{Type: "file", File: f})
	}
	return items, nil
}

func (uc *cloudUseCase) RestoreItem(ctx context.Context, userID, itemID, itemType string) error {
	switch itemType {
	case "folder":
		return uc.folders.Restore(ctx, userID, itemID)
	case "file":
		file, err := uc.files.FindTrashedByID(ctx, userID, itemID)
		if err != nil {
			return fmt.Errorf("file not found in trash")
		}
		resolvedName, err := uc.resolveFileName(ctx, userID, file.FolderID, file.Name)
		if err != nil {
			return fmt.Errorf("resolve filename: %w", err)
		}
		if resolvedName != file.Name {
			file.Name = resolvedName
			file.UpdatedAt = time.Now()
			if err := uc.files.Update(ctx, file); err != nil {
				return fmt.Errorf("rename on restore: %w", err)
			}
		}
		return uc.files.Restore(ctx, userID, itemID)
	default:
		return fmt.Errorf("invalid item type: %s", itemType)
	}
}

func (uc *cloudUseCase) EmptyTrash(ctx context.Context, userID string) error {
	files, err := uc.files.ListTrashed(ctx, userID)
	if err != nil {
		return err
	}

	for _, f := range files {
		_ = uc.storage.Delete(f.StoragePath)
		_ = uc.files.HardDelete(ctx, f.ID)
		_ = uc.files.UpdateStorageUsed(ctx, userID, -f.SizeBytes)
	}

	folders, err := uc.folders.ListTrashed(ctx, userID)
	if err != nil {
		return err
	}
	for _, f := range folders {
		_ = uc.folders.HardDelete(ctx, f.ID)
	}

	return nil
}

// Views

func (uc *cloudUseCase) ListRecent(ctx context.Context, userID string) ([]portin.FolderItem, error) {
	files, err := uc.files.ListRecent(ctx, userID, 100)
	if err != nil {
		return nil, err
	}
	return toFileItems(files), nil
}

func (uc *cloudUseCase) ListShared(ctx context.Context, userID string) ([]portin.FolderItem, error) {
	folders, err := uc.shared.ListSharedFolders(ctx, userID)
	if err != nil {
		return nil, err
	}
	return toFolderItems(folders), nil
}

func (uc *cloudUseCase) ListStarred(ctx context.Context, userID string) ([]portin.FolderItem, error) {
	refs, err := uc.stars.List(ctx, userID)
	if err != nil {
		return nil, err
	}

	items := make([]portin.FolderItem, 0, len(refs))
	for _, ref := range refs {
		switch ref.ItemType {
		case "folder":
			folder, err := uc.folders.FindByID(ctx, userID, ref.ItemID)
			if err == nil && folder != nil {
				items = append(items, portin.FolderItem{Type: "folder", Folder: folder})
			}
		case "file":
			file, err := uc.files.FindByID(ctx, userID, ref.ItemID)
			if err == nil && file != nil {
				items = append(items, portin.FolderItem{Type: "file", File: file})
			}
		}
	}
	return items, nil
}

// Stars

func (uc *cloudUseCase) ListStars(ctx context.Context, userID string) ([]portin.StarItem, error) {
	refs, err := uc.stars.List(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]portin.StarItem, 0, len(refs))
	for _, ref := range refs {
		out = append(out, portin.StarItem{
			ID:   ref.ItemID,
			Type: ref.ItemType,
		})
	}
	return out, nil
}

func (uc *cloudUseCase) StarItem(ctx context.Context, userID, itemID, itemType string) error {
	switch itemType {
	case "folder":
		folder, err := uc.folders.FindByID(ctx, userID, itemID)
		if err != nil || folder == nil {
			return fmt.Errorf("folder not found")
		}
	case "file":
		file, err := uc.files.FindByID(ctx, userID, itemID)
		if err != nil || file == nil {
			return fmt.Errorf("file not found")
		}
	default:
		return fmt.Errorf("invalid item type: %s", itemType)
	}
	return uc.stars.Set(ctx, userID, itemID, itemType)
}

func (uc *cloudUseCase) UnstarItem(ctx context.Context, userID, itemID, itemType string) error {
	if itemType != "folder" && itemType != "file" {
		return fmt.Errorf("invalid item type: %s", itemType)
	}
	return uc.stars.Unset(ctx, userID, itemID, itemType)
}

// Search

func (uc *cloudUseCase) Search(ctx context.Context, userID, query string) ([]*domain.File, error) {
	return uc.files.Search(ctx, userID, query, 50)
}

func toFolderItems(folders []*domain.Folder) []portin.FolderItem {
	items := make([]portin.FolderItem, 0, len(folders))
	for _, f := range folders {
		items = append(items, portin.FolderItem{Type: "folder", Folder: f})
	}
	return items
}

func toFileItems(files []*domain.File) []portin.FolderItem {
	items := make([]portin.FolderItem, 0, len(files))
	for _, f := range files {
		items = append(items, portin.FolderItem{Type: "file", File: f})
	}
	return items
}

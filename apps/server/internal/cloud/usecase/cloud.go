package usecase

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"lifebase/internal/cloud/domain"
	portin "lifebase/internal/cloud/port/in"
	portout "lifebase/internal/cloud/port/out"
)

const maxEditableFileBytes = 2 * 1024 * 1024

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
		descendant, err := uc.isDescendantFolder(ctx, userID, folderID, newParentID)
		if err != nil {
			return fmt.Errorf("validate target folder: %w", err)
		}
		if descendant {
			return fmt.Errorf("cannot move folder into its descendant")
		}
	}

	if (folder.ParentID == nil && newParentID == nil) || (folder.ParentID != nil && newParentID != nil && *folder.ParentID == *newParentID) {
		return nil
	}

	folder.ParentID = newParentID
	folder.UpdatedAt = time.Now()
	return uc.folders.Update(ctx, folder)
}

func (uc *cloudUseCase) CopyFolder(ctx context.Context, userID, folderID string, targetParentID *string) error {
	_ = ctx
	_ = userID
	_ = folderID
	_ = targetParentID
	return fmt.Errorf("folder copy is not supported")
}

func (uc *cloudUseCase) DeleteFolder(ctx context.Context, userID, folderID string) error {
	root, err := uc.folders.FindByID(ctx, userID, folderID)
	if err != nil || root == nil {
		return fmt.Errorf("folder not found")
	}

	folders, files, err := uc.collectActiveFolderTree(ctx, userID, root)
	if err != nil {
		return err
	}

	for _, file := range files {
		if err := uc.files.SoftDelete(ctx, userID, file.ID); err != nil {
			return fmt.Errorf("delete file: %w", err)
		}
	}
	for i := len(folders) - 1; i >= 0; i-- {
		if err := uc.folders.SoftDelete(ctx, userID, folders[i].ID); err != nil {
			return fmt.Errorf("delete folder: %w", err)
		}
	}
	return nil
}

func (uc *cloudUseCase) GetTrashFolder(ctx context.Context, userID, folderID string) (*domain.Folder, error) {
	folder, err := uc.findFolderInTrashScope(ctx, userID, folderID, nil)
	if err != nil {
		return nil, fmt.Errorf("folder not found in trash")
	}
	return folder, nil
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

func (uc *cloudUseCase) resolveFolderName(ctx context.Context, userID string, parentID *string, name string) (string, error) {
	exists, err := uc.folders.ExistsByName(ctx, userID, parentID, name)
	if err != nil {
		return "", err
	}
	if !exists {
		return name, nil
	}

	for i := 1; i <= 10000; i++ {
		candidate := fmt.Sprintf("%s (%d)", name, i)
		exists, err := uc.folders.ExistsByName(ctx, userID, parentID, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve unique folder name for %q", name)
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

func isEditableTextFile(name, mimeType string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == ".md" || ext == ".txt" {
		return true
	}
	if strings.HasPrefix(strings.ToLower(mimeType), "text/") {
		return true
	}
	return false
}

func (uc *cloudUseCase) GetFileContent(ctx context.Context, userID, fileID string) (string, *domain.File, error) {
	file, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil {
		return "", nil, fmt.Errorf("file not found")
	}
	if !isEditableTextFile(file.Name, file.MimeType) {
		return "", nil, fmt.Errorf("file is not editable")
	}
	if file.SizeBytes > maxEditableFileBytes {
		return "", nil, fmt.Errorf("file is too large to edit")
	}

	data, err := uc.storage.Read(file.StoragePath)
	if err != nil {
		return "", nil, fmt.Errorf("read file: %w", err)
	}
	if !utf8.Valid(data) {
		return "", nil, fmt.Errorf("file is not valid utf-8")
	}

	return string(data), file, nil
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

func (uc *cloudUseCase) UpdateFileContent(ctx context.Context, userID, fileID, content string) error {
	file, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil {
		return fmt.Errorf("file not found")
	}
	if !isEditableTextFile(file.Name, file.MimeType) {
		return fmt.Errorf("file is not editable")
	}

	data := []byte(content)
	if int64(len(data)) > maxEditableFileBytes {
		return fmt.Errorf("file is too large to edit")
	}

	storagePath, err := uc.storage.Save(userID, file.ID, data)
	if err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	delta := int64(len(data)) - file.SizeBytes
	file.StoragePath = storagePath
	file.SizeBytes = int64(len(data))
	file.UpdatedAt = time.Now()

	if err := uc.files.Update(ctx, file); err != nil {
		return fmt.Errorf("update file metadata: %w", err)
	}
	if delta != 0 {
		if err := uc.files.UpdateStorageUsed(ctx, userID, delta); err != nil {
			return fmt.Errorf("update storage used: %w", err)
		}
	}

	return nil
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

func (uc *cloudUseCase) CopyFile(ctx context.Context, userID, fileID string, targetFolderID *string) error {
	source, err := uc.files.FindByID(ctx, userID, fileID)
	if err != nil || source == nil {
		return fmt.Errorf("file not found")
	}
	return uc.copyFileToFolder(ctx, userID, source, targetFolderID)
}

func (uc *cloudUseCase) DeleteFile(ctx context.Context, userID, fileID string) error {
	return uc.files.SoftDelete(ctx, userID, fileID)
}

// Trash

func (uc *cloudUseCase) ListTrash(ctx context.Context, userID string, folderID *string) ([]portin.FolderItem, error) {
	folders, err := uc.folders.ListTrashed(ctx, userID)
	if err != nil {
		return nil, err
	}

	files, err := uc.files.ListTrashed(ctx, userID)
	if err != nil {
		return nil, err
	}

	trashedFolderIDs := make(map[string]struct{}, len(folders))
	for _, folder := range folders {
		trashedFolderIDs[folder.ID] = struct{}{}
	}

	if folderID != nil {
		if _, err := uc.findFolderInTrashScope(ctx, userID, *folderID, trashedFolderIDs); err != nil {
			return nil, fmt.Errorf("folder not found in trash")
		}
	}

	items := make([]portin.FolderItem, 0, len(folders)+len(files))
	folderSeen := make(map[string]struct{}, len(folders))
	fileSeen := make(map[string]struct{}, len(files))
	for _, f := range folders {
		if folderID != nil {
			if f.ParentID == nil || *f.ParentID != *folderID {
				continue
			}
		} else if f.ParentID != nil {
			if _, ok := trashedFolderIDs[*f.ParentID]; ok {
				continue
			}
		}
		items = append(items, portin.FolderItem{Type: "folder", Folder: f})
		folderSeen[f.ID] = struct{}{}
	}
	for _, f := range files {
		if folderID != nil {
			if f.FolderID == nil || *f.FolderID != *folderID {
				continue
			}
		} else if f.FolderID != nil {
			if _, ok := trashedFolderIDs[*f.FolderID]; ok {
				continue
			}
		}
		items = append(items, portin.FolderItem{Type: "file", File: f})
		fileSeen[f.ID] = struct{}{}
	}

	if folderID != nil {
		activeFolders, err := uc.folders.ListByParent(ctx, userID, folderID)
		if err != nil {
			return nil, err
		}
		for _, folder := range activeFolders {
			if _, ok := folderSeen[folder.ID]; ok {
				continue
			}
			items = append(items, portin.FolderItem{Type: "folder", Folder: folder})
		}

		activeFiles, err := uc.files.ListByFolder(ctx, userID, folderID, "name", "asc")
		if err != nil {
			return nil, err
		}
		for _, file := range activeFiles {
			if _, ok := fileSeen[file.ID]; ok {
				continue
			}
			items = append(items, portin.FolderItem{Type: "file", File: file})
		}
	}
	return items, nil
}

func (uc *cloudUseCase) RestoreItem(ctx context.Context, userID, itemID, itemType string) error {
	switch itemType {
	case "folder":
		folder, err := uc.folders.FindTrashedByID(ctx, userID, itemID)
		if err != nil || folder == nil {
			activeFolder, activeErr := uc.findFolderInTrashScope(ctx, userID, itemID, nil)
			if activeErr != nil {
				return fmt.Errorf("folder not found in trash")
			}
			return uc.restoreDeletedFolderPath(ctx, userID, activeFolder.ParentID)
		}
		if err := uc.restoreDeletedFolderPath(ctx, userID, folder.ParentID); err != nil {
			return err
		}
		trashedFolders, err := uc.folders.ListTrashed(ctx, userID)
		if err != nil {
			return err
		}
		trashedFiles, err := uc.files.ListTrashed(ctx, userID)
		if err != nil {
			return err
		}
		return uc.restoreFolderSubtree(ctx, userID, itemID, trashedFolders, trashedFiles)
	case "file":
		file, err := uc.files.FindTrashedByID(ctx, userID, itemID)
		if err != nil {
			activeFile, activeErr := uc.files.FindByID(ctx, userID, itemID)
			if activeErr != nil || activeFile == nil {
				return fmt.Errorf("file not found in trash")
			}
			inTrash, trashErr := uc.hasTrashedAncestor(ctx, userID, activeFile.FolderID, nil)
			if trashErr != nil || !inTrash {
				return fmt.Errorf("file not found in trash")
			}
			return uc.restoreDeletedFolderPath(ctx, userID, activeFile.FolderID)
		}
		if err := uc.restoreDeletedFolderPath(ctx, userID, file.FolderID); err != nil {
			return err
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

	folders, err := uc.folders.ListTrashed(ctx, userID)
	if err != nil {
		return err
	}

	fileMap := make(map[string]*domain.File, len(files))
	for _, file := range files {
		fileMap[file.ID] = file
	}
	folderMap := make(map[string]*domain.Folder, len(folders))
	for _, folder := range folders {
		folderMap[folder.ID] = folder
	}

	for _, folder := range folders {
		activeFolders, activeFiles, collectErr := uc.collectActiveFolderTree(ctx, userID, folder)
		if collectErr != nil {
			return collectErr
		}
		for _, activeFolder := range activeFolders {
			folderMap[activeFolder.ID] = activeFolder
		}
		for _, activeFile := range activeFiles {
			fileMap[activeFile.ID] = activeFile
		}
	}

	for _, file := range fileMap {
		_ = uc.storage.Delete(file.StoragePath)
		_ = uc.files.HardDelete(ctx, file.ID)
		_ = uc.files.UpdateStorageUsed(ctx, userID, -file.SizeBytes)
	}

	folderList := make([]*domain.Folder, 0, len(folderMap))
	for _, folder := range folderMap {
		folderList = append(folderList, folder)
	}
	sort.SliceStable(folderList, func(i, j int) bool {
		return folderDepth(folderList[i], folderMap) > folderDepth(folderList[j], folderMap)
	})
	for _, folder := range folderList {
		_ = uc.folders.HardDelete(ctx, folder.ID)
	}

	return nil
}

// Views

func (uc *cloudUseCase) ListRecent(ctx context.Context, userID string) ([]portin.FolderItem, error) {
	files, err := uc.files.ListRecent(ctx, userID, 100)
	if err != nil {
		return nil, err
	}

	cache := map[string]string{}
	items := make([]portin.FolderItem, 0, len(files))
	for _, f := range files {
		folderPath, _ := uc.buildFolderPath(ctx, userID, f.FolderID, cache)
		path := folderPath + "/" + f.Name
		if folderPath == "/" {
			path = "/" + f.Name
		}
		items = append(items, portin.FolderItem{
			Type: "file",
			File: f,
			Path: path,
		})
	}
	return items, nil
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

func (uc *cloudUseCase) collectActiveFolderTree(
	ctx context.Context,
	userID string,
	root *domain.Folder,
) ([]*domain.Folder, []*domain.File, error) {
	folders := []*domain.Folder{root}
	files, err := uc.files.ListByFolder(ctx, userID, &root.ID, "name", "asc")
	if err != nil {
		return nil, nil, err
	}
	children, err := uc.folders.ListByParent(ctx, userID, &root.ID)
	if err != nil {
		return nil, nil, err
	}
	collectedFiles := append([]*domain.File{}, files...)
	for _, child := range children {
		subFolders, subFiles, err := uc.collectActiveFolderTree(ctx, userID, child)
		if err != nil {
			return nil, nil, err
		}
		folders = append(folders, subFolders...)
		collectedFiles = append(collectedFiles, subFiles...)
	}
	return folders, collectedFiles, nil
}

func (uc *cloudUseCase) findFolderInTrashScope(
	ctx context.Context,
	userID string,
	folderID string,
	trashedFolderIDs map[string]struct{},
) (*domain.Folder, error) {
	folder, err := uc.folders.FindTrashedByID(ctx, userID, folderID)
	if err == nil && folder != nil {
		return folder, nil
	}

	folder, err = uc.folders.FindByID(ctx, userID, folderID)
	if err != nil || folder == nil {
		return nil, fmt.Errorf("folder not found in trash")
	}

	inTrash, err := uc.hasTrashedAncestor(ctx, userID, folder.ParentID, trashedFolderIDs)
	if err != nil || !inTrash {
		return nil, fmt.Errorf("folder not found in trash")
	}
	return folder, nil
}

func (uc *cloudUseCase) hasTrashedAncestor(
	ctx context.Context,
	userID string,
	folderID *string,
	trashedFolderIDs map[string]struct{},
) (bool, error) {
	current := folderID
	visited := map[string]bool{}
	for current != nil && *current != "" {
		if visited[*current] {
			return false, fmt.Errorf("cycle detected in folder tree")
		}
		visited[*current] = true

		if trashedFolderIDs != nil {
			if _, ok := trashedFolderIDs[*current]; ok {
				return true, nil
			}
		}

		folder, err := uc.folders.FindTrashedByID(ctx, userID, *current)
		if err == nil && folder != nil {
			return true, nil
		}

		folder, err = uc.folders.FindByID(ctx, userID, *current)
		if err != nil || folder == nil {
			return false, fmt.Errorf("folder not found")
		}
		current = folder.ParentID
	}
	return false, nil
}

func (uc *cloudUseCase) restoreDeletedFolderPath(ctx context.Context, userID string, folderID *string) error {
	if folderID == nil || *folderID == "" {
		return nil
	}
	if folder, err := uc.folders.FindByID(ctx, userID, *folderID); err == nil && folder != nil {
		return nil
	}
	folder, err := uc.folders.FindTrashedByID(ctx, userID, *folderID)
	if err != nil || folder == nil {
		return fmt.Errorf("folder not found in trash")
	}
	if err := uc.restoreDeletedFolderPath(ctx, userID, folder.ParentID); err != nil {
		return err
	}
	return uc.restoreFolderSelf(ctx, userID, folder)
}

func (uc *cloudUseCase) restoreFolderSelf(ctx context.Context, userID string, folder *domain.Folder) error {
	resolvedName, err := uc.resolveFolderName(ctx, userID, folder.ParentID, folder.Name)
	if err != nil {
		return fmt.Errorf("resolve folder name: %w", err)
	}
	if resolvedName != folder.Name {
		folder.Name = resolvedName
		folder.UpdatedAt = time.Now()
		if err := uc.folders.Update(ctx, folder); err != nil {
			return fmt.Errorf("rename folder on restore: %w", err)
		}
	}
	if err := uc.folders.Restore(ctx, userID, folder.ID); err != nil {
		return fmt.Errorf("restore folder: %w", err)
	}
	return nil
}

func (uc *cloudUseCase) restoreFolderSubtree(
	ctx context.Context,
	userID string,
	rootID string,
	trashedFolders []*domain.Folder,
	trashedFiles []*domain.File,
) error {
	folderByID := make(map[string]*domain.Folder, len(trashedFolders))
	foldersByParent := make(map[string][]*domain.Folder)
	for _, folder := range trashedFolders {
		folderByID[folder.ID] = folder
		if folder.ParentID != nil {
			foldersByParent[*folder.ParentID] = append(foldersByParent[*folder.ParentID], folder)
		}
	}
	filesByFolder := make(map[string][]*domain.File)
	for _, file := range trashedFiles {
		if file.FolderID != nil {
			filesByFolder[*file.FolderID] = append(filesByFolder[*file.FolderID], file)
		}
	}

	var restore func(folderID string) error
	restore = func(folderID string) error {
		folder := folderByID[folderID]
		if folder == nil {
			return fmt.Errorf("folder not found in trash")
		}
		if err := uc.restoreFolderSelf(ctx, userID, folder); err != nil {
			return err
		}
		for _, child := range foldersByParent[folderID] {
			if err := restore(child.ID); err != nil {
				return err
			}
		}
		for _, file := range filesByFolder[folderID] {
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
			if err := uc.files.Restore(ctx, userID, file.ID); err != nil {
				return fmt.Errorf("restore file: %w", err)
			}
		}
		return nil
	}

	return restore(rootID)
}

func (uc *cloudUseCase) buildFolderPath(
	ctx context.Context,
	userID string,
	folderID *string,
	cache map[string]string,
) (string, error) {
	if folderID == nil || *folderID == "" {
		return "/", nil
	}

	if cached, ok := cache[*folderID]; ok {
		return cached, nil
	}

	folder, err := uc.folders.FindByID(ctx, userID, *folderID)
	if err != nil || folder == nil {
		return "/", err
	}

	parentPath, err := uc.buildFolderPath(ctx, userID, folder.ParentID, cache)
	if err != nil {
		parentPath = "/"
	}
	fullPath := parentPath + folder.Name
	if parentPath != "/" {
		fullPath = parentPath + "/" + folder.Name
	}
	cache[*folderID] = fullPath
	return fullPath, nil
}

func (uc *cloudUseCase) isDescendantFolder(
	ctx context.Context,
	userID string,
	ancestorFolderID string,
	candidateParentID *string,
) (bool, error) {
	current := candidateParentID
	visited := map[string]bool{}
	for current != nil {
		if *current == ancestorFolderID {
			return true, nil
		}
		if visited[*current] {
			return false, fmt.Errorf("cycle detected in folder tree")
		}
		visited[*current] = true

		f, err := uc.folders.FindByID(ctx, userID, *current)
		if err != nil || f == nil {
			return false, fmt.Errorf("folder not found")
		}
		current = f.ParentID
	}
	return false, nil
}

func folderDepth(folder *domain.Folder, folders map[string]*domain.Folder) int {
	depth := 0
	current := folder.ParentID
	visited := map[string]bool{}
	for current != nil && *current != "" {
		if visited[*current] {
			break
		}
		visited[*current] = true
		depth++
		parent, ok := folders[*current]
		if !ok {
			break
		}
		current = parent.ParentID
	}
	return depth
}

func (uc *cloudUseCase) copyFileToFolder(ctx context.Context, userID string, source *domain.File, targetFolderID *string) error {
	if targetFolderID != nil {
		folder, err := uc.folders.FindByID(ctx, userID, *targetFolderID)
		if err != nil || folder == nil {
			return fmt.Errorf("target folder not found")
		}
	}

	resolvedName, err := uc.resolveFileName(ctx, userID, targetFolderID, source.Name)
	if err != nil {
		return fmt.Errorf("resolve filename: %w", err)
	}

	data, err := uc.storage.Read(source.StoragePath)
	if err != nil {
		return fmt.Errorf("read source file: %w", err)
	}

	newFileID := uuid.New().String()
	storagePath, err := uc.storage.Save(userID, newFileID, data)
	if err != nil {
		return fmt.Errorf("save copied file: %w", err)
	}

	now := time.Now()
	copied := &domain.File{
		ID:          newFileID,
		UserID:      userID,
		FolderID:    targetFolderID,
		Name:        resolvedName,
		MimeType:    source.MimeType,
		SizeBytes:   source.SizeBytes,
		StoragePath: storagePath,
		ThumbStatus: "pending",
		TakenAt:     source.TakenAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := uc.files.Create(ctx, copied); err != nil {
		_ = uc.storage.Delete(storagePath)
		return fmt.Errorf("create copied file record: %w", err)
	}

	if err := uc.files.UpdateStorageUsed(ctx, userID, source.SizeBytes); err != nil {
		return fmt.Errorf("update storage used: %w", err)
	}

	if uc.queue != nil {
		_ = uc.queue.EnqueueThumbnail(ctx, portout.ThumbnailTask{
			FileID:      newFileID,
			UserID:      userID,
			StoragePath: storagePath,
			MimeType:    source.MimeType,
		})
	}

	return nil
}

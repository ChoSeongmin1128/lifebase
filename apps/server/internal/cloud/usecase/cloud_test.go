package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"lifebase/internal/cloud/domain"
	portout "lifebase/internal/cloud/port/out"
)

type folderRepoStub struct {
	byID           map[string]*domain.Folder
	byParent       map[string][]*domain.Folder
	findErr        error
	findTrashedErr error
	listErr        error
	listByParentFn func(string, *string) ([]*domain.Folder, error)
	createErr      error
	updateErr      error
	softErr        error
	restoreErr     error
	restoreFn      func(string, string) error
	hardErr        error
	existsErr      error
	existsByName   map[string]bool
	existsByNameFn func(string, *string, string) (bool, error)
	softDeleted    []string
	restored       []string
	hardDeleted    []string
}

func newFolderRepoStub() *folderRepoStub {
	return &folderRepoStub{
		byID:         map[string]*domain.Folder{},
		byParent:     map[string][]*domain.Folder{},
		existsByName: map[string]bool{},
	}
}

func parentKey(id *string) string {
	if id == nil {
		return "__root__"
	}
	return *id
}

func (m *folderRepoStub) Create(_ context.Context, folder *domain.Folder) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.byID[folder.ID] = folder
	m.byParent[parentKey(folder.ParentID)] = append(m.byParent[parentKey(folder.ParentID)], folder)
	return nil
}
func (m *folderRepoStub) FindByID(_ context.Context, userID, id string) (*domain.Folder, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt != nil {
		return nil, errors.New("not found")
	}
	return f, nil
}
func (m *folderRepoStub) FindTrashedByID(_ context.Context, userID, id string) (*domain.Folder, error) {
	if m.findTrashedErr != nil {
		return nil, m.findTrashedErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt == nil {
		return nil, errors.New("not found")
	}
	return f, nil
}
func (m *folderRepoStub) ListByParent(_ context.Context, userID string, parentID *string) ([]*domain.Folder, error) {
	if m.listByParentFn != nil {
		return m.listByParentFn(userID, parentID)
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.Folder{}
	for _, f := range m.byParent[parentKey(parentID)] {
		if f.UserID == userID && f.DeletedAt == nil {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *folderRepoStub) Update(_ context.Context, folder *domain.Folder) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.byID[folder.ID] = folder
	return nil
}
func (m *folderRepoStub) SoftDelete(_ context.Context, userID, id string) error {
	if m.softErr != nil {
		return m.softErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt != nil {
		return errors.New("not found")
	}
	now := time.Now()
	f.DeletedAt = &now
	m.softDeleted = append(m.softDeleted, id)
	return nil
}
func (m *folderRepoStub) Restore(_ context.Context, userID, id string) error {
	if m.restoreFn != nil {
		if err := m.restoreFn(userID, id); err != nil {
			return err
		}
	}
	if m.restoreErr != nil {
		return m.restoreErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt == nil {
		return errors.New("not found")
	}
	f.DeletedAt = nil
	m.restored = append(m.restored, id)
	return nil
}
func (m *folderRepoStub) HardDelete(_ context.Context, id string) error {
	if m.hardErr != nil {
		return m.hardErr
	}
	delete(m.byID, id)
	m.hardDeleted = append(m.hardDeleted, id)
	return nil
}
func (m *folderRepoStub) ListTrashed(context.Context, string) ([]*domain.Folder, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.Folder{}
	for _, f := range m.byID {
		if f.DeletedAt != nil {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *folderRepoStub) ExistsByName(_ context.Context, userID string, parentID *string, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(userID, parentID, name)
	}
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.existsByName[fileNameKey(userID, parentID, name)], nil
}

type fileRepoStub struct {
	byID             map[string]*domain.File
	byFolder         map[string][]*domain.File
	recent           []*domain.File
	searchResult     []*domain.File
	findErr          error
	listErr          error
	createErr        error
	updateErr        error
	softErr          error
	restoreErr       error
	restoreFn        func(string, string) error
	hardErr          error
	updateStorageErr error
	existsByName     map[string]bool
	existsAlways     bool
	existsErr        error
	existsByNameFn   func(string, *string, string) (bool, error)
	listByFolderFn   func(string, *string, string, string) ([]*domain.File, error)
	findTrashedErr   error
	softDeleted      []string
	restored         []string
	hardDeleted      []string
	storageDeltas    []int64
}

func newFileRepoStub() *fileRepoStub {
	return &fileRepoStub{
		byID:         map[string]*domain.File{},
		byFolder:     map[string][]*domain.File{},
		existsByName: map[string]bool{},
	}
}

func fileNameKey(userID string, folderID *string, name string) string {
	if folderID == nil {
		return fmt.Sprintf("%s|root|%s", userID, name)
	}
	return fmt.Sprintf("%s|%s|%s", userID, *folderID, name)
}

func (m *fileRepoStub) Create(_ context.Context, file *domain.File) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.byID[file.ID] = file
	m.byFolder[parentKey(file.FolderID)] = append(m.byFolder[parentKey(file.FolderID)], file)
	return nil
}
func (m *fileRepoStub) FindByID(_ context.Context, userID, id string) (*domain.File, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt != nil {
		return nil, errors.New("not found")
	}
	return f, nil
}
func (m *fileRepoStub) ListByFolder(_ context.Context, userID string, folderID *string, _, _ string) ([]*domain.File, error) {
	if m.listByFolderFn != nil {
		return m.listByFolderFn(userID, folderID, "", "")
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.File{}
	for _, f := range m.byFolder[parentKey(folderID)] {
		if f.UserID == userID && f.DeletedAt == nil {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *fileRepoStub) ListRecent(context.Context, string, int) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.recent == nil {
		out := make([]*domain.File, 0, len(m.byID))
		for _, f := range m.byID {
			if f.DeletedAt == nil {
				out = append(out, f)
			}
		}
		return out, nil
	}
	return m.recent, nil
}
func (m *fileRepoStub) Update(_ context.Context, file *domain.File) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.byID[file.ID] = file
	return nil
}
func (m *fileRepoStub) SoftDelete(_ context.Context, userID, id string) error {
	if m.softErr != nil {
		return m.softErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt != nil {
		return errors.New("not found")
	}
	now := time.Now()
	f.DeletedAt = &now
	m.softDeleted = append(m.softDeleted, id)
	return nil
}
func (m *fileRepoStub) Restore(_ context.Context, userID, id string) error {
	if m.restoreFn != nil {
		if err := m.restoreFn(userID, id); err != nil {
			return err
		}
	}
	if m.restoreErr != nil {
		return m.restoreErr
	}
	f, ok := m.byID[id]
	if !ok || f.UserID != userID || f.DeletedAt == nil {
		return errors.New("not found")
	}
	f.DeletedAt = nil
	m.restored = append(m.restored, id)
	return nil
}
func (m *fileRepoStub) HardDelete(_ context.Context, id string) error {
	if m.hardErr != nil {
		return m.hardErr
	}
	if file, ok := m.byID[id]; ok {
		key := parentKey(file.FolderID)
		files := m.byFolder[key]
		next := files[:0]
		for _, candidate := range files {
			if candidate.ID != id {
				next = append(next, candidate)
			}
		}
		m.byFolder[key] = next
	}
	delete(m.byID, id)
	m.hardDeleted = append(m.hardDeleted, id)
	return nil
}
func (m *fileRepoStub) ListTrashed(context.Context, string) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.File{}
	for _, f := range m.byID {
		if f.DeletedAt != nil {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *fileRepoStub) UpdateStorageUsed(_ context.Context, _ string, delta int64) error {
	if m.updateStorageErr != nil {
		return m.updateStorageErr
	}
	m.storageDeltas = append(m.storageDeltas, delta)
	return nil
}
func (m *fileRepoStub) Search(context.Context, string, string, int) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.searchResult, nil
}
func (m *fileRepoStub) ExistsByName(_ context.Context, userID string, folderID *string, name string) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(userID, folderID, name)
	}
	if m.existsErr != nil {
		return false, m.existsErr
	}
	if m.existsAlways {
		return true, nil
	}
	return m.existsByName[fileNameKey(userID, folderID, name)], nil
}
func (m *fileRepoStub) FindTrashedByID(_ context.Context, userID, id string) (*domain.File, error) {
	if m.findTrashedErr != nil {
		return nil, m.findTrashedErr
	}
	f, ok := m.byID[id]
	if ok && f.UserID == userID && f.DeletedAt != nil {
		return f, nil
	}
	return nil, errors.New("not found")
}

type sharedRepoStub struct {
	folders []*domain.Folder
	err     error
}

func (m *sharedRepoStub) ListSharedFolders(context.Context, string) ([]*domain.Folder, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.folders, nil
}

type starRepoStub struct {
	refs     []portout.StarRef
	listErr  error
	setErr   error
	unsetErr error
}

func (m *starRepoStub) List(context.Context, string) ([]portout.StarRef, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.refs, nil
}
func (m *starRepoStub) Set(context.Context, string, string, string) error   { return m.setErr }
func (m *starRepoStub) Unset(context.Context, string, string, string) error { return m.unsetErr }

type storageStub struct {
	data      map[string][]byte
	savePath  string
	saveErr   error
	readErr   error
	deleteErr error
	deleted   []string
}

func newStorageStub() *storageStub { return &storageStub{data: map[string][]byte{}} }
func (m *storageStub) Save(userID, fileID string, data []byte) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	path := m.savePath
	if path == "" {
		path = fmt.Sprintf("%s/%s.bin", userID, fileID)
	}
	m.data[path] = data
	return path, nil
}
func (m *storageStub) Read(storagePath string) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	b, ok := m.data[storagePath]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}
func (m *storageStub) Delete(storagePath string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.data, storagePath)
	m.deleted = append(m.deleted, storagePath)
	return nil
}

type thumbnailStorageStub struct {
	deleteErr error
	deleted   []string
}

func (m *thumbnailStorageStub) Delete(userID, fileID string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, fmt.Sprintf("%s/%s", userID, fileID))
	return nil
}

type queueStub struct {
	tasks []portout.ThumbnailTask
	err   error
}

func (m *queueStub) EnqueueThumbnail(_ context.Context, task portout.ThumbnailTask) error {
	m.tasks = append(m.tasks, task)
	return m.err
}

func seedFolder(repo *folderRepoStub, id, userID, name string, parentID *string) *domain.Folder {
	f := &domain.Folder{ID: id, UserID: userID, Name: name, ParentID: parentID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	repo.byID[id] = f
	repo.byParent[parentKey(parentID)] = append(repo.byParent[parentKey(parentID)], f)
	return f
}

func seedFile(repo *fileRepoStub, id, userID, name, mime, storagePath string, folderID *string, size int64) *domain.File {
	f := &domain.File{
		ID: id, UserID: userID, Name: name, MimeType: mime, StoragePath: storagePath, FolderID: folderID,
		SizeBytes: size, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.byID[id] = f
	repo.byFolder[parentKey(folderID)] = append(repo.byFolder[parentKey(folderID)], f)
	return f
}

func newCloudUCForTest() (*cloudUseCase, *folderRepoStub, *fileRepoStub, *sharedRepoStub, *starRepoStub, *storageStub, *thumbnailStorageStub, *queueStub) {
	folders := newFolderRepoStub()
	files := newFileRepoStub()
	shared := &sharedRepoStub{}
	stars := &starRepoStub{}
	storage := newStorageStub()
	thumbs := &thumbnailStorageStub{}
	queue := &queueStub{}
	uc := NewCloudUseCase(folders, files, shared, stars, storage, thumbs, queue, "test-hmac").(*cloudUseCase)
	return uc, folders, files, shared, stars, storage, thumbs, queue
}

func markFolderTrashed(folder *domain.Folder) {
	now := time.Now()
	folder.DeletedAt = &now
}

func markFileTrashed(file *domain.File) {
	now := time.Now()
	file.DeletedAt = &now
}

func TestCloudUseCaseFolderFlows(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, _, _, _ := newCloudUCForTest()

	parent := seedFolder(folders, "parent", "u1", "Parent", nil)
	if _, err := uc.CreateFolder(ctx, "u1", strPtr("missing"), "child"); err == nil {
		t.Fatal("expected parent folder not found")
	}
	child, err := uc.CreateFolder(ctx, "u1", &parent.ID, "Child")
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID {
		t.Fatalf("unexpected parent: %#v", child.ParentID)
	}

	got, err := uc.GetFolder(ctx, "u1", parent.ID)
	if err != nil || got.ID != parent.ID {
		t.Fatalf("get folder failed: %v", err)
	}

	seedFile(files, "file1", "u1", "a.txt", "text/plain", "u1/file1", &parent.ID, 5)
	items, err := uc.ListFolder(ctx, "u1", &parent.ID, "name", "asc")
	if err != nil || len(items) != 2 {
		t.Fatalf("list folder failed: %v len=%d", err, len(items))
	}

	if err := uc.RenameFolder(ctx, "u1", "missing", "new"); err == nil {
		t.Fatal("expected folder not found")
	}
	if err := uc.RenameFolder(ctx, "u1", parent.ID, "Renamed"); err != nil {
		t.Fatalf("rename folder: %v", err)
	}

	if _, err := uc.MoveFolder(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected move folder not found")
	}
	if _, err := uc.MoveFolder(ctx, "u1", parent.ID, &parent.ID); err == nil {
		t.Fatal("expected cannot move into itself")
	}
	if _, err := uc.MoveFolder(ctx, "u1", parent.ID, strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}

	// descendant validation
	grand := seedFolder(folders, "grand", "u1", "Grand", &child.ID)
	if _, err := uc.MoveFolder(ctx, "u1", parent.ID, &grand.ID); err == nil {
		t.Fatal("expected cannot move folder into descendant")
	}

	if _, err := uc.MoveFolder(ctx, "u1", child.ID, &parent.ID); err != nil {
		t.Fatalf("same parent move should be no-op: %v", err)
	}

	folders.updateErr = errors.New("update fail")
	if _, err := uc.MoveFolder(ctx, "u1", child.ID, nil); err == nil {
		t.Fatal("expected move folder update error")
	}
	folders.updateErr = nil

	if err := uc.CopyFolder(ctx, "u1", parent.ID, nil); err == nil {
		t.Fatal("expected unsupported copy folder")
	}

	if err := uc.DeleteFolder(ctx, "u1", parent.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
	if parent.DeletedAt == nil || child.DeletedAt == nil || grand.DeletedAt == nil {
		t.Fatal("expected folder subtree to be soft deleted")
	}
	if len(files.softDeleted) != 1 || files.softDeleted[0] != "file1" {
		t.Fatalf("expected child files to be soft deleted, got %#v", files.softDeleted)
	}
}

func TestCloudUseCaseFileFlows(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, storage, thumbs, queue := newCloudUCForTest()
	folder := seedFolder(folders, "fold", "u1", "Fold", nil)
	targetFolder := seedFolder(folders, "fold-2", "u1", "Fold2", nil)
	seedFile(files, "source", "u1", "doc.txt", "text/plain", "u1/source.bin", &folder.ID, 4)
	storage.data["u1/source.bin"] = []byte("abcd")

	if _, err := uc.UploadFile(ctx, "u1", strPtr("none"), "doc.txt", "text/plain", 4, []byte("abcd")); err == nil {
		t.Fatal("expected folder not found")
	}
	files.existsByName[fileNameKey("u1", &folder.ID, "doc.txt")] = true
	files.existsByName[fileNameKey("u1", &folder.ID, "doc (1).txt")] = false
	uploaded, err := uc.UploadFile(ctx, "u1", &folder.ID, "doc.txt", "text/plain", 4, []byte("abcd"))
	if err != nil {
		t.Fatalf("upload file: %v", err)
	}
	if uploaded.Name != "doc (1).txt" {
		t.Fatalf("expected renamed upload, got %s", uploaded.Name)
	}
	if len(queue.tasks) == 0 {
		t.Fatal("thumbnail task should be enqueued")
	}

	files.existsErr = errors.New("exists fail")
	if _, err := uc.UploadFile(ctx, "u1", &folder.ID, "doc-error.txt", "text/plain", 4, []byte("abcd")); err == nil {
		t.Fatal("expected resolve filename error during upload")
	}
	files.existsErr = nil

	files.createErr = errors.New("create fail")
	if _, err := uc.UploadFile(ctx, "u1", &folder.ID, "x.txt", "text/plain", 1, []byte("x")); err == nil {
		t.Fatal("expected create file record error")
	}
	files.createErr = nil
	files.updateStorageErr = errors.New("update storage used fail")
	if _, err := uc.UploadFile(ctx, "u1", &folder.ID, "y.txt", "text/plain", 1, []byte("y")); err == nil {
		t.Fatal("expected update storage used error")
	}
	if len(files.hardDeleted) == 0 {
		t.Fatal("expected file record rollback after storage used update failure")
	}
	lastDeletedID := files.hardDeleted[len(files.hardDeleted)-1]
	if _, ok := files.byID[lastDeletedID]; ok {
		t.Fatal("expected rolled back file record to be removed")
	}
	if len(storage.deleted) == 0 {
		t.Fatal("expected storage cleanup after storage used update failure")
	}
	files.updateStorageErr = nil
	storage.saveErr = errors.New("save fail")
	if _, err := uc.UploadFile(ctx, "u1", &folder.ID, "z.txt", "text/plain", 1, []byte("z")); err == nil {
		t.Fatal("expected save file error")
	}
	storage.saveErr = nil

	if _, err := uc.GetFile(ctx, "u1", uploaded.ID); err != nil {
		t.Fatalf("get file: %v", err)
	}
	if _, _, err := uc.DownloadFile(ctx, "u1", "missing"); err == nil {
		t.Fatal("expected file not found")
	}
	storage.readErr = errors.New("read fail")
	if _, _, err := uc.DownloadFile(ctx, "u1", uploaded.ID); err == nil {
		t.Fatal("expected read file error")
	}
	storage.readErr = nil
	if data, file, err := uc.DownloadFile(ctx, "u1", uploaded.ID); err != nil || file.ID == "" || len(data) == 0 {
		t.Fatalf("download file failed: %v", err)
	}

	if !isEditableTextFile("a.md", "") || !isEditableTextFile("a.bin", "text/plain") || isEditableTextFile("a.bin", "application/octet-stream") {
		t.Fatal("isEditableTextFile checks failed")
	}

	binary := seedFile(files, "bin", "u1", "bin.bin", "application/octet-stream", "u1/bin", &folder.ID, 2)
	storage.data["u1/bin"] = []byte{0xff, 0xfe}
	if _, _, err := uc.GetFileContent(ctx, "u1", "missing"); err == nil {
		t.Fatal("expected file not found")
	}
	if _, _, err := uc.GetFileContent(ctx, "u1", binary.ID); err == nil {
		t.Fatal("expected not editable")
	}
	large := seedFile(files, "large", "u1", "big.txt", "text/plain", "u1/big", &folder.ID, maxEditableFileBytes+1)
	if _, _, err := uc.GetFileContent(ctx, "u1", large.ID); err == nil {
		t.Fatal("expected file too large")
	}
	utf := seedFile(files, "utf", "u1", "ok.txt", "text/plain", "u1/utf", &folder.ID, 2)
	storage.readErr = errors.New("read fail")
	if _, _, err := uc.GetFileContent(ctx, "u1", utf.ID); err == nil {
		t.Fatal("expected read error")
	}
	storage.readErr = nil
	storage.data["u1/utf"] = []byte{0xff, 0xfe}
	if _, _, err := uc.GetFileContent(ctx, "u1", utf.ID); err == nil {
		t.Fatal("expected invalid utf-8")
	}
	storage.data["u1/utf"] = []byte("hello")
	if content, _, err := uc.GetFileContent(ctx, "u1", utf.ID); err != nil || content != "hello" {
		t.Fatalf("get file content failed: %v content=%q", err, content)
	}

	if err := uc.RenameFile(ctx, "u1", "missing", "renamed.txt"); err == nil {
		t.Fatal("expected rename file not found")
	}
	if err := uc.RenameFile(ctx, "u1", utf.ID, "renamed.txt"); err != nil {
		t.Fatalf("rename file: %v", err)
	}

	if err := uc.UpdateFileContent(ctx, "u1", "missing", "a"); err == nil {
		t.Fatal("expected update file not found")
	}
	if err := uc.UpdateFileContent(ctx, "u1", binary.ID, "a"); err == nil {
		t.Fatal("expected update non-editable file error")
	}
	if err := uc.UpdateFileContent(ctx, "u1", utf.ID, string(make([]byte, maxEditableFileBytes+1))); err == nil {
		t.Fatal("expected content too large")
	}
	storage.saveErr = errors.New("save fail")
	if err := uc.UpdateFileContent(ctx, "u1", utf.ID, "ok"); err == nil {
		t.Fatal("expected save file error")
	}
	storage.saveErr = nil
	files.updateErr = errors.New("update metadata fail")
	if err := uc.UpdateFileContent(ctx, "u1", utf.ID, "ok"); err == nil {
		t.Fatal("expected update metadata error")
	}
	files.updateErr = nil
	files.updateStorageErr = errors.New("update storage fail")
	if err := uc.UpdateFileContent(ctx, "u1", utf.ID, "content changed"); err == nil {
		t.Fatal("expected update storage used error")
	}
	files.updateStorageErr = nil
	if err := uc.UpdateFileContent(ctx, "u1", utf.ID, "ok"); err != nil {
		t.Fatalf("update file content: %v", err)
	}

	if _, err := uc.MoveFile(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected move file not found")
	}
	if _, err := uc.MoveFile(ctx, "u1", utf.ID, strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}
	moveResult, err := uc.MoveFile(ctx, "u1", utf.ID, &targetFolder.ID)
	if err != nil {
		t.Fatalf("move file: %v", err)
	}
	if moveResult == nil || moveResult.UndoToken == "" {
		t.Fatal("expected move undo token")
	}

	if _, err := uc.CopyFile(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected file not found")
	}
	if _, err := uc.CopyFile(ctx, "u1", "source", strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}
	files.existsErr = errors.New("exists fail")
	if _, err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected resolve filename error")
	}
	files.existsErr = nil
	storage.readErr = errors.New("read fail")
	if _, err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected read source file error")
	}
	storage.readErr = nil
	storage.saveErr = errors.New("save fail")
	if _, err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected save copied file error")
	}
	storage.saveErr = nil
	files.createErr = errors.New("create copied fail")
	if _, err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected create copied file record error")
	}
	files.createErr = nil
	files.updateStorageErr = errors.New("update storage fail")
	if _, err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected update storage used error")
	}
	files.updateStorageErr = nil
	copyResult, err := uc.CopyFile(ctx, "u1", "source", &folder.ID)
	if err != nil {
		t.Fatalf("copy file: %v", err)
	}
	copied := copyResult.File
	if copied == nil || copied.ID == "" || copied.FolderID == nil || *copied.FolderID != folder.ID {
		t.Fatalf("unexpected copied file: %#v", copied)
	}
	if copyResult.UndoToken == "" {
		t.Fatal("expected copy undo token")
	}
	if err := uc.UndoOperation(ctx, "u2", copyResult.UndoToken); err == nil {
		t.Fatal("expected undo user mismatch error")
	}
	thumbs.deleteErr = errors.New("thumb delete fail")
	if err := uc.UndoOperation(ctx, "u1", copyResult.UndoToken); err == nil {
		t.Fatal("expected undo thumbnail delete error")
	}
	thumbs.deleteErr = nil

	storage.deleteErr = errors.New("delete storage fail")
	if err := uc.UndoOperation(ctx, "u1", copyResult.UndoToken); err == nil {
		t.Fatal("expected undo storage delete error")
	}
	storage.deleteErr = nil
	storage.data[copied.StoragePath] = []byte("source")
	files.hardErr = errors.New("hard delete fail")
	if err := uc.UndoOperation(ctx, "u1", copyResult.UndoToken); err == nil {
		t.Fatal("expected undo hard delete error")
	}
	files.hardErr = nil
	storage.data[copied.StoragePath] = []byte("source")
	files.updateStorageErr = errors.New("update storage fail")
	if err := uc.UndoOperation(ctx, "u1", copyResult.UndoToken); err == nil {
		t.Fatal("expected undo storage used error")
	}
	files.updateStorageErr = nil
	thumbs.deleted = nil
	copyResult, err = uc.CopyFile(ctx, "u1", "source", &folder.ID)
	if err != nil {
		t.Fatalf("copy file for undo retry: %v", err)
	}
	copied = copyResult.File
	if err := uc.UndoOperation(ctx, "u1", copyResult.UndoToken); err != nil {
		t.Fatalf("undo copy file: %v", err)
	}
	if len(thumbs.deleted) == 0 || thumbs.deleted[len(thumbs.deleted)-1] != fmt.Sprintf("u1/%s", copied.ID) {
		t.Fatalf("expected thumbnail cleanup, got %#v", thumbs.deleted)
	}

	if err := uc.DeleteFile(ctx, "u1", utf.ID); err != nil {
		t.Fatalf("delete file: %v", err)
	}
}

func TestCloudUseCaseTrashViewsStarsSearchAndHelpers(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, shared, stars, storage, _, _ := newCloudUCForTest()

	root := seedFolder(folders, "root", "u1", "Root", nil)
	child := seedFolder(folders, "child", "u1", "Child", &root.ID)
	grand := seedFolder(folders, "grand", "u1", "Grand", &child.ID)
	file1 := seedFile(files, "f1", "u1", "a.txt", "text/plain", "u1/a", &child.ID, 10)
	file2 := seedFile(files, "f2", "u1", "b.txt", "text/plain", "u1/b", nil, 20)
	file3 := seedFile(files, "f3", "u1", "c.txt", "text/plain", "u1/c", &grand.ID, 30)
	file4 := seedFile(files, "f4", "u1", "loose.txt", "text/plain", "u1/d", &root.ID, 40)
	markFolderTrashed(child)
	markFolderTrashed(grand)
	markFileTrashed(file1)
	markFileTrashed(file3)
	markFileTrashed(file4)
	storage.data["u1/a"] = []byte("a")
	storage.data["u1/b"] = []byte("b")
	storage.data["u1/c"] = []byte("c")
	storage.data["u1/d"] = []byte("d")

	if items, err := uc.ListTrash(ctx, "u1", nil); err != nil || len(items) != 2 {
		t.Fatalf("list trash failed: %v len=%d", err, len(items))
	}
	if items, err := uc.ListTrash(ctx, "u1", &child.ID); err != nil || len(items) != 2 {
		t.Fatalf("list trash folder contents failed: %v len=%d", err, len(items))
	}
	if _, err := uc.GetTrashFolder(ctx, "u1", child.ID); err != nil {
		t.Fatalf("get trash folder failed: %v", err)
	}
	folders.listErr = errors.New("folder list fail")
	if _, err := uc.ListTrash(ctx, "u1", nil); err == nil {
		t.Fatal("expected list trashed folders error")
	}
	folders.listErr = nil
	files.listErr = errors.New("file list fail")
	if _, err := uc.ListTrash(ctx, "u1", nil); err == nil {
		t.Fatal("expected list trashed files error")
	}
	files.listErr = nil

	if err := uc.RestoreItem(ctx, "u1", "id", "invalid"); err == nil {
		t.Fatal("expected invalid item type")
	}
	if err := uc.RestoreItem(ctx, "u1", "missing", "file"); err == nil {
		t.Fatal("expected file not found in trash")
	}
	files.findTrashedErr = nil
	files.existsErr = errors.New("resolve fail")
	if err := uc.RestoreItem(ctx, "u1", "f1", "file"); err == nil {
		t.Fatal("expected resolve filename error")
	}
	files.existsErr = nil
	files.existsByName[fileNameKey("u1", file1.FolderID, file1.Name)] = true
	files.existsByName[fileNameKey("u1", file1.FolderID, "a (1).txt")] = false
	files.updateErr = errors.New("rename fail")
	if err := uc.RestoreItem(ctx, "u1", "f1", "file"); err == nil {
		t.Fatal("expected rename on restore error")
	}
	files.updateErr = nil
	files.restoreErr = errors.New("restore fail")
	if err := uc.RestoreItem(ctx, "u1", "f1", "file"); err == nil {
		t.Fatal("expected restore file error")
	}
	files.restoreErr = nil
	if err := uc.RestoreItem(ctx, "u1", "f1", "file"); err != nil {
		t.Fatalf("restore file: %v", err)
	}
	if child.DeletedAt != nil {
		t.Fatal("expected parent folder path to restore with file restore")
	}
	files.existsByName[fileNameKey("u1", child.ParentID, child.Name)] = true
	files.existsByName[fileNameKey("u1", &child.ID, "unused")] = false
	folders.restoreErr = errors.New("restore folder fail")
	if err := uc.RestoreItem(ctx, "u1", "child", "folder"); err == nil {
		t.Fatal("expected restore folder error")
	}
	folders.restoreErr = nil
	markFolderTrashed(child)
	markFolderTrashed(grand)
	markFileTrashed(file1)
	markFileTrashed(file3)
	folders.existsByName[fileNameKey("u1", root.ParentID, root.Name)] = false
	folders.existsByName[fileNameKey("u1", child.ParentID, child.Name)] = true
	folders.existsByName[fileNameKey("u1", child.ParentID, "Child (1)")] = false
	files.existsByName[fileNameKey("u1", file1.FolderID, file1.Name)] = false
	files.existsByName[fileNameKey("u1", file3.FolderID, file3.Name)] = false
	if err := uc.RestoreItem(ctx, "u1", "child", "folder"); err != nil {
		t.Fatalf("restore folder subtree: %v", err)
	}
	if child.DeletedAt != nil || grand.DeletedAt != nil || file1.DeletedAt != nil || file3.DeletedAt != nil {
		t.Fatal("expected folder subtree to restore recursively")
	}
	if child.Name != "Child (1)" {
		t.Fatalf("expected restored folder to resolve name conflict, got %q", child.Name)
	}

	files.listErr = errors.New("list fail")
	if err := uc.EmptyTrash(ctx, "u1"); err == nil {
		t.Fatal("expected list trashed files error")
	}
	files.listErr = nil
	folders.listErr = errors.New("list folders fail")
	if err := uc.EmptyTrash(ctx, "u1"); err == nil {
		t.Fatal("expected list trashed folders error")
	}
	folders.listErr = nil
	markFolderTrashed(child)
	markFolderTrashed(grand)
	markFileTrashed(file1)
	markFileTrashed(file3)
	markFileTrashed(file4)
	if err := uc.EmptyTrash(ctx, "u1"); err != nil {
		t.Fatalf("empty trash: %v", err)
	}
	if len(files.hardDeleted) == 0 || len(folders.hardDeleted) == 0 {
		t.Fatal("expected trashed subtree to be hard deleted")
	}

	trashDeleteFolder := seedFolder(folders, "trash-delete-folder", "u1", "TrashDeleteFolder", nil)
	trashDeleteFile := seedFile(files, "trash-delete-file", "u1", "trash-delete.txt", "text/plain", "u1/trash-delete", &trashDeleteFolder.ID, 9)
	markFolderTrashed(trashDeleteFolder)
	markFileTrashed(trashDeleteFile)
	storage.data["u1/trash-delete"] = []byte("trash-delete")
	if err := uc.DeleteFile(ctx, "u1", trashDeleteFile.ID); err != nil {
		t.Fatalf("delete file from trash: %v", err)
	}
	if _, ok := files.byID[trashDeleteFile.ID]; ok {
		t.Fatal("expected trashed file to be hard deleted")
	}

	child = seedFolder(folders, "trash-delete-child", "u1", "TrashDeleteChild", nil)
	grand = seedFolder(folders, "trash-delete-grand", "u1", "TrashDeleteGrand", &child.ID)
	file3 = seedFile(files, "trash-delete-grand-file", "u1", "grand.txt", "text/plain", "u1/trash-delete-grand", &grand.ID, 13)
	activeNestedDelete := seedFolder(folders, "active-nested-delete", "u1", "ActiveNestedDelete", &child.ID)
	activeNestedFile := seedFile(files, "active-nested-file", "u1", "nested.txt", "text/plain", "u1/nested-delete", &activeNestedDelete.ID, 11)
	markFolderTrashed(child)
	markFolderTrashed(grand)
	markFileTrashed(file3)
	storage.data["u1/trash-delete-grand"] = []byte("grand")
	storage.data["u1/nested-delete"] = []byte("nested")
	if err := uc.DeleteFolder(ctx, "u1", child.ID); err != nil {
		t.Fatalf("delete folder from trash: %v", err)
	}
	for _, id := range []string{child.ID, grand.ID, activeNestedDelete.ID} {
		if _, ok := folders.byID[id]; ok {
			t.Fatalf("expected folder %s to be hard deleted via trash delete", id)
		}
	}
	for _, id := range []string{file3.ID, activeNestedFile.ID} {
		if _, ok := files.byID[id]; ok {
			t.Fatalf("expected file %s to be hard deleted via trash delete", id)
		}
	}

	files.recent = []*domain.File{file1, file2}
	if items, err := uc.ListRecent(ctx, "u1"); err != nil || len(items) != 2 {
		t.Fatalf("list recent failed: %v len=%d", err, len(items))
	}
	files.listErr = errors.New("list recent fail")
	if _, err := uc.ListRecent(ctx, "u1"); err == nil {
		t.Fatal("expected list recent error")
	}
	files.listErr = nil

	shared.folders = []*domain.Folder{root}
	if items, err := uc.ListShared(ctx, "u1"); err != nil || len(items) != 1 {
		t.Fatalf("list shared failed: %v len=%d", err, len(items))
	}
	shared.err = errors.New("shared fail")
	if _, err := uc.ListShared(ctx, "u1"); err == nil {
		t.Fatal("expected list shared error")
	}
	shared.err = nil

	stars.refs = []portout.StarRef{
		{ItemID: root.ID, ItemType: "folder"},
		{ItemID: file1.ID, ItemType: "file"},
		{ItemID: "ignored", ItemType: "unknown"},
	}
	if items, err := uc.ListStarred(ctx, "u1"); err != nil || len(items) != 1 {
		t.Fatalf("list starred failed: %v len=%d", err, len(items))
	}
	stars.listErr = errors.New("list stars fail")
	if _, err := uc.ListStarred(ctx, "u1"); err == nil {
		t.Fatal("expected list stars error")
	}
	stars.listErr = nil

	if items, err := uc.ListStars(ctx, "u1"); err != nil || len(items) == 0 {
		t.Fatalf("list stars failed: %v", err)
	}
	stars.listErr = errors.New("list stars fail")
	if _, err := uc.ListStars(ctx, "u1"); err == nil {
		t.Fatal("expected list stars error")
	}
	stars.listErr = nil

	if err := uc.StarItem(ctx, "u1", "id", "invalid"); err == nil {
		t.Fatal("expected invalid type")
	}
	if err := uc.StarItem(ctx, "u1", "missing", "folder"); err == nil {
		t.Fatal("expected folder not found")
	}
	if err := uc.StarItem(ctx, "u1", "missing", "file"); err == nil {
		t.Fatal("expected file not found")
	}
	stars.setErr = errors.New("set fail")
	if err := uc.StarItem(ctx, "u1", root.ID, "folder"); err == nil {
		t.Fatal("expected set star error")
	}
	stars.setErr = nil
	if err := uc.StarItem(ctx, "u1", root.ID, "folder"); err != nil {
		t.Fatalf("star folder: %v", err)
	}

	if err := uc.UnstarItem(ctx, "u1", root.ID, "invalid"); err == nil {
		t.Fatal("expected invalid unstar type")
	}
	stars.unsetErr = errors.New("unset fail")
	if err := uc.UnstarItem(ctx, "u1", root.ID, "folder"); err == nil {
		t.Fatal("expected unset error")
	}
	stars.unsetErr = nil
	if err := uc.UnstarItem(ctx, "u1", root.ID, "folder"); err != nil {
		t.Fatalf("unstar item: %v", err)
	}

	files.searchResult = []*domain.File{file1}
	if items, err := uc.Search(ctx, "u1", "a"); err != nil || len(items) != 1 {
		t.Fatalf("search failed: %v", err)
	}

	if len(toFolderItems([]*domain.Folder{root})) != 1 || len(toFileItems([]*domain.File{file1})) != 1 {
		t.Fatal("folder/file item converters failed")
	}

	liveFolder := seedFolder(folders, "live", "u1", "Live", nil)
	cache := map[string]string{}
	if p, err := uc.buildFolderPath(ctx, "u1", &liveFolder.ID, cache); err != nil || p == "" {
		t.Fatalf("build folder path failed: p=%q err=%v", p, err)
	}
	if p, err := uc.buildFolderPath(ctx, "u1", nil, cache); err != nil || p != "/" {
		t.Fatalf("root path expected '/', got %q err=%v", p, err)
	}

	// cycle detection in descendant check
	folders.byID["cycle-a"] = &domain.Folder{ID: "cycle-a", UserID: "u1", ParentID: strPtr("cycle-b"), Name: "a"}
	folders.byID["cycle-b"] = &domain.Folder{ID: "cycle-b", UserID: "u1", ParentID: strPtr("cycle-a"), Name: "b"}
	if _, err := uc.isDescendantFolder(ctx, "u1", "ancestor", strPtr("cycle-a")); err == nil {
		t.Fatal("expected cycle detected error")
	}
	if _, err := uc.buildFolderPath(ctx, "u1", strPtr("missing"), cache); err == nil {
		t.Fatal("expected build folder path error for missing folder")
	}
	files.existsAlways = true
	if _, err := uc.resolveFileName(ctx, "u1", nil, "dup.txt"); err == nil {
		t.Fatal("expected unique filename resolution exhaustion")
	}
	files.existsAlways = false
	if _, err := uc.MoveFolder(ctx, "u1", root.ID, strPtr("cycle-a")); err == nil {
		t.Fatal("expected move folder validation error")
	}
}

func TestCloudUseCaseTrashLegacySubtreeVisibilityAndCleanup(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, storage, _, _ := newCloudUCForTest()

	root := seedFolder(folders, "root", "u1", "Root", nil)
	trashed := seedFolder(folders, "trashed", "u1", "Trashed", &root.ID)
	nestedTrashed := seedFolder(folders, "nested-trashed", "u1", "Nested Trashed", &trashed.ID)
	activeNested := seedFolder(folders, "active-nested", "u1", "Active Nested", &trashed.ID)
	trashedFile := seedFile(files, "trashed-file", "u1", "trashed.txt", "text/plain", "u1/trashed", &trashed.ID, 10)
	activeFile := seedFile(files, "active-file", "u1", "active.txt", "text/plain", "u1/active", &trashed.ID, 20)
	activeDeepFile := seedFile(files, "active-deep-file", "u1", "deep.txt", "text/plain", "u1/deep", &activeNested.ID, 30)
	markFolderTrashed(trashed)
	markFolderTrashed(nestedTrashed)
	markFileTrashed(trashedFile)
	storage.data["u1/trashed"] = []byte("trashed")
	storage.data["u1/active"] = []byte("active")
	storage.data["u1/deep"] = []byte("deep")

	items, err := uc.ListTrash(ctx, "u1", &trashed.ID)
	if err != nil {
		t.Fatalf("list legacy trash subtree failed: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("expected 4 direct children in trash folder, got %d", len(items))
	}
	itemIDs := map[string]bool{}
	for _, item := range items {
		if item.Type == "folder" {
			itemIDs[item.Folder.ID] = true
		} else {
			itemIDs[item.File.ID] = true
		}
	}
	for _, id := range []string{nestedTrashed.ID, activeNested.ID, trashedFile.ID, activeFile.ID} {
		if !itemIDs[id] {
			t.Fatalf("expected trash folder to include %s", id)
		}
	}

	if _, err := uc.GetTrashFolder(ctx, "u1", activeNested.ID); err != nil {
		t.Fatalf("expected active descendant folder to resolve in trash: %v", err)
	}

	if err := uc.RestoreItem(ctx, "u1", activeFile.ID, "file"); err != nil {
		t.Fatalf("restore active file under trashed ancestor: %v", err)
	}
	if trashed.DeletedAt != nil {
		t.Fatal("expected trashed ancestor folder to restore when restoring active child file")
	}

	markFolderTrashed(trashed)
	markFolderTrashed(nestedTrashed)
	markFileTrashed(trashedFile)

	if err := uc.EmptyTrash(ctx, "u1"); err != nil {
		t.Fatalf("empty trash with legacy subtree failed: %v", err)
	}
	for _, id := range []string{trashed.ID, nestedTrashed.ID, activeNested.ID} {
		if _, ok := folders.byID[id]; ok {
			t.Fatalf("expected folder %s to be hard deleted", id)
		}
	}
	for _, id := range []string{trashedFile.ID, activeFile.ID, activeDeepFile.ID} {
		if _, ok := files.byID[id]; ok {
			t.Fatalf("expected file %s to be hard deleted", id)
		}
	}
}

func strPtr(s string) *string { return &s }

func TestCloudUseCaseAdditionalBranchCoverage(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
	root := seedFolder(folders, "root", "u1", "Root", nil)

	folders.createErr = errors.New("create fail")
	if _, err := uc.CreateFolder(ctx, "u1", nil, "x"); err == nil {
		t.Fatal("expected create folder repo error")
	}
	folders.createErr = nil

	folders.listErr = errors.New("list folder fail")
	if _, err := uc.ListFolder(ctx, "u1", nil, "name", "asc"); err == nil {
		t.Fatal("expected list folder repo error")
	}
	folders.listErr = nil
	files.listErr = errors.New("list file fail")
	if _, err := uc.ListFolder(ctx, "u1", nil, "name", "asc"); err == nil {
		t.Fatal("expected list file repo error")
	}
	files.listErr = nil

	cache := map[string]string{root.ID: "/cached-root"}
	if p, err := uc.buildFolderPath(ctx, "u1", &root.ID, cache); err != nil || p != "/cached-root" {
		t.Fatalf("expected cached path, got p=%q err=%v", p, err)
	}

	if ok, err := uc.isDescendantFolder(ctx, "u1", root.ID, nil); err != nil || ok {
		t.Fatalf("expected nil parent to be non-descendant, ok=%v err=%v", ok, err)
	}

	files.existsByName[fileNameKey("u1", nil, "dup.txt")] = true
	files.existsErr = errors.New("exists fail")
	if _, err := uc.resolveFileName(ctx, "u1", nil, "dup.txt"); err == nil {
		t.Fatal("expected resolve filename initial exists error")
	}
	files.existsErr = nil

	files.existsByNameFn = func(userID string, folderID *string, name string) (bool, error) {
		if name == "dup.txt" {
			return true, nil
		}
		return false, errors.New("candidate exists fail")
	}
	if _, err := uc.resolveFileName(ctx, "u1", nil, "dup.txt"); err == nil {
		t.Fatal("expected resolve filename candidate exists error")
	}
	files.existsByNameFn = nil

	leaf := seedFolder(folders, "leaf", "u1", "Leaf", &root.ID)
	if p, err := uc.buildFolderPath(ctx, "u1", &leaf.ID, map[string]string{}); err != nil || p != "/Root/Leaf" {
		t.Fatalf("expected recursive folder path, got p=%q err=%v", p, err)
	}

	orphanParent := "missing-parent"
	orphan := seedFolder(folders, "orphan", "u1", "Orphan", &orphanParent)
	if p, err := uc.buildFolderPath(ctx, "u1", &orphan.ID, map[string]string{}); err != nil || p != "/Orphan" {
		t.Fatalf("expected parent fallback path, got p=%q err=%v", p, err)
	}

	missing := "missing"
	if ok, err := uc.isDescendantFolder(ctx, "u1", root.ID, &missing); err == nil || ok {
		t.Fatalf("expected missing parent error, ok=%v err=%v", ok, err)
	}

	ucNoQueue, folders2, _, _, _, storage2, _, _ := newCloudUCForTest()
	ucNoQueue.queue = nil
	root2 := seedFolder(folders2, "root2", "u1", "Root2", nil)
	if _, err := ucNoQueue.UploadFile(ctx, "u1", &root2.ID, "root.txt", "text/plain", 1, []byte("x")); err != nil {
		t.Fatalf("expected upload without queue to succeed: %v", err)
	}
	if len(storage2.data) == 0 {
		t.Fatal("expected file to be saved without queue")
	}
	if _, err := ucNoQueue.UploadFile(ctx, "u1", nil, "rootless.txt", "text/plain", 1, []byte("y")); err != nil {
		t.Fatalf("expected root upload without folder to succeed: %v", err)
	}

	ucQueueErr, folders3, _, _, _, _, _, queue3 := newCloudUCForTest()
	root3 := seedFolder(folders3, "root3", "u1", "Root3", nil)
	queue3.err = errors.New("queue fail")
	if _, err := ucQueueErr.UploadFile(ctx, "u1", &root3.ID, "queued.txt", "text/plain", 1, []byte("z")); err != nil {
		t.Fatalf("expected upload success even when queue fails: %v", err)
	}
}

func TestCloudUseCaseTrashAndDeleteAdditionalErrorBranches(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, stars, _, _, _ := newCloudUCForTest()

	root := seedFolder(folders, "root", "u1", "Root", nil)
	child := seedFolder(folders, "child", "u1", "Child", &root.ID)
	seedFile(files, "child-file", "u1", "child.txt", "text/plain", "u1/child.txt", &child.ID, 10)

	if err := uc.DeleteFolder(ctx, "u1", "missing"); err == nil {
		t.Fatal("expected delete folder not found error")
	}

	files.listErr = errors.New("list files fail")
	if err := uc.DeleteFolder(ctx, "u1", root.ID); err == nil {
		t.Fatal("expected delete folder list files error")
	}
	files.listErr = nil

	folders.listErr = errors.New("list folders fail")
	if err := uc.DeleteFolder(ctx, "u1", root.ID); err == nil {
		t.Fatal("expected delete folder list folders error")
	}
	folders.listErr = nil

	files.softErr = errors.New("soft delete file fail")
	if err := uc.DeleteFolder(ctx, "u1", root.ID); err == nil {
		t.Fatal("expected delete folder file soft delete error")
	}
	files.softErr = nil

	files.byFolder[parentKey(&child.ID)] = nil
	folders.softErr = errors.New("soft delete folder fail")
	if err := uc.DeleteFolder(ctx, "u1", root.ID); err == nil {
		t.Fatal("expected delete folder folder soft delete error")
	}
	folders.softErr = nil

	if _, err := uc.GetTrashFolder(ctx, "u1", root.ID); err == nil {
		t.Fatal("expected get trash folder error")
	}

	if _, err := uc.ListTrash(ctx, "u1", strPtr("missing")); err == nil {
		t.Fatal("expected list trash invalid folder error")
	}

	stars.refs = []portout.StarRef{{ItemID: "missing", ItemType: "folder"}, {ItemID: "missing-file", ItemType: "file"}}
	if items, err := uc.ListStarred(ctx, "u1"); err != nil || len(items) != 0 {
		t.Fatalf("expected missing starred refs to be skipped, err=%v len=%d", err, len(items))
	}

	// findFolderInTrashScope: active folder path but no trashed ancestor
	if _, err := uc.findFolderInTrashScope(ctx, "u1", root.ID, nil); err == nil {
		t.Fatal("expected active folder outside trash to fail")
	}

	// hasTrashedAncestor: missing ancestor path
	missingParent := "missing-parent"
	if ok, err := uc.hasTrashedAncestor(ctx, "u1", &missingParent, nil); err == nil || ok {
		t.Fatalf("expected missing ancestor error, ok=%v err=%v", ok, err)
	}

	// restoreDeletedFolderPath: missing trashed folder path
	if err := uc.restoreDeletedFolderPath(ctx, "u1", &missingParent); err == nil {
		t.Fatal("expected restoreDeletedFolderPath missing error")
	}

	// restoreFolderSelf: resolve name error
	markFolderTrashed(child)
	folders.existsErr = errors.New("exists by name fail")
	if err := uc.restoreFolderSelf(ctx, "u1", child); err == nil {
		t.Fatal("expected restoreFolderSelf resolve name error")
	}
	folders.existsErr = nil

	// restoreFolderSubtree: root missing in trash map
	if err := uc.restoreFolderSubtree(ctx, "u1", "unknown-root", nil, nil); err == nil {
		t.Fatal("expected restoreFolderSubtree missing root error")
	}

	// restore file path: active file with no trashed ancestor should fail
	active := seedFile(files, "active", "u1", "a.txt", "text/plain", "u1/a.txt", nil, 1)
	if err := uc.RestoreItem(ctx, "u1", active.ID, "file"); err == nil {
		t.Fatal("expected restore active file outside trash to fail")
	}
}

func TestCloudUseCaseFolderNameExhaustionAndDepthCycle(t *testing.T) {
	ctx := context.Background()
	uc, folders, _, _, _, _, _, _ := newCloudUCForTest()

	folders.existsByName[fileNameKey("u1", nil, "dup")] = true
	for i := 1; i <= 10000; i++ {
		folders.existsByName[fileNameKey("u1", nil, fmt.Sprintf("dup (%d)", i))] = true
	}
	if _, err := uc.resolveFolderName(ctx, "u1", nil, "dup"); err == nil {
		t.Fatal("expected resolveFolderName exhaustion error")
	}

	cycleA := "cycle-a"
	cycleB := "cycle-b"
	folder := &domain.Folder{ID: cycleA, ParentID: &cycleB}
	foldersByID := map[string]*domain.Folder{
		cycleA: {ID: cycleA, ParentID: &cycleB},
		cycleB: {ID: cycleB, ParentID: &cycleA},
	}
	if got := folderDepth(folder, foldersByID); got != 2 {
		t.Fatalf("expected folderDepth to stop on cycle with depth 2, got %d", got)
	}
}

func TestCloudUseCaseRestoreAndTraversalAdditionalBranches(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, stars, _, _, _ := newCloudUCForTest()

	parentID := "parent"
	if ok, err := uc.hasTrashedAncestor(ctx, "u1", &parentID, map[string]struct{}{"parent": {}}); err != nil || !ok {
		t.Fatalf("expected trashed ancestor hit via id map, ok=%v err=%v", ok, err)
	}

	folders.byID["cycle-a"] = &domain.Folder{ID: "cycle-a", UserID: "u1", ParentID: strPtr("cycle-b"), Name: "A"}
	folders.byID["cycle-b"] = &domain.Folder{ID: "cycle-b", UserID: "u1", ParentID: strPtr("cycle-a"), Name: "B"}
	if _, err := uc.hasTrashedAncestor(ctx, "u1", strPtr("cycle-a"), nil); err == nil {
		t.Fatal("expected hasTrashedAncestor cycle error")
	}

	active := seedFolder(folders, "active", "u1", "Active", nil)
	if err := uc.restoreDeletedFolderPath(ctx, "u1", &active.ID); err != nil {
		t.Fatalf("expected already-active path to restore without error: %v", err)
	}

	trashed := seedFolder(folders, "trashed", "u1", "Trashed", nil)
	markFolderTrashed(trashed)
	if err := uc.restoreFolderSelf(ctx, "u1", trashed); err != nil {
		t.Fatalf("restoreFolderSelf success failed: %v", err)
	}
	markFolderTrashed(trashed)
	folders.restoreErr = errors.New("restore fail")
	if err := uc.restoreFolderSelf(ctx, "u1", trashed); err == nil {
		t.Fatal("expected restoreFolderSelf restore error")
	}
	folders.restoreErr = nil

	root := seedFolder(folders, "sub-root", "u1", "SubRoot", nil)
	markFolderTrashed(root)
	file := seedFile(files, "sub-file", "u1", "dup.txt", "text/plain", "u1/dup", &root.ID, 1)
	markFileTrashed(file)
	files.existsByName[fileNameKey("u1", &root.ID, "dup.txt")] = true
	files.updateErr = errors.New("update fail")
	if err := uc.restoreFolderSubtree(ctx, "u1", root.ID, []*domain.Folder{root}, []*domain.File{file}); err == nil {
		t.Fatal("expected restoreFolderSubtree file rename update error")
	}
	files.updateErr = nil
	files.restoreErr = errors.New("restore fail")
	if err := uc.restoreFolderSubtree(ctx, "u1", root.ID, []*domain.Folder{root}, []*domain.File{file}); err == nil {
		t.Fatal("expected restoreFolderSubtree file restore error")
	}
	files.restoreErr = nil

	files.listErr = errors.New("list by folder fail")
	if _, _, err := uc.collectActiveFolderTree(ctx, "u1", active); err == nil {
		t.Fatal("expected collectActiveFolderTree file list error")
	}
	files.listErr = nil
	folders.listErr = errors.New("list by parent fail")
	if _, _, err := uc.collectActiveFolderTree(ctx, "u1", active); err == nil {
		t.Fatal("expected collectActiveFolderTree folder list error")
	}
	folders.listErr = nil

	stars.refs = []portout.StarRef{{ItemID: "x", ItemType: "unknown"}}
	items, err := uc.ListStarred(ctx, "u1")
	if err != nil {
		t.Fatalf("ListStarred unknown type should be ignored without error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected unknown star type to be ignored, got %d items", len(items))
	}
}

func TestCloudUseCaseTargetedGapBranches(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, stars, _, _, _ := newCloudUCForTest()

	root := seedFolder(folders, "gap-root", "u1", "GapRoot", nil)
	child := seedFolder(folders, "gap-child", "u1", "GapChild", &root.ID)
	fileInChild := seedFile(files, "gap-file", "u1", "gap.txt", "text/plain", "u1/gap.txt", &child.ID, 2)

	// resolveFolderName: duplicate resolution on non-root parent
	folders.existsByName[fileNameKey("u1", &root.ID, "GapChild")] = true
	if resolved, err := uc.resolveFolderName(ctx, "u1", &root.ID, "GapChild"); err != nil || resolved != "GapChild (1)" {
		t.Fatalf("expected duplicate folder name resolution, got=%q err=%v", resolved, err)
	}

	// ListTrash: folderID path with active children load errors
	markFolderTrashed(root)
	if _, err := uc.ListTrash(ctx, "u1", &root.ID); err != nil {
		t.Fatalf("expected list trash success before forcing errors: %v", err)
	}
	folders.listErr = errors.New("active folder list fail")
	if _, err := uc.ListTrash(ctx, "u1", &root.ID); err == nil {
		t.Fatal("expected list trash active folder list error")
	}
	folders.listErr = nil
	files.listErr = errors.New("active file list fail")
	if _, err := uc.ListTrash(ctx, "u1", &root.ID); err == nil {
		t.Fatal("expected list trash active file list error")
	}
	files.listErr = nil

	// RestoreItem(file): resolve filename failure branch
	markFileTrashed(fileInChild)
	files.existsErr = errors.New("resolve name fail")
	if err := uc.RestoreItem(ctx, "u1", fileInChild.ID, "file"); err == nil {
		t.Fatal("expected restore item file resolve name error")
	}
	files.existsErr = nil

	// RestoreItem(folder): active folder branch with successful ancestor restore path
	activeChild := seedFolder(folders, "gap-active-child", "u1", "ActiveChild", &root.ID)
	if err := uc.RestoreItem(ctx, "u1", activeChild.ID, "folder"); err != nil {
		t.Fatalf("expected restore active folder in trash scope success: %v", err)
	}

	// restoreDeletedFolderPath: recursive parent restore failure branch
	trashedParent := seedFolder(folders, "gap-trashed-parent", "u1", "TP", nil)
	markFolderTrashed(trashedParent)
	restoreTarget := seedFolder(folders, "gap-restore-target", "u1", "Target", &trashedParent.ID)
	markFolderTrashed(restoreTarget)
	folders.restoreErr = errors.New("restore parent fail")
	if err := uc.restoreDeletedFolderPath(ctx, "u1", &restoreTarget.ID); err == nil {
		t.Fatal("expected restoreDeletedFolderPath recursive restore error")
	}
	folders.restoreErr = nil

	// restoreFolderSelf: rename needed + update error branch
	renameFolder := seedFolder(folders, "gap-rename-folder", "u1", "DupFolder", nil)
	markFolderTrashed(renameFolder)
	folders.existsByName[fileNameKey("u1", nil, "DupFolder")] = true
	folders.updateErr = errors.New("folder update fail")
	if err := uc.restoreFolderSelf(ctx, "u1", renameFolder); err == nil {
		t.Fatal("expected restoreFolderSelf rename update error")
	}
	folders.updateErr = nil
	delete(folders.existsByName, fileNameKey("u1", nil, "DupFolder"))

	// restoreFolderSubtree: child folder recursion branch
	subRoot := seedFolder(folders, "gap-sub-root", "u1", "SubRoot", nil)
	markFolderTrashed(subRoot)
	subChild := seedFolder(folders, "gap-sub-child", "u1", "SubChild", &subRoot.ID)
	markFolderTrashed(subChild)
	subFile := seedFile(files, "gap-sub-file", "u1", "sub.txt", "text/plain", "u1/sub.txt", &subChild.ID, 3)
	markFileTrashed(subFile)
	if err := uc.restoreFolderSubtree(ctx, "u1", subRoot.ID, []*domain.Folder{subRoot, subChild}, []*domain.File{subFile}); err != nil {
		t.Fatalf("expected restoreFolderSubtree recursive success: %v", err)
	}

	// collectActiveFolderTree: recursive success branch
	activeRoot := seedFolder(folders, "gap-active-root", "u1", "ActiveRoot", nil)
	activeSub := seedFolder(folders, "gap-active-sub", "u1", "ActiveSub", &activeRoot.ID)
	seedFile(files, "gap-active-root-file", "u1", "a.txt", "text/plain", "u1/a.txt", &activeRoot.ID, 1)
	seedFile(files, "gap-active-sub-file", "u1", "b.txt", "text/plain", "u1/b.txt", &activeSub.ID, 1)
	colFolders, colFiles, err := uc.collectActiveFolderTree(ctx, "u1", activeRoot)
	if err != nil {
		t.Fatalf("collectActiveFolderTree recursive success failed: %v", err)
	}
	if len(colFolders) < 2 || len(colFiles) < 2 {
		t.Fatalf("expected recursive tree collection, folders=%d files=%d", len(colFolders), len(colFiles))
	}

	// EmptyTrash: collectActiveFolderTree error propagation
	errFolder := seedFolder(folders, "gap-empty-err-root", "u1", "ErrRoot", nil)
	markFolderTrashed(errFolder)
	files.listErr = errors.New("empty trash collect fail")
	if err := uc.EmptyTrash(ctx, "u1"); err == nil {
		t.Fatal("expected EmptyTrash collect tree error")
	}
	files.listErr = nil

	// ListStarred: include both folder and file success entries
	stars.refs = []portout.StarRef{
		{ItemID: activeRoot.ID, ItemType: "folder"},
		{ItemID: fileInChild.ID, ItemType: "file"},
	}
	fileInChild.DeletedAt = nil
	items, err := uc.ListStarred(ctx, "u1")
	if err != nil {
		t.Fatalf("ListStarred mixed refs failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 starred items, got %d", len(items))
	}

	// resolveFolderName: error on candidate existence check inside loop
	checkCalls := 0
	folders.existsByNameFn = func(userID string, parentID *string, name string) (bool, error) {
		checkCalls++
		if checkCalls == 1 {
			return true, nil
		}
		return false, errors.New("candidate exists check fail")
	}
	if _, err := uc.resolveFolderName(ctx, "u1", nil, "LoopErr"); err == nil {
		t.Fatal("expected resolveFolderName loop exists check error")
	}
	folders.existsByNameFn = nil

	// RestoreItem(folder): list trashed folders/files error branches
	restoreListRoot := seedFolder(folders, "gap-restore-list-root", "u1", "RestoreListRoot", nil)
	markFolderTrashed(restoreListRoot)
	folders.listErr = errors.New("list trashed folders fail")
	if err := uc.RestoreItem(ctx, "u1", restoreListRoot.ID, "folder"); err == nil {
		t.Fatal("expected restore folder list trashed folders error")
	}
	folders.listErr = nil
	files.listErr = errors.New("list trashed files fail")
	if err := uc.RestoreItem(ctx, "u1", restoreListRoot.ID, "folder"); err == nil {
		t.Fatal("expected restore folder list trashed files error")
	}
	files.listErr = nil

	// restoreFolderSubtree: resolve filename error branch
	resolveErrRoot := seedFolder(folders, "gap-resolve-err-root", "u1", "ResolveErrRoot", nil)
	markFolderTrashed(resolveErrRoot)
	resolveErrFile := seedFile(files, "gap-resolve-err-file", "u1", "resolve.txt", "text/plain", "u1/resolve.txt", &resolveErrRoot.ID, 1)
	markFileTrashed(resolveErrFile)
	files.existsErr = errors.New("resolve filename fail")
	if err := uc.restoreFolderSubtree(ctx, "u1", resolveErrRoot.ID, []*domain.Folder{resolveErrRoot}, []*domain.File{resolveErrFile}); err == nil {
		t.Fatal("expected restoreFolderSubtree resolve filename error")
	}
	files.existsErr = nil

	// collectActiveFolderTree: recursive child traversal failure branch
	recRoot := seedFolder(folders, "gap-rec-root", "u1", "RecRoot", nil)
	recChild := seedFolder(folders, "gap-rec-child", "u1", "RecChild", &recRoot.ID)
	seedFile(files, "gap-rec-file", "u1", "rec.txt", "text/plain", "u1/rec.txt", &recRoot.ID, 1)
	files.listByFolderFn = func(_ string, folderID *string, _, _ string) ([]*domain.File, error) {
		if folderID != nil && *folderID == recChild.ID {
			return nil, errors.New("recursive child file list fail")
		}
		return files.byFolder[parentKey(folderID)], nil
	}
	if _, _, err := uc.collectActiveFolderTree(ctx, "u1", recRoot); err == nil {
		t.Fatal("expected collectActiveFolderTree recursive child error")
	}
	files.listByFolderFn = nil
}

func TestCloudUseCaseLastGapBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListTrash duplicate active entries are skipped", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		trashed := seedFolder(folders, "trashed", "u1", "Trashed", &root.ID)
		trashedFile := seedFile(files, "trashed-file", "u1", "a.txt", "text/plain", "u1/a", &trashed.ID, 1)
		markFolderTrashed(trashed)
		markFileTrashed(trashedFile)

		folders.byParent[parentKey(&trashed.ID)] = append(folders.byParent[parentKey(&trashed.ID)],
			&domain.Folder{ID: trashed.ID, UserID: "u1", ParentID: &trashed.ID, Name: "duplicate"})
		files.byFolder[parentKey(&trashed.ID)] = append(files.byFolder[parentKey(&trashed.ID)],
			&domain.File{ID: trashedFile.ID, UserID: "u1", FolderID: &trashed.ID, Name: "duplicate.txt"})

		items, err := uc.ListTrash(ctx, "u1", &trashed.ID)
		if err != nil {
			t.Fatalf("ListTrash failed: %v", err)
		}
		counts := map[string]int{}
		for _, item := range items {
			if item.Type == "folder" {
				counts[item.Folder.ID]++
			} else {
				counts[item.File.ID]++
			}
		}
		if counts[trashed.ID] > 1 || counts[trashedFile.ID] > 1 {
			t.Fatalf("expected duplicate active entries to be skipped, counts=%#v", counts)
		}
	})

	t.Run("RestoreItem file restoreDeletedFolderPath error", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		child := seedFolder(folders, "child", "u1", "Child", &root.ID)
		file := seedFile(files, "file", "u1", "a.txt", "text/plain", "u1/a", &child.ID, 1)
		markFolderTrashed(root)
		markFolderTrashed(child)
		markFileTrashed(file)
		folders.restoreErr = errors.New("restore path fail")
		if err := uc.RestoreItem(ctx, "u1", file.ID, "file"); err == nil || err.Error() != "restore folder: restore path fail" {
			t.Fatalf("expected restore path error, got %v", err)
		}
	})

	t.Run("restoreDeletedFolderPath nil and empty noop", func(t *testing.T) {
		uc, _, _, _, _, _, _, _ := newCloudUCForTest()
		if err := uc.restoreDeletedFolderPath(ctx, "u1", nil); err != nil {
			t.Fatalf("expected nil path noop: %v", err)
		}
		empty := ""
		if err := uc.restoreDeletedFolderPath(ctx, "u1", &empty); err != nil {
			t.Fatalf("expected empty path noop: %v", err)
		}
	})
}

func TestCloudUseCaseExhaustiveGapBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("ListTrash active folder and file errors plus duplicate skips", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		trashed := seedFolder(folders, "trashed", "u1", "Trashed", &root.ID)
		seenFolder := seedFolder(folders, "seen-folder", "u1", "Seen Folder", &trashed.ID)
		markFolderTrashed(seenFolder)
		visibleFolder := &domain.Folder{ID: "active-folder", UserID: "u1", ParentID: &trashed.ID, Name: "Active Folder"}
		duplicateFolder := &domain.Folder{ID: seenFolder.ID, UserID: "u1", ParentID: &trashed.ID, Name: "Duplicate Folder"}
		visibleFile := &domain.File{ID: "active-file", UserID: "u1", FolderID: &trashed.ID, Name: "active.txt"}
		duplicateFile := &domain.File{ID: "seen-file", UserID: "u1", FolderID: &trashed.ID, Name: "dup.txt"}
		seenFile := seedFile(files, duplicateFile.ID, "u1", duplicateFile.Name, "text/plain", "u1/dup", &trashed.ID, 1)
		markFolderTrashed(trashed)
		markFileTrashed(seenFile)
		folders.byParent[parentKey(&trashed.ID)] = append(folders.byParent[parentKey(&trashed.ID)], visibleFolder, duplicateFolder)
		files.byFolder[parentKey(&trashed.ID)] = append(files.byFolder[parentKey(&trashed.ID)], visibleFile, duplicateFile)

		items, err := uc.ListTrash(ctx, "u1", &trashed.ID)
		if err != nil {
			t.Fatalf("ListTrash failed: %v", err)
		}
		counts := map[string]int{}
		for _, item := range items {
			if item.Type == "folder" {
				counts[item.Folder.ID]++
			} else {
				counts[item.File.ID]++
			}
		}
		if counts[visibleFolder.ID] != 1 || counts[visibleFile.ID] != 1 || counts[trashed.ID] > 1 || counts[duplicateFile.ID] > 1 {
			t.Fatalf("unexpected ListTrash counts: %#v", counts)
		}

		folders.listByParentFn = func(string, *string) ([]*domain.Folder, error) {
			return nil, errors.New("active folder list fail")
		}
		if _, err := uc.ListTrash(ctx, "u1", &trashed.ID); err == nil || err.Error() != "active folder list fail" {
			t.Fatalf("expected active folder list error, got %v", err)
		}
		folders.listByParentFn = nil
		files.listByFolderFn = func(string, *string, string, string) ([]*domain.File, error) {
			return nil, errors.New("active file list fail")
		}
		if _, err := uc.ListTrash(ctx, "u1", &trashed.ID); err == nil || err.Error() != "active file list fail" {
			t.Fatalf("expected active file list error, got %v", err)
		}
		files.listByFolderFn = nil
	})

	t.Run("RestoreItem folder path and listing errors", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		parent := seedFolder(folders, "parent", "u1", "Parent", &root.ID)
		child := seedFolder(folders, "child", "u1", "Child", &parent.ID)
		markFolderTrashed(parent)
		markFolderTrashed(child)

		folders.restoreFn = func(_ string, id string) error {
			if id == parent.ID {
				return errors.New("restore parent fail")
			}
			return nil
		}
		if err := uc.RestoreItem(ctx, "u1", child.ID, "folder"); err == nil || err.Error() != "restore folder: restore parent fail" {
			t.Fatalf("expected restoreDeletedFolderPath propagated error, got %v", err)
		}
		folders.restoreFn = nil
		markFolderTrashed(parent)
		markFolderTrashed(child)

		folders.listErr = errors.New("list trashed folders fail")
		if err := uc.RestoreItem(ctx, "u1", child.ID, "folder"); err == nil || err.Error() != "list trashed folders fail" {
			t.Fatalf("expected trashed folders list error, got %v", err)
		}
		folders.listErr = nil
		files.listErr = errors.New("list trashed files fail")
		if err := uc.RestoreItem(ctx, "u1", child.ID, "folder"); err == nil || err.Error() != "list trashed files fail" {
			t.Fatalf("expected trashed files list error, got %v", err)
		}
	})

	t.Run("EmptyTrash collectActiveFolderTree error", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		child := seedFolder(folders, "child", "u1", "Child", &root.ID)
		seedFile(files, "active-file", "u1", "a.txt", "text/plain", "u1/a", &child.ID, 1)
		markFolderTrashed(root)
		files.listByFolderFn = func(_ string, folderID *string, _, _ string) ([]*domain.File, error) {
			if folderID != nil && *folderID == root.ID {
				return []*domain.File{}, nil
			}
			return nil, errors.New("collect tree fail")
		}
		if err := uc.EmptyTrash(ctx, "u1"); err == nil || err.Error() != "collect tree fail" {
			t.Fatalf("expected collectActiveFolderTree error, got %v", err)
		}
	})

	t.Run("restoreFolderSubtree child recursion and file restore error", func(t *testing.T) {
		uc, folders, files, _, _, _, _, _ := newCloudUCForTest()
		root := seedFolder(folders, "root", "u1", "Root", nil)
		child := seedFolder(folders, "child", "u1", "Child", &root.ID)
		grand := seedFolder(folders, "grand", "u1", "Grand", &child.ID)
		file := seedFile(files, "file", "u1", "a.txt", "text/plain", "u1/a", &root.ID, 1)
		markFolderTrashed(root)
		markFolderTrashed(child)
		markFolderTrashed(grand)
		markFileTrashed(file)

		folders.restoreFn = func(_ string, id string) error {
			if id == child.ID {
				return errors.New("child restore fail")
			}
			return nil
		}
		if err := uc.restoreFolderSubtree(ctx, "u1", root.ID, []*domain.Folder{root, child, grand}, nil); err == nil || err.Error() != "restore folder: child restore fail" {
			t.Fatalf("expected child recursion error, got %v", err)
		}

		folders.restoreFn = nil
		markFolderTrashed(root)
		markFileTrashed(file)
		files.restoreFn = func(_ string, id string) error {
			if id == file.ID {
				return errors.New("file restore fail")
			}
			return nil
		}
		if err := uc.restoreFolderSubtree(ctx, "u1", root.ID, []*domain.Folder{root}, []*domain.File{file}); err == nil || err.Error() != "restore file: file restore fail" {
			t.Fatalf("expected file restore error, got %v", err)
		}
	})
}

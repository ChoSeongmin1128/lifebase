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
	byID       map[string]*domain.Folder
	byParent   map[string][]*domain.Folder
	trashed    []*domain.Folder
	findErr    error
	listErr    error
	createErr  error
	updateErr  error
	softErr    error
	restoreErr error
	hardErr    error
}

func newFolderRepoStub() *folderRepoStub {
	return &folderRepoStub{
		byID:     map[string]*domain.Folder{},
		byParent: map[string][]*domain.Folder{},
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
	if !ok || f.UserID != userID {
		return nil, errors.New("not found")
	}
	return f, nil
}
func (m *folderRepoStub) ListByParent(_ context.Context, userID string, parentID *string) ([]*domain.Folder, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.Folder{}
	for _, f := range m.byParent[parentKey(parentID)] {
		if f.UserID == userID {
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
func (m *folderRepoStub) SoftDelete(context.Context, string, string) error { return m.softErr }
func (m *folderRepoStub) Restore(context.Context, string, string) error    { return m.restoreErr }
func (m *folderRepoStub) HardDelete(context.Context, string) error         { return m.hardErr }
func (m *folderRepoStub) ListTrashed(context.Context, string) ([]*domain.Folder, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.trashed, nil
}

type fileRepoStub struct {
	byID             map[string]*domain.File
	byFolder         map[string][]*domain.File
	trashed          []*domain.File
	recent           []*domain.File
	searchResult     []*domain.File
	findErr          error
	listErr          error
	createErr        error
	updateErr        error
	softErr          error
	restoreErr       error
	hardErr          error
	updateStorageErr error
	existsByName     map[string]bool
	existsAlways     bool
	existsErr        error
	existsByNameFn   func(string, *string, string) (bool, error)
	findTrashedErr   error
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
	if !ok || f.UserID != userID {
		return nil, errors.New("not found")
	}
	return f, nil
}
func (m *fileRepoStub) ListByFolder(_ context.Context, userID string, folderID *string, _, _ string) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := []*domain.File{}
	for _, f := range m.byFolder[parentKey(folderID)] {
		if f.UserID == userID {
			out = append(out, f)
		}
	}
	return out, nil
}
func (m *fileRepoStub) ListRecent(context.Context, string, int) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
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
func (m *fileRepoStub) SoftDelete(context.Context, string, string) error { return m.softErr }
func (m *fileRepoStub) Restore(context.Context, string, string) error    { return m.restoreErr }
func (m *fileRepoStub) HardDelete(context.Context, string) error         { return m.hardErr }
func (m *fileRepoStub) ListTrashed(context.Context, string) ([]*domain.File, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.trashed, nil
}
func (m *fileRepoStub) UpdateStorageUsed(context.Context, string, int64) error { return m.updateStorageErr }
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
	for _, f := range m.trashed {
		if f.ID == id && f.UserID == userID {
			return f, nil
		}
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
	refs      []portout.StarRef
	listErr   error
	setErr    error
	unsetErr  error
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

func newCloudUCForTest() (*cloudUseCase, *folderRepoStub, *fileRepoStub, *sharedRepoStub, *starRepoStub, *storageStub, *queueStub) {
	folders := newFolderRepoStub()
	files := newFileRepoStub()
	shared := &sharedRepoStub{}
	stars := &starRepoStub{}
	storage := newStorageStub()
	queue := &queueStub{}
	uc := NewCloudUseCase(folders, files, shared, stars, storage, queue).(*cloudUseCase)
	return uc, folders, files, shared, stars, storage, queue
}

func TestCloudUseCaseFolderFlows(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, _, _ := newCloudUCForTest()

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

	if err := uc.MoveFolder(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected move folder not found")
	}
	if err := uc.MoveFolder(ctx, "u1", parent.ID, &parent.ID); err == nil {
		t.Fatal("expected cannot move into itself")
	}
	if err := uc.MoveFolder(ctx, "u1", parent.ID, strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}

	// descendant validation
	grand := seedFolder(folders, "grand", "u1", "Grand", &child.ID)
	if err := uc.MoveFolder(ctx, "u1", parent.ID, &grand.ID); err == nil {
		t.Fatal("expected cannot move folder into descendant")
	}

	if err := uc.MoveFolder(ctx, "u1", child.ID, &parent.ID); err != nil {
		t.Fatalf("same parent move should be no-op: %v", err)
	}

	folders.updateErr = errors.New("update fail")
	if err := uc.MoveFolder(ctx, "u1", child.ID, nil); err == nil {
		t.Fatal("expected move folder update error")
	}
	folders.updateErr = nil

	if err := uc.CopyFolder(ctx, "u1", parent.ID, nil); err == nil {
		t.Fatal("expected unsupported copy folder")
	}

	if err := uc.DeleteFolder(ctx, "u1", parent.ID); err != nil {
		t.Fatalf("delete folder: %v", err)
	}
}

func TestCloudUseCaseFileFlows(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, storage, queue := newCloudUCForTest()
	folder := seedFolder(folders, "fold", "u1", "Fold", nil)
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

	if err := uc.MoveFile(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected move file not found")
	}
	if err := uc.MoveFile(ctx, "u1", utf.ID, strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}
	if err := uc.MoveFile(ctx, "u1", utf.ID, &folder.ID); err != nil {
		t.Fatalf("move file: %v", err)
	}

	if err := uc.CopyFile(ctx, "u1", "missing", nil); err == nil {
		t.Fatal("expected file not found")
	}
	if err := uc.CopyFile(ctx, "u1", "source", strPtr("none")); err == nil {
		t.Fatal("expected target folder not found")
	}
	files.existsErr = errors.New("exists fail")
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected resolve filename error")
	}
	files.existsErr = nil
	storage.readErr = errors.New("read fail")
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected read source file error")
	}
	storage.readErr = nil
	storage.saveErr = errors.New("save fail")
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected save copied file error")
	}
	storage.saveErr = nil
	files.createErr = errors.New("create copied fail")
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected create copied file record error")
	}
	files.createErr = nil
	files.updateStorageErr = errors.New("update storage fail")
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err == nil {
		t.Fatal("expected update storage used error")
	}
	files.updateStorageErr = nil
	if err := uc.CopyFile(ctx, "u1", "source", &folder.ID); err != nil {
		t.Fatalf("copy file: %v", err)
	}

	if err := uc.DeleteFile(ctx, "u1", "x"); err != nil {
		t.Fatalf("delete file: %v", err)
	}
}

func TestCloudUseCaseTrashViewsStarsSearchAndHelpers(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, shared, stars, storage, _ := newCloudUCForTest()

	root := seedFolder(folders, "root", "u1", "Root", nil)
	child := seedFolder(folders, "child", "u1", "Child", &root.ID)
	file1 := seedFile(files, "f1", "u1", "a.txt", "text/plain", "u1/a", &child.ID, 10)
	file2 := seedFile(files, "f2", "u1", "b.txt", "text/plain", "u1/b", nil, 20)
	files.trashed = []*domain.File{file1}
	folders.trashed = []*domain.Folder{child}
	storage.data["u1/a"] = []byte("a")
	storage.data["u1/b"] = []byte("b")

	if items, err := uc.ListTrash(ctx, "u1"); err != nil || len(items) != 2 {
		t.Fatalf("list trash failed: %v len=%d", err, len(items))
	}
	folders.listErr = errors.New("folder list fail")
	if _, err := uc.ListTrash(ctx, "u1"); err == nil {
		t.Fatal("expected list trashed folders error")
	}
	folders.listErr = nil
	files.listErr = errors.New("file list fail")
	if _, err := uc.ListTrash(ctx, "u1"); err == nil {
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
	folders.restoreErr = errors.New("restore folder fail")
	if err := uc.RestoreItem(ctx, "u1", "child", "folder"); err == nil {
		t.Fatal("expected restore folder error")
	}
	folders.restoreErr = nil

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
	if err := uc.EmptyTrash(ctx, "u1"); err != nil {
		t.Fatalf("empty trash: %v", err)
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
	if items, err := uc.ListStarred(ctx, "u1"); err != nil || len(items) != 2 {
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

	cache := map[string]string{}
	if p, err := uc.buildFolderPath(ctx, "u1", &child.ID, cache); err != nil || p == "" {
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
	if err := uc.MoveFolder(ctx, "u1", root.ID, strPtr("cycle-a")); err == nil {
		t.Fatal("expected move folder validation error")
	}
}

func strPtr(s string) *string { return &s }

func TestCloudUseCaseAdditionalBranchCoverage(t *testing.T) {
	ctx := context.Background()
	uc, folders, files, _, _, _, _ := newCloudUCForTest()
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

	ucNoQueue, folders2, _, _, _, storage2, _ := newCloudUCForTest()
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

	ucQueueErr, folders3, _, _, _, _, queue3 := newCloudUCForTest()
	root3 := seedFolder(folders3, "root3", "u1", "Root3", nil)
	queue3.err = errors.New("queue fail")
	if _, err := ucQueueErr.UploadFile(ctx, "u1", &root3.ID, "queued.txt", "text/plain", 1, []byte("z")); err != nil {
		t.Fatalf("expected upload success even when queue fails: %v", err)
	}
}

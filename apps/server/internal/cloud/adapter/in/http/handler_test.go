package http

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/cloud/domain"
	portin "lifebase/internal/cloud/port/in"
	"lifebase/internal/shared/middleware"
)

type mockCloudUC struct {
	folderRes *domain.Folder
	fileRes   *domain.File
	fileData  []byte

	listFolderItems   []portin.FolderItem
	listTrashItems    []portin.FolderItem
	listViewItems     []portin.FolderItem
	listStarsItems    []portin.StarItem
	searchItems       []*domain.File
	lastUploadMime    string
	lastTrashFolderID *string

	createFolderErr   error
	getFolderErr      error
	getTrashFolderErr error
	listFolderErr     error
	renameFolderErr   error
	moveFolderErr     error
	copyFolderErr     error
	deleteFolderErr   error

	uploadErr         error
	getFileErr        error
	downloadErr       error
	getFileContentErr error
	renameFileErr     error
	updateContentErr  error
	moveFileErr       error
	copyFileErr       error
	discardFileErr    error
	deleteFileErr     error

	restoreErr     error
	emptyTrashErr  error
	listRecentErr  error
	listSharedErr  error
	listStarredErr error

	starErr   error
	unstarErr error
	searchErr error
}

func (m *mockCloudUC) CreateFolder(context.Context, string, *string, string) (*domain.Folder, error) {
	return m.folderRes, m.createFolderErr
}
func (m *mockCloudUC) GetFolder(context.Context, string, string) (*domain.Folder, error) {
	return m.folderRes, m.getFolderErr
}
func (m *mockCloudUC) ListFolder(context.Context, string, *string, string, string) ([]portin.FolderItem, error) {
	return m.listFolderItems, m.listFolderErr
}
func (m *mockCloudUC) RenameFolder(context.Context, string, string, string) error {
	return m.renameFolderErr
}
func (m *mockCloudUC) MoveFolder(context.Context, string, string, *string) error {
	return m.moveFolderErr
}
func (m *mockCloudUC) CopyFolder(context.Context, string, string, *string) error {
	return m.copyFolderErr
}
func (m *mockCloudUC) DeleteFolder(context.Context, string, string) error { return m.deleteFolderErr }
func (m *mockCloudUC) GetTrashFolder(context.Context, string, string) (*domain.Folder, error) {
	return m.folderRes, m.getTrashFolderErr
}

func (m *mockCloudUC) UploadFile(_ context.Context, _ string, _ *string, _ string, mimeType string, _ int64, _ []byte) (*domain.File, error) {
	m.lastUploadMime = mimeType
	return m.fileRes, m.uploadErr
}
func (m *mockCloudUC) GetFile(context.Context, string, string) (*domain.File, error) {
	return m.fileRes, m.getFileErr
}
func (m *mockCloudUC) DownloadFile(context.Context, string, string) ([]byte, *domain.File, error) {
	return m.fileData, m.fileRes, m.downloadErr
}
func (m *mockCloudUC) GetFileContent(context.Context, string, string) (string, *domain.File, error) {
	return "hello", m.fileRes, m.getFileContentErr
}
func (m *mockCloudUC) RenameFile(context.Context, string, string, string) error {
	return m.renameFileErr
}
func (m *mockCloudUC) UpdateFileContent(context.Context, string, string, string) error {
	return m.updateContentErr
}
func (m *mockCloudUC) MoveFile(context.Context, string, string, *string) error { return m.moveFileErr }
func (m *mockCloudUC) CopyFile(context.Context, string, string, *string) (*domain.File, error) {
	return m.fileRes, m.copyFileErr
}
func (m *mockCloudUC) DiscardFile(context.Context, string, string) error { return m.discardFileErr }
func (m *mockCloudUC) DeleteFile(context.Context, string, string) error  { return m.deleteFileErr }

func (m *mockCloudUC) ListTrash(_ context.Context, _ string, folderID *string) ([]portin.FolderItem, error) {
	m.lastTrashFolderID = folderID
	return m.listTrashItems, m.listFolderErr
}
func (m *mockCloudUC) RestoreItem(context.Context, string, string, string) error { return m.restoreErr }
func (m *mockCloudUC) EmptyTrash(context.Context, string) error                  { return m.emptyTrashErr }

func (m *mockCloudUC) ListRecent(context.Context, string) ([]portin.FolderItem, error) {
	return m.listViewItems, m.listRecentErr
}
func (m *mockCloudUC) ListShared(context.Context, string) ([]portin.FolderItem, error) {
	return m.listViewItems, m.listSharedErr
}
func (m *mockCloudUC) ListStarred(context.Context, string) ([]portin.FolderItem, error) {
	return m.listViewItems, m.listStarredErr
}

func (m *mockCloudUC) ListStars(context.Context, string) ([]portin.StarItem, error) {
	return m.listStarsItems, m.listFolderErr
}
func (m *mockCloudUC) StarItem(context.Context, string, string, string) error   { return m.starErr }
func (m *mockCloudUC) UnstarItem(context.Context, string, string, string) error { return m.unstarErr }
func (m *mockCloudUC) Search(context.Context, string, string) ([]*domain.File, error) {
	return m.searchItems, m.searchErr
}

func cloudReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func withParam(req *http.Request, key, value string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func multipartUploadReq(t *testing.T, target, folderID, filename, contentType string, content []byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if folderID != "" {
		if err := w.WriteField("folder_id", folderID); err != nil {
			t.Fatalf("write folder id: %v", err)
		}
	}
	fw, err := w.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func multipartUploadReqWithoutPartContentType(t *testing.T, target, filename string, content []byte) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	fw, err := w.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("write file content: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, target, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func TestCloudHandlerFolders(t *testing.T) {
	now := time.Now()
	uc := &mockCloudUC{
		folderRes: &domain.Folder{ID: "f1", UserID: "user-1", Name: "Root", CreatedAt: now, UpdatedAt: now},
		listFolderItems: []portin.FolderItem{
			{Type: "folder", Folder: &domain.Folder{ID: "f1", Name: "Root"}},
		},
	}
	h := NewCloudHandler(uc)

	rec := httptest.NewRecorder()
	h.CreateFolder(rec, cloudReq(http.MethodPost, "/folders", `{"name":"Root"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateFolder(rec, cloudReq(http.MethodPost, "/folders", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.createFolderErr = errors.New("failed")
	rec = httptest.NewRecorder()
	h.CreateFolder(rec, cloudReq(http.MethodPost, "/folders", `{"name":"Root"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.createFolderErr = nil

	rec = httptest.NewRecorder()
	req := withParam(cloudReq(http.MethodGet, "/folders/f1", ""), "folderID", "f1")
	h.GetFolder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.getFolderErr = errors.New("missing")
	rec = httptest.NewRecorder()
	h.GetFolder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.getFolderErr = nil

	rec = httptest.NewRecorder()
	h.ListFolder(rec, cloudReq(http.MethodGet, "/folders?folder_id=f1", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listFolderErr = errors.New("db")
	rec = httptest.NewRecorder()
	h.ListFolder(rec, cloudReq(http.MethodGet, "/folders", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listFolderErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1", `{"name":"new"}`), "folderID", "f1")
	h.RenameFolder(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1", `{`), "folderID", "f1")
	h.RenameFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.renameFolderErr = errors.New("rename fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1", `{"name":"new"}`), "folderID", "f1")
	h.RenameFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.renameFolderErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1/move", `{"parent_id":"p1"}`), "folderID", "f1")
	h.MoveFolder(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1/move", `{`), "folderID", "f1")
	h.MoveFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.moveFolderErr = errors.New("move fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/folders/f1/move", `{"parent_id":"p1"}`), "folderID", "f1")
	h.MoveFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.moveFolderErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/folders/f1/copy", `{}`), "folderID", "f1")
	h.CopyFolder(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/folders/f1/copy", `{`), "folderID", "f1")
	h.CopyFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.copyFolderErr = errors.New("copy fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/folders/f1/copy", `{}`), "folderID", "f1")
	h.CopyFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.copyFolderErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodDelete, "/folders/f1", ""), "folderID", "f1")
	h.DeleteFolder(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.deleteFolderErr = errors.New("delete fail")
	rec = httptest.NewRecorder()
	h.DeleteFolder(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCloudHandlerFiles(t *testing.T) {
	now := time.Now()
	uc := &mockCloudUC{
		fileRes:  &domain.File{ID: "file1", Name: "a.txt", MimeType: "text/plain", CreatedAt: now, UpdatedAt: now},
		fileData: []byte("hello"),
	}
	h := NewCloudHandler(uc)

	rec := httptest.NewRecorder()
	h.UploadFile(rec, cloudReq(http.MethodPost, "/files", "not-multipart"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/files", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user-1"))
	rec = httptest.NewRecorder()
	h.UploadFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing file, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.UploadFile(rec, multipartUploadReq(t, "/files", "folder1", "a.txt", "", []byte("hello")))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	uc.uploadErr = errors.New("upload fail")
	rec = httptest.NewRecorder()
	h.UploadFile(rec, multipartUploadReq(t, "/files", "", "a.txt", "", []byte("hello")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.uploadErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodGet, "/files/file1/download", ""), "fileID", "file1")
	h.DownloadFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.Contains(got, "attachment") {
		t.Fatalf("expected attachment disposition, got %q", got)
	}
	uc.downloadErr = errors.New("missing")
	rec = httptest.NewRecorder()
	h.DownloadFile(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.downloadErr = nil

	prevFormat := formatDownloadDisposition
	formatDownloadDisposition = func(string, map[string]string) string { return "" }
	t.Cleanup(func() { formatDownloadDisposition = prevFormat })
	rec = httptest.NewRecorder()
	h.DownloadFile(rec, req)
	if got := rec.Header().Get("Content-Disposition"); got != "attachment" {
		t.Fatalf("expected fallback attachment disposition, got %q", got)
	}
	formatDownloadDisposition = prevFormat

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodGet, "/files/file1/content", ""), "fileID", "file1")
	h.GetFileContent(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.getFileContentErr = errors.New("not editable")
	rec = httptest.NewRecorder()
	h.GetFileContent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.getFileContentErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodGet, "/files/file1", ""), "fileID", "file1")
	h.GetFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.getFileErr = errors.New("missing")
	rec = httptest.NewRecorder()
	h.GetFile(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.getFileErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1", `{"name":"b.txt"}`), "fileID", "file1")
	h.RenameFile(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1", `{`), "fileID", "file1")
	h.RenameFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.renameFileErr = errors.New("rename fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1", `{"name":"b.txt"}`), "fileID", "file1")
	h.RenameFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.renameFileErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/content", `{"content":"hello"}`), "fileID", "file1")
	h.UpdateFileContent(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/content", `{`), "fileID", "file1")
	h.UpdateFileContent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateContentErr = errors.New("update fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/content", `{"content":"hello"}`), "fileID", "file1")
	h.UpdateFileContent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateContentErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/move", `{"folder_id":"f1"}`), "fileID", "file1")
	h.MoveFile(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/move", `{`), "fileID", "file1")
	h.MoveFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.moveFileErr = errors.New("move fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPatch, "/files/file1/move", `{"folder_id":"f1"}`), "fileID", "file1")
	h.MoveFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.moveFileErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/files/file1/copy", `{}`), "fileID", "file1")
	h.CopyFile(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"id":"file1"`) {
		t.Fatalf("expected copied file payload, got %s", rec.Body.String())
	}
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/files/file1/copy", `{`), "fileID", "file1")
	h.CopyFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.copyFileErr = errors.New("copy fail")
	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodPost, "/files/file1/copy", `{}`), "fileID", "file1")
	h.CopyFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.copyFileErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodDelete, "/files/file1/discard", ""), "fileID", "file1")
	h.DiscardFile(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.discardFileErr = errors.New("discard fail")
	rec = httptest.NewRecorder()
	h.DiscardFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.discardFileErr = nil

	rec = httptest.NewRecorder()
	req = withParam(cloudReq(http.MethodDelete, "/files/file1", ""), "fileID", "file1")
	h.DeleteFile(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.deleteFileErr = errors.New("delete fail")
	rec = httptest.NewRecorder()
	h.DeleteFile(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCloudHandlerUploadFileReadError(t *testing.T) {
	prev := readAllUploadFile
	t.Cleanup(func() { readAllUploadFile = prev })
	readAllUploadFile = func(io.Reader) ([]byte, error) {
		return nil, errors.New("read fail")
	}

	h := NewCloudHandler(&mockCloudUC{})
	rec := httptest.NewRecorder()
	h.UploadFile(rec, multipartUploadReq(t, "/files", "", "a.txt", "", []byte("hello")))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 for read failure, got %d", rec.Code)
	}
}

func TestCloudHandlerUploadFileDefaultMime(t *testing.T) {
	now := time.Now()
	uc := &mockCloudUC{
		fileRes:  &domain.File{ID: "file1", Name: "a.txt", MimeType: "text/plain", CreatedAt: now, UpdatedAt: now},
		fileData: []byte("hello"),
	}
	h := NewCloudHandler(uc)

	rec := httptest.NewRecorder()
	h.UploadFile(rec, multipartUploadReqWithoutPartContentType(t, "/files", "a.txt", []byte("hello")))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	if uc.lastUploadMime != "text/plain; charset=utf-8" {
		t.Fatalf("expected detected mime type, got %q", uc.lastUploadMime)
	}
}

func TestCloudHandlerTrashViewsStarsAndSearch(t *testing.T) {
	uc := &mockCloudUC{
		listTrashItems: []portin.FolderItem{{Type: "file", File: &domain.File{ID: "f1"}}},
		listViewItems:  []portin.FolderItem{{Type: "folder", Folder: &domain.Folder{ID: "d1"}}},
		listStarsItems: []portin.StarItem{{ID: "f1", Type: "file"}},
		searchItems:    []*domain.File{{ID: "f1", Name: "match.txt"}},
	}
	h := NewCloudHandler(uc)

	rec := httptest.NewRecorder()
	h.ListTrash(rec, cloudReq(http.MethodGet, "/trash", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if uc.lastTrashFolderID != nil {
		t.Fatalf("expected nil trash folder id, got %v", *uc.lastTrashFolderID)
	}
	rec = httptest.NewRecorder()
	h.ListTrash(rec, cloudReq(http.MethodGet, "/trash?folder_id=folder-1", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for scoped trash list, got %d", rec.Code)
	}
	if uc.lastTrashFolderID == nil || *uc.lastTrashFolderID != "folder-1" {
		t.Fatalf("expected folder-1 trash folder id, got %#v", uc.lastTrashFolderID)
	}
	uc.listFolderErr = errors.New("list fail")
	rec = httptest.NewRecorder()
	h.ListTrash(rec, cloudReq(http.MethodGet, "/trash", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listFolderErr = nil

	rec = httptest.NewRecorder()
	req := withParam(cloudReq(http.MethodGet, "/trash/folders/f1", ""), "folderID", "f1")
	h.GetTrashFolder(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.getTrashFolderErr = errors.New("not found")
	rec = httptest.NewRecorder()
	h.GetTrashFolder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.getTrashFolderErr = nil

	rec = httptest.NewRecorder()
	h.RestoreItem(rec, cloudReq(http.MethodPost, "/trash/restore", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.RestoreItem(rec, cloudReq(http.MethodPost, "/trash/restore", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.restoreErr = errors.New("restore fail")
	rec = httptest.NewRecorder()
	h.RestoreItem(rec, cloudReq(http.MethodPost, "/trash/restore", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.restoreErr = nil

	rec = httptest.NewRecorder()
	h.EmptyTrash(rec, cloudReq(http.MethodDelete, "/trash", ""))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.emptyTrashErr = errors.New("empty fail")
	rec = httptest.NewRecorder()
	h.EmptyTrash(rec, cloudReq(http.MethodDelete, "/trash", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.emptyTrashErr = nil

	rec = httptest.NewRecorder()
	h.ListRecent(rec, cloudReq(http.MethodGet, "/recent", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listRecentErr = errors.New("recent fail")
	rec = httptest.NewRecorder()
	h.ListRecent(rec, cloudReq(http.MethodGet, "/recent", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listRecentErr = nil

	rec = httptest.NewRecorder()
	h.ListShared(rec, cloudReq(http.MethodGet, "/shared", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listSharedErr = errors.New("shared fail")
	rec = httptest.NewRecorder()
	h.ListShared(rec, cloudReq(http.MethodGet, "/shared", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listSharedErr = nil

	rec = httptest.NewRecorder()
	h.ListStarred(rec, cloudReq(http.MethodGet, "/starred", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listStarredErr = errors.New("starred fail")
	rec = httptest.NewRecorder()
	h.ListStarred(rec, cloudReq(http.MethodGet, "/starred", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listStarredErr = nil

	rec = httptest.NewRecorder()
	h.ListStars(rec, cloudReq(http.MethodGet, "/stars", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listFolderErr = errors.New("stars fail")
	rec = httptest.NewRecorder()
	h.ListStars(rec, cloudReq(http.MethodGet, "/stars", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listFolderErr = nil

	rec = httptest.NewRecorder()
	h.StarItem(rec, cloudReq(http.MethodPost, "/stars", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.StarItem(rec, cloudReq(http.MethodPost, "/stars", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.starErr = errors.New("star fail")
	rec = httptest.NewRecorder()
	h.StarItem(rec, cloudReq(http.MethodPost, "/stars", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.starErr = nil

	rec = httptest.NewRecorder()
	h.UnstarItem(rec, cloudReq(http.MethodDelete, "/stars", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.UnstarItem(rec, cloudReq(http.MethodDelete, "/stars", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.unstarErr = errors.New("unstar fail")
	rec = httptest.NewRecorder()
	h.UnstarItem(rec, cloudReq(http.MethodDelete, "/stars", `{"id":"f1","type":"file"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.SearchFiles(rec, cloudReq(http.MethodGet, "/search", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.SearchFiles(rec, cloudReq(http.MethodGet, "/search?q=abc", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.searchErr = errors.New("search fail")
	rec = httptest.NewRecorder()
	h.SearchFiles(rec, cloudReq(http.MethodGet, "/search?q=abc", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

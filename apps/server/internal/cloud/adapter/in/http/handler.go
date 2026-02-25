package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/cloud/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type CloudHandler struct {
	cloud portin.CloudUseCase
}

func NewCloudHandler(cloud portin.CloudUseCase) *CloudHandler {
	return &CloudHandler{cloud: cloud}
}

// Folders

func (h *CloudHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		ParentID *string `json:"parent_id"`
		Name     string  `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	folder, err := h.cloud.CreateFolder(r.Context(), userID, req.ParentID, req.Name)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, folder)
}

func (h *CloudHandler) GetFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := chi.URLParam(r, "folderID")

	folder, err := h.cloud.GetFolder(r.Context(), userID, folderID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "folder not found")
		return
	}
	response.JSON(w, http.StatusOK, folder)
}

func (h *CloudHandler) ListFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := r.URL.Query().Get("folder_id")
	sortBy := r.URL.Query().Get("sort_by")
	sortDir := r.URL.Query().Get("sort_dir")

	if sortBy == "" {
		sortBy = "name"
	}
	if sortDir == "" {
		sortDir = "asc"
	}

	var folderIDPtr *string
	if folderID != "" {
		folderIDPtr = &folderID
	}

	items, err := h.cloud.ListFolder(r.Context(), userID, folderIDPtr, sortBy, sortDir)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CloudHandler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := chi.URLParam(r, "folderID")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	if err := h.cloud.RenameFolder(r.Context(), userID, folderID, req.Name); err != nil {
		response.Error(w, http.StatusBadRequest, "RENAME_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CloudHandler) MoveFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := chi.URLParam(r, "folderID")
	var req struct {
		ParentID *string `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.cloud.MoveFolder(r.Context(), userID, folderID, req.ParentID); err != nil {
		response.Error(w, http.StatusBadRequest, "MOVE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CloudHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := chi.URLParam(r, "folderID")

	if err := h.cloud.DeleteFolder(r.Context(), userID, folderID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

// Files

func (h *CloudHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart form")
		return
	}

	f, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "MISSING_FILE", "file is required")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "READ_FAILED", "failed to read file")
		return
	}

	folderID := r.FormValue("folder_id")
	var folderIDPtr *string
	if folderID != "" {
		folderIDPtr = &folderID
	}

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	file, err := h.cloud.UploadFile(r.Context(), userID, folderIDPtr, header.Filename, mimeType, int64(len(data)), data)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UPLOAD_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, file)
}

func (h *CloudHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")

	data, file, err := h.cloud.DownloadFile(r.Context(), userID, fileID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "file not found")
		return
	}

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename=\""+file.Name+"\"")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *CloudHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")

	file, err := h.cloud.GetFile(r.Context(), userID, fileID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "file not found")
		return
	}
	response.JSON(w, http.StatusOK, file)
}

func (h *CloudHandler) RenameFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	if err := h.cloud.RenameFile(r.Context(), userID, fileID, req.Name); err != nil {
		response.Error(w, http.StatusBadRequest, "RENAME_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CloudHandler) MoveFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")
	var req struct {
		FolderID *string `json:"folder_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.cloud.MoveFile(r.Context(), userID, fileID, req.FolderID); err != nil {
		response.Error(w, http.StatusBadRequest, "MOVE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CloudHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")

	if err := h.cloud.DeleteFile(r.Context(), userID, fileID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

// Trash

func (h *CloudHandler) ListTrash(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	items, err := h.cloud.ListTrash(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *CloudHandler) RestoreItem(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" || req.Type == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "id and type are required")
		return
	}

	if err := h.cloud.RestoreItem(r.Context(), userID, req.ID, req.Type); err != nil {
		response.Error(w, http.StatusBadRequest, "RESTORE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CloudHandler) EmptyTrash(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	if err := h.cloud.EmptyTrash(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "EMPTY_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

// Search

func (h *CloudHandler) SearchFiles(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_QUERY", "q parameter is required")
		return
	}

	files, err := h.cloud.Search(r.Context(), userID, query)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "SEARCH_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"files": files})
}

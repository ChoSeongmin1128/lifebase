package http

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/gallery/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type GalleryHandler struct {
	gallery   portin.GalleryUseCase
	thumbPath string
}

func NewGalleryHandler(gallery portin.GalleryUseCase, thumbPath string) *GalleryHandler {
	return &GalleryHandler{gallery: gallery, thumbPath: thumbPath}
}

func (h *GalleryHandler) ListMedia(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	mediaType := r.URL.Query().Get("type")
	sortBy := r.URL.Query().Get("sort_by")
	sortDir := r.URL.Query().Get("sort_dir")
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	if sortBy == "" {
		sortBy = "taken_at"
	}
	if sortDir == "" {
		sortDir = "desc"
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	files, nextCursor, err := h.gallery.ListMedia(r.Context(), userID, mediaType, sortBy, sortDir, cursor, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}

	result := map[string]any{
		"items": files,
	}
	if nextCursor != "" {
		result["next_cursor"] = nextCursor
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *GalleryHandler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	fileID := chi.URLParam(r, "fileID")
	size := chi.URLParam(r, "size")

	if size != "small" && size != "medium" {
		response.Error(w, http.StatusBadRequest, "INVALID_SIZE", "size must be small or medium")
		return
	}
	media, err := h.gallery.GetMedia(r.Context(), userID, fileID)
	if err != nil || media == nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "thumbnail not found")
		return
	}

	thumbFile := filepath.Join(h.thumbPath, userID, fileID+"_"+size+".webp")
	data, err := os.ReadFile(thumbFile)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "thumbnail not found")
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

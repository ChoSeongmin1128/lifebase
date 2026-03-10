package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
	portin "lifebase/internal/sharing/port/in"
	sharingusecase "lifebase/internal/sharing/usecase"
)

type SharingHandler struct {
	sharing portin.SharingUseCase
}

func NewSharingHandler(sharing portin.SharingUseCase) *SharingHandler {
	return &SharingHandler{sharing: sharing}
}

func (h *SharingHandler) CreateInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		FolderID string `json:"folder_id"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FolderID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "folder_id is required")
		return
	}
	if req.Role == "" {
		req.Role = "viewer"
	}

	link, err := h.sharing.CreateInvite(r.Context(), userID, req.FolderID, req.Role)
	if err != nil {
		if errors.Is(err, sharingusecase.ErrShareAccessDenied) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "share access denied")
			return
		}
		response.Error(w, http.StatusBadRequest, "INVITE_FAILED", "failed to create invite")
		return
	}
	response.JSON(w, http.StatusCreated, link)
}

func (h *SharingHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "token is required")
		return
	}

	share, err := h.sharing.AcceptInvite(r.Context(), userID, req.Token)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "ACCEPT_FAILED", "failed to accept invite")
		return
	}
	response.JSON(w, http.StatusOK, share)
}

func (h *SharingHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	folderID := r.URL.Query().Get("folder_id")
	if folderID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "folder_id is required")
		return
	}

	shares, err := h.sharing.ListShares(r.Context(), userID, folderID)
	if err != nil {
		if errors.Is(err, sharingusecase.ErrShareAccessDenied) {
			response.Error(w, http.StatusForbidden, "FORBIDDEN", "share access denied")
			return
		}
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list shares")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (h *SharingHandler) ListSharedWithMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	shares, err := h.sharing.ListSharedWithMe(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list shares")
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"shares": shares})
}

func (h *SharingHandler) RemoveShare(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	shareID := chi.URLParam(r, "shareID")

	if err := h.sharing.RemoveShare(r.Context(), userID, shareID); err != nil {
		response.Error(w, http.StatusBadRequest, "REMOVE_FAILED", "failed to remove share")
		return
	}
	response.NoContent(w)
}

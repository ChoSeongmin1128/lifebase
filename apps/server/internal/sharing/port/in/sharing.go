package in

import (
	"context"

	"lifebase/internal/sharing/domain"
)

type InviteLink struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type SharingUseCase interface {
	CreateInvite(ctx context.Context, ownerID, folderID, role string) (*InviteLink, error)
	AcceptInvite(ctx context.Context, userID, token string) (*domain.Share, error)
	ListShares(ctx context.Context, userID, folderID string) ([]*domain.Share, error)
	ListSharedWithMe(ctx context.Context, userID string) ([]*domain.Share, error)
	RemoveShare(ctx context.Context, ownerID, shareID string) error
}

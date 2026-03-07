package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"

	"lifebase/internal/sharing/domain"
	portin "lifebase/internal/sharing/port/in"
	portout "lifebase/internal/sharing/port/out"
)

var shareRandRead = rand.Read

type sharingUseCase struct {
	shares  portout.ShareRepository
	invites portout.ShareInviteRepository
}

func NewSharingUseCase(shares portout.ShareRepository, invites portout.ShareInviteRepository) portin.SharingUseCase {
	return &sharingUseCase{shares: shares, invites: invites}
}

func (uc *sharingUseCase) CreateInvite(ctx context.Context, ownerID, folderID, role string) (*portin.InviteLink, error) {
	if role != "viewer" && role != "editor" {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	token, err := generateToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	now := time.Now()
	invite := &domain.ShareInvite{
		ID:        uuid.New().String(),
		FolderID:  folderID,
		OwnerID:   ownerID,
		Token:     token,
		Role:      role,
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	}

	if err := uc.invites.Create(ctx, invite); err != nil {
		return nil, fmt.Errorf("create invite: %w", err)
	}

	return &portin.InviteLink{
		Token:     token,
		ExpiresAt: invite.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func (uc *sharingUseCase) AcceptInvite(ctx context.Context, userID, token string) (*domain.Share, error) {
	invite, err := uc.invites.FindByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("invalid invite token")
	}

	if invite.AcceptedAt != nil {
		return nil, fmt.Errorf("invite already accepted")
	}

	if time.Now().After(invite.ExpiresAt) {
		return nil, fmt.Errorf("invite expired")
	}

	if invite.OwnerID == userID {
		return nil, fmt.Errorf("cannot share with yourself")
	}

	now := time.Now()
	share := &domain.Share{
		ID:         uuid.New().String(),
		FolderID:   invite.FolderID,
		OwnerID:    invite.OwnerID,
		SharedWith: userID,
		Role:       invite.Role,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := uc.shares.Create(ctx, share); err != nil {
		return nil, fmt.Errorf("create share: %w", err)
	}

	if err := uc.invites.MarkAccepted(ctx, invite.ID); err != nil {
		return nil, fmt.Errorf("mark accepted: %w", err)
	}

	return share, nil
}

func (uc *sharingUseCase) ListShares(ctx context.Context, userID, folderID string) ([]*domain.Share, error) {
	return uc.shares.ListByFolder(ctx, folderID)
}

func (uc *sharingUseCase) ListSharedWithMe(ctx context.Context, userID string) ([]*domain.Share, error) {
	return uc.shares.ListByUser(ctx, userID)
}

func (uc *sharingUseCase) RemoveShare(ctx context.Context, ownerID, shareID string) error {
	share, err := uc.shares.FindByID(ctx, shareID)
	if err != nil {
		return fmt.Errorf("share not found")
	}

	if share.OwnerID != ownerID {
		return fmt.Errorf("not authorized to remove this share")
	}

	return uc.shares.Delete(ctx, shareID)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := shareRandRead(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

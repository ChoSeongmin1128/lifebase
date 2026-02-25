package out

import (
	"context"

	"lifebase/internal/sharing/domain"
)

type ShareRepository interface {
	Create(ctx context.Context, share *domain.Share) error
	FindByID(ctx context.Context, id string) (*domain.Share, error)
	ListByFolder(ctx context.Context, folderID string) ([]*domain.Share, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Share, error)
	Delete(ctx context.Context, id string) error
}

type ShareInviteRepository interface {
	Create(ctx context.Context, invite *domain.ShareInvite) error
	FindByToken(ctx context.Context, token string) (*domain.ShareInvite, error)
	MarkAccepted(ctx context.Context, id string) error
}

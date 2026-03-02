package out

import (
	"context"
	"time"

	admindomain "lifebase/internal/admin/domain"
)

type AdminChecker interface {
	IsActiveAdmin(ctx context.Context, userID string) (bool, error)
}

type AdminUserRepository interface {
	AdminChecker
	FindByUserID(ctx context.Context, userID string) (*admindomain.AdminUser, error)
	FindByID(ctx context.Context, adminID string) (*admindomain.AdminUser, error)
	List(ctx context.Context) ([]*admindomain.AdminUser, error)
	Create(ctx context.Context, admin *admindomain.AdminUser) error
	Update(ctx context.Context, admin *admindomain.AdminUser) error
	CountActiveSuperAdmins(ctx context.Context) (int, error)
}

type FileRef struct {
	ID          string
	StoragePath string
}

type StorageResetRepository interface {
	ListFilesByUser(ctx context.Context, userID string) ([]FileRef, error)
	DeleteAllFilesByUser(ctx context.Context, userID string) error
	DeleteAllFoldersByUser(ctx context.Context, userID string) error
	DeleteAllStarsByUser(ctx context.Context, userID string) error
	DeleteSharesByOwner(ctx context.Context, ownerID string) error
	SumStorageUsed(ctx context.Context, userID string) (int64, error)
}

type GoogleAccountAdminRepository interface {
	ListByUserID(ctx context.Context, userID string) ([]GoogleAccountRecord, error)
	UpdateStatus(ctx context.Context, accountID, userID, status string) error
}

type GoogleAccountRecord struct {
	ID          string
	UserID      string
	GoogleEmail string
	GoogleID    string
	Status      string
	IsPrimary   bool
	ConnectedAt time.Time
}

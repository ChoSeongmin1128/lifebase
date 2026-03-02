package in

import (
	"context"
	"time"

	admindomain "lifebase/internal/admin/domain"
	authdomain "lifebase/internal/auth/domain"
)

type UserDetail struct {
	User           *authdomain.User
	GoogleAccounts []GoogleAccountSummary
}

type GoogleAccountSummary struct {
	ID          string
	GoogleEmail string
	GoogleID    string
	Status      string
	IsPrimary   bool
	ConnectedAt time.Time
}

type AdminUserView struct {
	ID        string
	UserID    string
	Email     string
	Name      string
	Role      admindomain.Role
	IsActive  bool
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AdminUseCase interface {
	ListUsers(ctx context.Context, actorUserID, search, cursor string, limit int) ([]*authdomain.User, string, error)
	GetUserDetail(ctx context.Context, actorUserID, targetUserID string) (*UserDetail, error)
	UpdateStorageQuota(ctx context.Context, actorUserID, targetUserID string, quotaBytes int64) error
	RecalculateStorageUsed(ctx context.Context, actorUserID, targetUserID string) (int64, error)
	ResetUserStorage(ctx context.Context, actorUserID, targetUserID, confirm string) error
	UpdateGoogleAccountStatus(ctx context.Context, actorUserID, targetUserID, accountID, status string) error

	ListAdmins(ctx context.Context, actorUserID string) ([]*AdminUserView, error)
	CreateAdmin(ctx context.Context, actorUserID, email string, role admindomain.Role) (*AdminUserView, error)
	UpdateAdminRole(ctx context.Context, actorUserID, adminID string, role admindomain.Role) error
	DeactivateAdmin(ctx context.Context, actorUserID, adminID string) error
}

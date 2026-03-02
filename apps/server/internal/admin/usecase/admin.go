package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	admindomain "lifebase/internal/admin/domain"
	portin "lifebase/internal/admin/port/in"
	portout "lifebase/internal/admin/port/out"
	authdomain "lifebase/internal/auth/domain"
	authout "lifebase/internal/auth/port/out"
)

const storageResetConfirmPrefix = "DELETE "

type adminUseCase struct {
	admins      portout.AdminUserRepository
	users       authout.UserRepository
	googleAccts portout.GoogleAccountAdminRepository
	storage     portout.StorageResetRepository
	dataPath    string
	thumbPath   string
}

func NewAdminUseCase(
	admins portout.AdminUserRepository,
	users authout.UserRepository,
	googleAccts portout.GoogleAccountAdminRepository,
	storage portout.StorageResetRepository,
	dataPath, thumbPath string,
) portin.AdminUseCase {
	return &adminUseCase{
		admins:      admins,
		users:       users,
		googleAccts: googleAccts,
		storage:     storage,
		dataPath:    dataPath,
		thumbPath:   thumbPath,
	}
}

func (uc *adminUseCase) ListUsers(ctx context.Context, actorUserID, search, cursor string, limit int) ([]*authdomain.User, string, error) {
	if _, err := uc.requireAdmin(ctx, actorUserID); err != nil {
		return nil, "", err
	}
	return uc.users.ListUsers(ctx, search, cursor, limit)
}

func (uc *adminUseCase) GetUserDetail(ctx context.Context, actorUserID, targetUserID string) (*portin.UserDetail, error) {
	if _, err := uc.requireAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}
	user, err := uc.users.FindByID(ctx, targetUserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	accounts, err := uc.googleAccts.ListByUserID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	summaries := make([]portin.GoogleAccountSummary, 0, len(accounts))
	for _, account := range accounts {
		summaries = append(summaries, portin.GoogleAccountSummary{
			ID:          account.ID,
			GoogleEmail: account.GoogleEmail,
			GoogleID:    account.GoogleID,
			Status:      account.Status,
			IsPrimary:   account.IsPrimary,
			ConnectedAt: account.ConnectedAt,
		})
	}
	return &portin.UserDetail{
		User:           user,
		GoogleAccounts: summaries,
	}, nil
}

func (uc *adminUseCase) UpdateStorageQuota(ctx context.Context, actorUserID, targetUserID string, quotaBytes int64) error {
	if _, err := uc.requireAdmin(ctx, actorUserID); err != nil {
		return err
	}
	if quotaBytes <= 0 {
		return fmt.Errorf("quota must be positive")
	}
	user, err := uc.users.FindByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if quotaBytes < user.StorageUsedBytes {
		return fmt.Errorf("quota cannot be less than used storage")
	}
	return uc.users.UpdateStorageQuota(ctx, targetUserID, quotaBytes)
}

func (uc *adminUseCase) RecalculateStorageUsed(ctx context.Context, actorUserID, targetUserID string) (int64, error) {
	if _, err := uc.requireAdmin(ctx, actorUserID); err != nil {
		return 0, err
	}
	if _, err := uc.users.FindByID(ctx, targetUserID); err != nil {
		return 0, fmt.Errorf("user not found")
	}

	used, err := uc.storage.SumStorageUsed(ctx, targetUserID)
	if err != nil {
		return 0, err
	}
	if err := uc.users.UpdateStorageUsed(ctx, targetUserID, used); err != nil {
		return 0, err
	}
	return used, nil
}

func (uc *adminUseCase) ResetUserStorage(ctx context.Context, actorUserID, targetUserID, confirm string) error {
	if _, err := uc.requireSuperAdmin(ctx, actorUserID); err != nil {
		return err
	}
	user, err := uc.users.FindByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	expectedConfirm := storageResetConfirmPrefix + user.Email
	if confirm != expectedConfirm {
		return fmt.Errorf("confirmation text mismatch")
	}

	files, err := uc.storage.ListFilesByUser(ctx, targetUserID)
	if err != nil {
		return err
	}
	for _, file := range files {
		fullPath := filepath.Join(uc.dataPath, file.StoragePath)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove storage file", "file_id", file.ID, "path", fullPath, "error", err)
		}
	}
	if err := os.RemoveAll(filepath.Join(uc.dataPath, targetUserID)); err != nil {
		slog.Warn("failed to remove user data dir", "user_id", targetUserID, "error", err)
	}
	if err := os.RemoveAll(filepath.Join(uc.thumbPath, targetUserID)); err != nil {
		slog.Warn("failed to remove user thumb dir", "user_id", targetUserID, "error", err)
	}

	if err := uc.storage.DeleteAllStarsByUser(ctx, targetUserID); err != nil {
		return err
	}
	if err := uc.storage.DeleteSharesByOwner(ctx, targetUserID); err != nil {
		return err
	}
	if err := uc.storage.DeleteAllFilesByUser(ctx, targetUserID); err != nil {
		return err
	}
	if err := uc.storage.DeleteAllFoldersByUser(ctx, targetUserID); err != nil {
		return err
	}
	return uc.users.UpdateStorageUsed(ctx, targetUserID, 0)
}

func (uc *adminUseCase) UpdateGoogleAccountStatus(ctx context.Context, actorUserID, targetUserID, accountID, status string) error {
	if _, err := uc.requireAdmin(ctx, actorUserID); err != nil {
		return err
	}
	if status != "active" && status != "reauth_required" && status != "revoked" {
		return fmt.Errorf("invalid status")
	}
	if _, err := uc.users.FindByID(ctx, targetUserID); err != nil {
		return fmt.Errorf("user not found")
	}
	if err := uc.googleAccts.UpdateStatus(ctx, accountID, targetUserID, status); err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("google account not found")
		}
		return err
	}
	return nil
}

func (uc *adminUseCase) ListAdmins(ctx context.Context, actorUserID string) ([]*portin.AdminUserView, error) {
	if _, err := uc.requireSuperAdmin(ctx, actorUserID); err != nil {
		return nil, err
	}
	admins, err := uc.admins.List(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*portin.AdminUserView, 0, len(admins))
	for _, admin := range admins {
		user, err := uc.users.FindByID(ctx, admin.UserID)
		if err != nil {
			continue
		}
		out = append(out, &portin.AdminUserView{
			ID:        admin.ID,
			UserID:    admin.UserID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      admin.Role,
			IsActive:  admin.IsActive,
			CreatedBy: admin.CreatedBy,
			CreatedAt: admin.CreatedAt,
			UpdatedAt: admin.UpdatedAt,
		})
	}
	return out, nil
}

func (uc *adminUseCase) CreateAdmin(ctx context.Context, actorUserID, email string, role admindomain.Role) (*portin.AdminUserView, error) {
	if role != admindomain.RoleAdmin && role != admindomain.RoleSuperAdmin {
		return nil, fmt.Errorf("invalid role")
	}
	targetUser, err := uc.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("target user not found")
	}

	activeSuperAdmins, err := uc.admins.CountActiveSuperAdmins(ctx)
	if err != nil {
		return nil, err
	}
	createdBy := actorUserID
	if activeSuperAdmins == 0 {
		// Bootstrap rule: first admin row allows self-created super_admin only.
		if targetUser.ID != actorUserID || role != admindomain.RoleSuperAdmin {
			return nil, fmt.Errorf("first admin bootstrap must be self super_admin")
		}
		createdBy = targetUser.ID
	} else {
		if _, err := uc.requireSuperAdmin(ctx, actorUserID); err != nil {
			return nil, err
		}
	}

	existing, err := uc.admins.FindByUserID(ctx, targetUser.ID)
	if err == nil && existing != nil {
		existing.Role = role
		existing.IsActive = true
		existing.UpdatedAt = time.Now()
		if err := uc.admins.Update(ctx, existing); err != nil {
			return nil, err
		}
		return &portin.AdminUserView{
			ID:        existing.ID,
			UserID:    targetUser.ID,
			Email:     targetUser.Email,
			Name:      targetUser.Name,
			Role:      existing.Role,
			IsActive:  existing.IsActive,
			CreatedBy: existing.CreatedBy,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: existing.UpdatedAt,
		}, nil
	} else if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}

	now := time.Now()
	admin := &admindomain.AdminUser{
		ID:        uuid.New().String(),
		UserID:    targetUser.ID,
		Role:      role,
		IsActive:  true,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.admins.Create(ctx, admin); err != nil {
		return nil, err
	}
	return &portin.AdminUserView{
		ID:        admin.ID,
		UserID:    targetUser.ID,
		Email:     targetUser.Email,
		Name:      targetUser.Name,
		Role:      admin.Role,
		IsActive:  admin.IsActive,
		CreatedBy: admin.CreatedBy,
		CreatedAt: admin.CreatedAt,
		UpdatedAt: admin.UpdatedAt,
	}, nil
}

func (uc *adminUseCase) UpdateAdminRole(ctx context.Context, actorUserID, adminID string, role admindomain.Role) error {
	if role != admindomain.RoleAdmin && role != admindomain.RoleSuperAdmin {
		return fmt.Errorf("invalid role")
	}
	if _, err := uc.requireSuperAdmin(ctx, actorUserID); err != nil {
		return err
	}

	target, err := uc.admins.FindByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("admin not found")
	}
	if target.UserID == actorUserID {
		return fmt.Errorf("cannot change own role")
	}
	if target.Role == admindomain.RoleSuperAdmin && role != admindomain.RoleSuperAdmin && target.IsActive {
		count, err := uc.admins.CountActiveSuperAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("cannot demote last active super admin")
		}
	}
	target.Role = role
	target.UpdatedAt = time.Now()
	return uc.admins.Update(ctx, target)
}

func (uc *adminUseCase) DeactivateAdmin(ctx context.Context, actorUserID, adminID string) error {
	if _, err := uc.requireSuperAdmin(ctx, actorUserID); err != nil {
		return err
	}
	target, err := uc.admins.FindByID(ctx, adminID)
	if err != nil {
		return fmt.Errorf("admin not found")
	}
	if target.UserID == actorUserID {
		return fmt.Errorf("cannot deactivate yourself")
	}
	if target.Role == admindomain.RoleSuperAdmin && target.IsActive {
		count, err := uc.admins.CountActiveSuperAdmins(ctx)
		if err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("cannot deactivate last active super admin")
		}
	}
	target.IsActive = false
	target.UpdatedAt = time.Now()
	return uc.admins.Update(ctx, target)
}

func (uc *adminUseCase) requireAdmin(ctx context.Context, actorUserID string) (*admindomain.AdminUser, error) {
	admin, err := uc.admins.FindByUserID(ctx, actorUserID)
	if err != nil || admin == nil || !admin.IsActive {
		return nil, fmt.Errorf("admin access denied")
	}
	return admin, nil
}

func (uc *adminUseCase) requireSuperAdmin(ctx context.Context, actorUserID string) (*admindomain.AdminUser, error) {
	admin, err := uc.requireAdmin(ctx, actorUserID)
	if err != nil {
		return nil, err
	}
	if !admin.IsSuperAdmin() {
		return nil, fmt.Errorf("super admin required")
	}
	return admin, nil
}

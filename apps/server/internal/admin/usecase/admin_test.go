package usecase

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	admindomain "lifebase/internal/admin/domain"
	portin "lifebase/internal/admin/port/in"
	portout "lifebase/internal/admin/port/out"
	authdomain "lifebase/internal/auth/domain"
)

type adminRepoStub struct {
	byID             map[string]*admindomain.AdminUser
	byUser           map[string]*admindomain.AdminUser
	list             []*admindomain.AdminUser
	findByUserFn     func(string) (*admindomain.AdminUser, error)
	findByUserErr    error
	findByIDErr      error
	listErr          error
	createErr        error
	updateErr        error
	countSuper       int
	countSuperErr    error
}

func newAdminRepoStub() *adminRepoStub {
	return &adminRepoStub{
		byID:   map[string]*admindomain.AdminUser{},
		byUser: map[string]*admindomain.AdminUser{},
	}
}

func (m *adminRepoStub) IsActiveAdmin(context.Context, string) (bool, error) { return true, nil }
func (m *adminRepoStub) FindByUserID(_ context.Context, userID string) (*admindomain.AdminUser, error) {
	if m.findByUserFn != nil {
		return m.findByUserFn(userID)
	}
	if m.findByUserErr != nil {
		return nil, m.findByUserErr
	}
	v, ok := m.byUser[userID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return v, nil
}
func (m *adminRepoStub) FindByID(_ context.Context, adminID string) (*admindomain.AdminUser, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	v, ok := m.byID[adminID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return v, nil
}
func (m *adminRepoStub) List(context.Context) ([]*admindomain.AdminUser, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.list != nil {
		return m.list, nil
	}
	out := make([]*admindomain.AdminUser, 0, len(m.byID))
	for _, a := range m.byID {
		out = append(out, a)
	}
	return out, nil
}
func (m *adminRepoStub) Create(_ context.Context, admin *admindomain.AdminUser) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.byID[admin.ID] = admin
	m.byUser[admin.UserID] = admin
	return nil
}
func (m *adminRepoStub) Update(_ context.Context, admin *admindomain.AdminUser) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.byID[admin.ID] = admin
	m.byUser[admin.UserID] = admin
	return nil
}
func (m *adminRepoStub) CountActiveSuperAdmins(context.Context) (int, error) {
	if m.countSuperErr != nil {
		return 0, m.countSuperErr
	}
	return m.countSuper, nil
}

type userRepoStub struct {
	byID             map[string]*authdomain.User
	byEmail          map[string]*authdomain.User
	findByIDErr      error
	findByEmailErr   error
	listUsers        []*authdomain.User
	listUsersErr     error
	updateQuotaErr   error
	updateUsedErr    error
}

func newUserRepoStub() *userRepoStub {
	return &userRepoStub{
		byID:    map[string]*authdomain.User{},
		byEmail: map[string]*authdomain.User{},
	}
}

func (m *userRepoStub) FindByEmail(_ context.Context, email string) (*authdomain.User, error) {
	if m.findByEmailErr != nil {
		return nil, m.findByEmailErr
	}
	u, ok := m.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (m *userRepoStub) FindByID(_ context.Context, id string) (*authdomain.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	u, ok := m.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (m *userRepoStub) ListUsers(context.Context, string, string, int) ([]*authdomain.User, string, error) {
	if m.listUsersErr != nil {
		return nil, "", m.listUsersErr
	}
	return m.listUsers, "next", nil
}
func (m *userRepoStub) Create(context.Context, *authdomain.User) error { return nil }
func (m *userRepoStub) Update(context.Context, *authdomain.User) error { return nil }
func (m *userRepoStub) UpdateStorageQuota(context.Context, string, int64) error { return m.updateQuotaErr }
func (m *userRepoStub) UpdateStorageUsed(context.Context, string, int64) error  { return m.updateUsedErr }

type googleAdminRepoStub struct {
	listByUser map[string][]portout.GoogleAccountRecord
	listErr    error
	updateErr  error
}

func (m *googleAdminRepoStub) ListByUserID(_ context.Context, userID string) ([]portout.GoogleAccountRecord, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listByUser[userID], nil
}
func (m *googleAdminRepoStub) UpdateStatus(context.Context, string, string, string) error { return m.updateErr }

type storageRepoStub struct {
	files             []portout.FileRef
	listFilesErr      error
	sumUsed           int64
	sumUsedErr        error
	deleteFilesErr    error
	deleteFoldersErr  error
	deleteStarsErr    error
	deleteSharesErr   error
}

func (m *storageRepoStub) ListFilesByUser(context.Context, string) ([]portout.FileRef, error) {
	if m.listFilesErr != nil {
		return nil, m.listFilesErr
	}
	return m.files, nil
}
func (m *storageRepoStub) DeleteAllFilesByUser(context.Context, string) error { return m.deleteFilesErr }
func (m *storageRepoStub) DeleteAllFoldersByUser(context.Context, string) error { return m.deleteFoldersErr }
func (m *storageRepoStub) DeleteAllStarsByUser(context.Context, string) error { return m.deleteStarsErr }
func (m *storageRepoStub) DeleteSharesByOwner(context.Context, string) error { return m.deleteSharesErr }
func (m *storageRepoStub) SumStorageUsed(context.Context, string) (int64, error) {
	if m.sumUsedErr != nil {
		return 0, m.sumUsedErr
	}
	return m.sumUsed, nil
}

func seedAdmin(admins *adminRepoStub, userID string, role admindomain.Role, active bool) *admindomain.AdminUser {
	a := &admindomain.AdminUser{
		ID:       "admin-" + userID,
		UserID:   userID,
		Role:     role,
		IsActive: active,
	}
	admins.byID[a.ID] = a
	admins.byUser[a.UserID] = a
	return a
}

func TestAdminUseCaseCoreFlows(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	users.byID["u1"] = &authdomain.User{ID: "u1", Email: "u1@example.com", StorageUsedBytes: 100}
	users.byEmail["u1@example.com"] = users.byID["u1"]
	users.listUsers = []*authdomain.User{users.byID["u1"]}
	google.listByUser["u1"] = []portout.GoogleAccountRecord{{ID: "ga1", GoogleEmail: "u1@gmail.com", Status: "active"}}
	seedAdmin(admins, "actor-admin", admindomain.RoleAdmin, true)
	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)

	if _, _, err := uc.ListUsers(ctx, "missing-admin", "", "", 20); err == nil {
		t.Fatal("expected admin access denied")
	}
	if _, _, err := uc.ListUsers(ctx, "actor-admin", "", "", 20); err != nil {
		t.Fatalf("list users: %v", err)
	}
	if _, err := uc.GetUserDetail(ctx, "missing-admin", "u1"); err == nil {
		t.Fatal("expected admin access denied for get user detail")
	}

	if _, err := uc.GetUserDetail(ctx, "actor-admin", "missing"); err == nil {
		t.Fatal("expected user not found")
	}
	if detail, err := uc.GetUserDetail(ctx, "actor-admin", "u1"); err != nil || len(detail.GoogleAccounts) != 1 {
		t.Fatalf("get user detail failed: %v", err)
	}

	if err := uc.UpdateStorageQuota(ctx, "actor-admin", "u1", 0); err == nil {
		t.Fatal("expected positive quota validation")
	}
	if err := uc.UpdateStorageQuota(ctx, "missing-admin", "u1", 200); err == nil {
		t.Fatal("expected admin access denied for update quota")
	}
	if err := uc.UpdateStorageQuota(ctx, "actor-admin", "missing", 200); err == nil {
		t.Fatal("expected target user not found")
	}
	if err := uc.UpdateStorageQuota(ctx, "actor-admin", "u1", 50); err == nil {
		t.Fatal("expected cannot set quota below used")
	}
	users.updateQuotaErr = errors.New("quota update fail")
	if err := uc.UpdateStorageQuota(ctx, "actor-admin", "u1", 200); err == nil {
		t.Fatal("expected update quota error")
	}
	users.updateQuotaErr = nil
	if err := uc.UpdateStorageQuota(ctx, "actor-admin", "u1", 200); err != nil {
		t.Fatalf("update storage quota: %v", err)
	}

	if _, err := uc.RecalculateStorageUsed(ctx, "actor-admin", "missing"); err == nil {
		t.Fatal("expected user not found")
	}
	if _, err := uc.RecalculateStorageUsed(ctx, "missing-admin", "u1"); err == nil {
		t.Fatal("expected admin access denied for recalculate storage")
	}
	storage.sumUsedErr = errors.New("sum fail")
	if _, err := uc.RecalculateStorageUsed(ctx, "actor-admin", "u1"); err == nil {
		t.Fatal("expected sum storage error")
	}
	storage.sumUsedErr = nil
	storage.sumUsed = 321
	users.updateUsedErr = errors.New("update used fail")
	if _, err := uc.RecalculateStorageUsed(ctx, "actor-admin", "u1"); err == nil {
		t.Fatal("expected update used error")
	}
	users.updateUsedErr = nil
	if used, err := uc.RecalculateStorageUsed(ctx, "actor-admin", "u1"); err != nil || used != 321 {
		t.Fatalf("recalculate storage used failed: used=%d err=%v", used, err)
	}

	if err := uc.UpdateGoogleAccountStatus(ctx, "actor-admin", "u1", "ga1", "wrong"); err == nil {
		t.Fatal("expected invalid status")
	}
	if err := uc.UpdateGoogleAccountStatus(ctx, "missing-admin", "u1", "ga1", "active"); err == nil {
		t.Fatal("expected admin access denied for update google account status")
	}
	if err := uc.UpdateGoogleAccountStatus(ctx, "actor-admin", "missing", "ga1", "active"); err == nil {
		t.Fatal("expected target user not found")
	}
	google.updateErr = pgx.ErrNoRows
	if err := uc.UpdateGoogleAccountStatus(ctx, "actor-admin", "u1", "ga1", "active"); err == nil {
		t.Fatal("expected google account not found")
	}
	google.updateErr = errors.New("db fail")
	if err := uc.UpdateGoogleAccountStatus(ctx, "actor-admin", "u1", "ga1", "active"); err == nil {
		t.Fatal("expected update google status error")
	}
	google.updateErr = nil
	if err := uc.UpdateGoogleAccountStatus(ctx, "actor-admin", "u1", "ga1", "reauth_required"); err != nil {
		t.Fatalf("update google status: %v", err)
	}
}

func TestAdminUseCaseResetStorage(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	dataPath := t.TempDir()
	thumbPath := t.TempDir()
	uc := NewAdminUseCase(admins, users, google, storage, dataPath, thumbPath)

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	users.byID["u1"] = &authdomain.User{ID: "u1", Email: "u1@example.com"}

	if err := uc.ResetUserStorage(ctx, "missing", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected super admin required")
	}
	if err := uc.ResetUserStorage(ctx, "actor-super", "missing", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected user not found")
	}
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "wrong"); err == nil {
		t.Fatal("expected confirmation mismatch")
	}

	userDir := filepath.Join(dataPath, "u1")
	thumbDir := filepath.Join(thumbPath, "u1")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("mkdir user dir: %v", err)
	}
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		t.Fatalf("mkdir thumb dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "f1.bin"), []byte("x"), 0o600); err != nil {
		t.Fatalf("write data file: %v", err)
	}
	storage.files = []portout.FileRef{{ID: "f1", StoragePath: filepath.Join("u1", "f1.bin")}}

	storage.listFilesErr = errors.New("list files fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected list files error")
	}
	storage.listFilesErr = nil

	storage.deleteStarsErr = errors.New("delete stars fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected delete stars error")
	}
	storage.deleteStarsErr = nil

	storage.deleteSharesErr = errors.New("delete shares fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected delete shares error")
	}
	storage.deleteSharesErr = nil

	storage.deleteFilesErr = errors.New("delete files fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected delete files error")
	}
	storage.deleteFilesErr = nil

	storage.deleteFoldersErr = errors.New("delete folders fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected delete folders error")
	}
	storage.deleteFoldersErr = nil

	users.updateUsedErr = errors.New("update used fail")
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err == nil {
		t.Fatal("expected update used error")
	}
	users.updateUsedErr = nil
	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err != nil {
		t.Fatalf("reset user storage failed: %v", err)
	}
}

func TestAdminUseCaseResetStorageWarningBranches(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{
		files: []portout.FileRef{{ID: "f1", StoragePath: filepath.Join("u1", "f1.bin")}},
	}

	dataRoot := filepath.Join(t.TempDir(), "data-root-file")
	if err := os.WriteFile(dataRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write data root file: %v", err)
	}
	thumbRoot := filepath.Join(t.TempDir(), "thumb-root-file")
	if err := os.WriteFile(thumbRoot, []byte("x"), 0o600); err != nil {
		t.Fatalf("write thumb root file: %v", err)
	}

	uc := NewAdminUseCase(admins, users, google, storage, dataRoot, thumbRoot)
	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	users.byID["u1"] = &authdomain.User{ID: "u1", Email: "u1@example.com"}

	if err := uc.ResetUserStorage(ctx, "actor-super", "u1", "DELETE u1@example.com"); err != nil {
		t.Fatalf("reset user storage with warning branches failed: %v", err)
	}
}

func TestAdminUseCaseAdminRoleManagement(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	now := time.Now()
	super := &admindomain.AdminUser{ID: "a-super", UserID: "actor-super", Role: admindomain.RoleSuperAdmin, IsActive: true, CreatedAt: now, UpdatedAt: now}
	target := &admindomain.AdminUser{ID: "a-target", UserID: "target", Role: admindomain.RoleSuperAdmin, IsActive: true, CreatedAt: now, UpdatedAt: now}
	admins.byID[super.ID], admins.byUser[super.UserID] = super, super
	admins.byID[target.ID], admins.byUser[target.UserID] = target, target
	admins.list = []*admindomain.AdminUser{super, target}
	admins.countSuper = 2
	users.byID["actor-super"] = &authdomain.User{ID: "actor-super", Email: "super@example.com", Name: "Super"}
	users.byID["target"] = &authdomain.User{ID: "target", Email: "target@example.com", Name: "Target"}
	users.byEmail["target@example.com"] = users.byID["target"]
	users.byEmail["super@example.com"] = users.byID["actor-super"]

	if _, err := uc.ListAdmins(ctx, "actor-admin"); err == nil {
		t.Fatal("expected super admin required")
	}
	if adminsList, err := uc.ListAdmins(ctx, "actor-super"); err != nil || len(adminsList) != 2 {
		t.Fatalf("list admins failed: %v", err)
	}

	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", "wrong"); err == nil {
		t.Fatal("expected invalid role")
	}
	if _, err := uc.CreateAdmin(ctx, "actor-super", "missing@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected target user not found")
	}
	if _, err := uc.CreateAdmin(ctx, "actor-admin", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected super admin required when active admins exist")
	}

	admins.countSuperErr = errors.New("count fail")
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected count super error")
	}
	admins.countSuperErr = nil

	admins.byUser["target"] = &admindomain.AdminUser{
		ID: "a-existing", UserID: "target", Role: admindomain.RoleAdmin, IsActive: false,
	}
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleSuperAdmin); err != nil {
		t.Fatalf("reactivate existing admin: %v", err)
	}

	admins.findByUserErr = errors.New("db fail")
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected find existing admin error")
	}
	admins.findByUserErr = nil
	admins.byUser = map[string]*admindomain.AdminUser{
		"actor-super": super,
	}
	admins.createErr = errors.New("create fail")
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected create admin error")
	}
	admins.createErr = nil
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err != nil {
		t.Fatalf("create admin success path failed: %v", err)
	}

	admins.byUser["target"] = nil
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err != nil {
		t.Fatalf("create admin should tolerate nil existing entry: %v", err)
	}

	// Bootstrap first admin rule.
	admins.byID = map[string]*admindomain.AdminUser{}
	admins.byUser = map[string]*admindomain.AdminUser{}
	admins.countSuper = 0
	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected bootstrap self super admin rule")
	}
	users.byEmail["actor-self@example.com"] = &authdomain.User{ID: "actor-self", Email: "actor-self@example.com", Name: "Self"}
	if _, err := uc.CreateAdmin(ctx, "actor-self", "actor-self@example.com", admindomain.RoleSuperAdmin); err != nil {
		t.Fatalf("bootstrap self super admin failed: %v", err)
	}

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	admins.byID["admin-target"] = &admindomain.AdminUser{ID: "admin-target", UserID: "target", Role: admindomain.RoleSuperAdmin, IsActive: true}
	admins.byUser["target"] = admins.byID["admin-target"]
	admins.countSuper = 1

	if err := uc.UpdateAdminRole(ctx, "actor-super", "admin-target", "wrong"); err == nil {
		t.Fatal("expected invalid role")
	}
	if err := uc.UpdateAdminRole(ctx, "actor-admin", "admin-target", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected super admin required")
	}
	if err := uc.UpdateAdminRole(ctx, "actor-super", "missing", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected admin not found")
	}
	if err := uc.UpdateAdminRole(ctx, "actor-super", "admin-actor-super", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected cannot change own role")
	}
	if err := uc.UpdateAdminRole(ctx, "actor-super", "admin-target", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected cannot demote last active super admin")
	}
	admins.countSuper = 2
	admins.updateErr = errors.New("update fail")
	if err := uc.UpdateAdminRole(ctx, "actor-super", "admin-target", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected update admin role error")
	}
	admins.updateErr = nil
	if err := uc.UpdateAdminRole(ctx, "actor-super", "admin-target", admindomain.RoleAdmin); err != nil {
		t.Fatalf("update admin role success failed: %v", err)
	}

	admins.byID["admin-target"].Role = admindomain.RoleSuperAdmin
	admins.byID["admin-target"].IsActive = true
	admins.countSuper = 1
	if err := uc.DeactivateAdmin(ctx, "actor-super", "admin-target"); err == nil {
		t.Fatal("expected cannot deactivate last super admin")
	}
	admins.countSuper = 2
	admins.updateErr = errors.New("update fail")
	if err := uc.DeactivateAdmin(ctx, "actor-super", "admin-target"); err == nil {
		t.Fatal("expected deactivate update error")
	}
	admins.updateErr = nil
	if err := uc.DeactivateAdmin(ctx, "actor-super", "admin-target"); err != nil {
		t.Fatalf("deactivate admin success failed: %v", err)
	}
	if err := uc.DeactivateAdmin(ctx, "actor-admin", "admin-target"); err == nil {
		t.Fatal("expected super admin required for deactivate admin")
	}

	if err := uc.DeactivateAdmin(ctx, "actor-super", "missing"); err == nil {
		t.Fatal("expected admin not found")
	}
	if err := uc.DeactivateAdmin(ctx, "actor-super", "admin-actor-super"); err == nil {
		t.Fatal("expected cannot deactivate yourself")
	}
}

func TestAdminUseCaseRequireSuperAdminBranches(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	seedAdmin(admins, "actor-admin", admindomain.RoleAdmin, true)
	admins.countSuper = 1
	users.byID["target"] = &authdomain.User{ID: "target", Email: "target@example.com", StorageUsedBytes: 0}

	if _, err := uc.ListAdmins(ctx, "actor-admin"); err == nil {
		t.Fatal("expected super admin required for list admins")
	}

	admins.listErr = errors.New("list admins fail")
	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	if _, err := uc.ListAdmins(ctx, "actor-super"); err == nil {
		t.Fatal("expected list admins repository error")
	}
	admins.listErr = nil

	google.listErr = errors.New("google list fail")
	if _, err := uc.GetUserDetail(ctx, "actor-admin", "target"); err == nil {
		t.Fatal("expected google account list error")
	}
}

func TestAdminUseCaseListAdminsSkipsMissingUser(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	missing := &admindomain.AdminUser{ID: "a-missing", UserID: "u-missing", Role: admindomain.RoleAdmin, IsActive: true}
	existing := &admindomain.AdminUser{ID: "a-existing", UserID: "u-existing", Role: admindomain.RoleAdmin, IsActive: true}
	admins.list = []*admindomain.AdminUser{missing, existing}
	users.byID["u-existing"] = &authdomain.User{ID: "u-existing", Email: "u@example.com", Name: "U"}

	list, err := uc.ListAdmins(ctx, "actor-super")
	if err != nil {
		t.Fatalf("list admins failed: %v", err)
	}
	if len(list) != 1 || list[0].UserID != "u-existing" {
		t.Fatalf("expected only existing user admin in result: %#v", list)
	}
}

func TestAdminUseCaseCreateAdminExistingUpdateError(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	users.byEmail["target@example.com"] = &authdomain.User{ID: "target", Email: "target@example.com", Name: "Target"}
	admins.byUser["target"] = &admindomain.AdminUser{
		ID: "a-target", UserID: "target", Role: admindomain.RoleAdmin, IsActive: false,
	}
	admins.countSuper = 1
	admins.updateErr = errors.New("update fail")

	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleSuperAdmin); err == nil {
		t.Fatal("expected existing admin update error")
	}
}

func TestAdminUseCaseCreateAdminFindExistingUnexpectedError(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	admins.countSuper = 1
	admins.findByUserFn = func(userID string) (*admindomain.AdminUser, error) {
		if userID == "actor-super" {
			return admins.byUser[userID], nil
		}
		return nil, errors.New("find existing fail")
	}
	users.byEmail["target@example.com"] = &authdomain.User{ID: "target", Email: "target@example.com", Name: "Target"}

	if _, err := uc.CreateAdmin(ctx, "actor-super", "target@example.com", admindomain.RoleAdmin); err == nil {
		t.Fatal("expected unexpected find existing error")
	}
}

func TestAdminUseCaseAdditionalErrorBranches(t *testing.T) {
	ctx := context.Background()
	admins := newAdminRepoStub()
	users := newUserRepoStub()
	google := &googleAdminRepoStub{listByUser: map[string][]portout.GoogleAccountRecord{}}
	storage := &storageRepoStub{}
	uc := NewAdminUseCase(admins, users, google, storage, t.TempDir(), t.TempDir())

	seedAdmin(admins, "actor-super", admindomain.RoleSuperAdmin, true)
	target := seedAdmin(admins, "target-user", admindomain.RoleSuperAdmin, true)
	target.ID = "target-admin-id"
	admins.byID[target.ID] = target
	admins.countSuper = 2
	users.listUsersErr = errors.New("list users fail")

	if _, _, err := uc.ListUsers(ctx, "actor-super", "", "", 10); err == nil {
		t.Fatal("expected list users error")
	}

	admins.countSuperErr = errors.New("count super fail")
	if err := uc.UpdateAdminRole(ctx, "actor-super", target.ID, admindomain.RoleAdmin); err == nil {
		t.Fatal("expected update role count super error")
	}

	admins.countSuperErr = errors.New("count super fail")
	if err := uc.DeactivateAdmin(ctx, "actor-super", target.ID); err == nil {
		t.Fatal("expected deactivate count super error")
	}
}

var _ portin.AdminUseCase = (*adminUseCase)(nil)

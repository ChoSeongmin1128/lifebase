package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/cloud/domain"
)

type scanErrorRows struct {
	nextCalled bool
}

func (r *scanErrorRows) Close() {}
func (r *scanErrorRows) Err() error {
	return nil
}
func (r *scanErrorRows) CommandTag() pgconn.CommandTag {
	return pgconn.CommandTag{}
}
func (r *scanErrorRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (r *scanErrorRows) Next() bool {
	if r.nextCalled {
		return false
	}
	r.nextCalled = true
	return true
}
func (r *scanErrorRows) Scan(dest ...any) error {
	return errors.New("scan failed")
}
func (r *scanErrorRows) Values() ([]any, error) {
	return nil, nil
}
func (r *scanErrorRows) RawValues() [][]byte {
	return nil
}
func (r *scanErrorRows) Conn() *pgx.Conn {
	return nil
}

func newUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://lifebase:lifebase@127.0.0.1:1/lifebase?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestCloudPostgresReposDBErrorBranches(t *testing.T) {
	db := newUnreachablePool(t)
	ctx := context.Background()
	now := time.Now().UTC()
	userID := "u1"
	folderID := "f1"
	fileID := "file1"

	fileRepo := NewFileRepo(db)
	folderRepo := NewFolderRepo(db)
	sharedRepo := NewSharedRepo(db)
	starRepo := NewStarRepo(db)

	if err := fileRepo.Create(ctx, &domain.File{
		ID: fileID, UserID: userID, Name: "a.txt", MimeType: "text/plain",
		StoragePath: "u1/file1", SizeBytes: 1, CreatedAt: now, UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected file create error")
	}
	if _, err := fileRepo.FindByID(ctx, userID, fileID); err == nil {
		t.Fatal("expected file find error")
	}
	if _, err := fileRepo.ListByFolder(ctx, userID, nil, "name", "asc"); err == nil {
		t.Fatal("expected file list root error")
	}
	if _, err := fileRepo.ListByFolder(ctx, userID, &folderID, "size", "desc"); err == nil {
		t.Fatal("expected file list child error")
	}
	if _, err := fileRepo.ListRecent(ctx, userID, 1); err == nil {
		t.Fatal("expected file list recent error")
	}
	if err := fileRepo.Update(ctx, &domain.File{
		ID: fileID, UserID: userID, FolderID: &folderID, Name: "b.txt", MimeType: "text/plain",
		StoragePath: "u1/file1", SizeBytes: 2, UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected file update error")
	}
	if err := fileRepo.SoftDelete(ctx, userID, fileID); err == nil {
		t.Fatal("expected file soft delete error")
	}
	if _, err := fileRepo.FindTrashedByID(ctx, userID, fileID); err == nil {
		t.Fatal("expected file find trashed error")
	}
	if err := fileRepo.Restore(ctx, userID, fileID); err == nil {
		t.Fatal("expected file restore error")
	}
	if err := fileRepo.HardDelete(ctx, fileID); err == nil {
		t.Fatal("expected file hard delete error")
	}
	if _, err := fileRepo.ListTrashed(ctx, userID); err == nil {
		t.Fatal("expected file list trashed error")
	}
	if err := fileRepo.UpdateStorageUsed(ctx, userID, 1); err == nil {
		t.Fatal("expected update storage used error")
	}
	if _, err := fileRepo.ExistsByName(ctx, userID, nil, "a.txt"); err == nil {
		t.Fatal("expected exists by name root error")
	}
	if _, err := fileRepo.ExistsByName(ctx, userID, &folderID, "a.txt"); err == nil {
		t.Fatal("expected exists by name child error")
	}
	if _, err := fileRepo.Search(ctx, userID, "a", 10); err == nil {
		t.Fatal("expected search error")
	}

	if err := folderRepo.Create(ctx, &domain.Folder{
		ID: folderID, UserID: userID, Name: "folder", CreatedAt: now, UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected folder create error")
	}
	if _, err := folderRepo.FindByID(ctx, userID, folderID); err == nil {
		t.Fatal("expected folder find error")
	}
	if _, err := folderRepo.ListByParent(ctx, userID, nil); err == nil {
		t.Fatal("expected folder list root error")
	}
	if _, err := folderRepo.ListByParent(ctx, userID, &folderID); err == nil {
		t.Fatal("expected folder list child error")
	}
	if err := folderRepo.Update(ctx, &domain.Folder{
		ID: folderID, UserID: userID, Name: "new", UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected folder update error")
	}
	if err := folderRepo.SoftDelete(ctx, userID, folderID); err == nil {
		t.Fatal("expected folder soft delete error")
	}
	if err := folderRepo.Restore(ctx, userID, folderID); err == nil {
		t.Fatal("expected folder restore error")
	}
	if err := folderRepo.HardDelete(ctx, folderID); err == nil {
		t.Fatal("expected folder hard delete error")
	}
	if _, err := folderRepo.ListTrashed(ctx, userID); err == nil {
		t.Fatal("expected folder list trashed error")
	}

	if _, err := sharedRepo.ListSharedFolders(ctx, userID); err == nil {
		t.Fatal("expected list shared folders error")
	}
	if _, err := starRepo.List(ctx, userID); err == nil {
		t.Fatal("expected list stars error")
	}
	if err := starRepo.Set(ctx, userID, "item-1", "file"); err == nil {
		t.Fatal("expected set star error")
	}
	if err := starRepo.Unset(ctx, userID, "item-1", "file"); err == nil {
		t.Fatal("expected unset star error")
	}
}

func TestScanHelpersPropagateScanError(t *testing.T) {
	t.Run("scanFiles", func(t *testing.T) {
		rows := &scanErrorRows{}
		if _, err := scanFiles(rows); err == nil {
			t.Fatal("expected scanFiles to return scan error")
		}
	})

	t.Run("scanFolders", func(t *testing.T) {
		rows := &scanErrorRows{}
		if _, err := scanFolders(rows); err == nil {
			t.Fatal("expected scanFolders to return scan error")
		}
	})
}

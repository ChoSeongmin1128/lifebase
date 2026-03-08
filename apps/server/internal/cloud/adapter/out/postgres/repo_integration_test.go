package postgres

import (
	"context"
	"testing"
	"time"

	"lifebase/internal/cloud/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestFolderAndFileReposIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	rootID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	childID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	fileRootID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	fileChildID := "dddddddd-dddd-dddd-dddd-dddddddddddd"

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, picture, storage_quota_bytes, storage_used_bytes, created_at, updated_at)
		 VALUES ($1, 'u1@example.com', 'u1', '', 1000, 0, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	folderRepo := NewFolderRepo(db)
	fileRepo := NewFileRepo(db)

	root := &domain.Folder{
		ID:        rootID,
		UserID:    userID,
		Name:      "Root A",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := folderRepo.Create(ctx, root); err != nil {
		t.Fatalf("create root folder: %v", err)
	}
	child := &domain.Folder{
		ID:        childID,
		UserID:    userID,
		ParentID:  &rootID,
		Name:      "Child B",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := folderRepo.Create(ctx, child); err != nil {
		t.Fatalf("create child folder: %v", err)
	}

	gotRoot, err := folderRepo.FindByID(ctx, userID, rootID)
	if err != nil || gotRoot.Name != "Root A" {
		t.Fatalf("find root folder failed: err=%v folder=%#v", err, gotRoot)
	}
	if _, err := folderRepo.FindByID(ctx, userID, "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"); err == nil {
		t.Fatal("expected folder not found error")
	}

	rootFolders, err := folderRepo.ListByParent(ctx, userID, nil)
	if err != nil || len(rootFolders) != 1 || rootFolders[0].ID != rootID {
		t.Fatalf("list root folders failed: err=%v folders=%#v", err, rootFolders)
	}
	childFolders, err := folderRepo.ListByParent(ctx, userID, &rootID)
	if err != nil || len(childFolders) != 1 || childFolders[0].ID != childID {
		t.Fatalf("list child folders failed: err=%v folders=%#v", err, childFolders)
	}

	child.Name = "Child Renamed"
	child.ParentID = nil
	child.UpdatedAt = now.Add(time.Minute)
	if err := folderRepo.Update(ctx, child); err != nil {
		t.Fatalf("update child folder: %v", err)
	}

	fRoot := &domain.File{
		ID:          fileRootID,
		UserID:      userID,
		Name:        "alpha.txt",
		MimeType:    "text/plain",
		SizeBytes:   10,
		StoragePath: userID + "/" + fileRootID,
		ThumbStatus: "done",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := fileRepo.Create(ctx, fRoot); err != nil {
		t.Fatalf("create root file: %v", err)
	}
	fChild := &domain.File{
		ID:          fileChildID,
		UserID:      userID,
		FolderID:    &childID,
		Name:        "photo-sunrise.jpg",
		MimeType:    "image/jpeg",
		SizeBytes:   120,
		StoragePath: userID + "/" + fileChildID,
		ThumbStatus: "pending",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := fileRepo.Create(ctx, fChild); err != nil {
		t.Fatalf("create child file: %v", err)
	}

	gotFile, err := fileRepo.FindByID(ctx, userID, fileChildID)
	if err != nil || gotFile.Name != "photo-sunrise.jpg" {
		t.Fatalf("find file failed: err=%v file=%#v", err, gotFile)
	}
	if _, err := fileRepo.FindByID(ctx, userID, "ffffffff-ffff-ffff-ffff-ffffffffffff"); err == nil {
		t.Fatal("expected file not found error")
	}

	rootFiles, err := fileRepo.ListByFolder(ctx, userID, nil, "name", "asc")
	if err != nil || len(rootFiles) != 1 || rootFiles[0].ID != fileRootID {
		t.Fatalf("list root files failed: err=%v files=%#v", err, rootFiles)
	}
	childFiles, err := fileRepo.ListByFolder(ctx, userID, &childID, "size", "desc")
	if err != nil || len(childFiles) != 1 || childFiles[0].ID != fileChildID {
		t.Fatalf("list child files failed: err=%v files=%#v", err, childFiles)
	}

	recent, err := fileRepo.ListRecent(ctx, userID, 1)
	if err != nil || len(recent) != 1 {
		t.Fatalf("list recent with limit failed: err=%v files=%#v", err, recent)
	}
	recentDefaultLimit, err := fileRepo.ListRecent(ctx, userID, 0)
	if err != nil || len(recentDefaultLimit) != 2 {
		t.Fatalf("list recent default limit failed: err=%v files=%#v", err, recentDefaultLimit)
	}

	fChild.Name = "vacation-photo.jpg"
	fChild.MimeType = "image/webp"
	fChild.SizeBytes = 333
	fChild.StoragePath = userID + "/new/" + fileChildID
	fChild.UpdatedAt = now.Add(2 * time.Minute)
	if err := fileRepo.Update(ctx, fChild); err != nil {
		t.Fatalf("update file: %v", err)
	}
	updatedFile, err := fileRepo.FindByID(ctx, userID, fileChildID)
	if err != nil || updatedFile.Name != "vacation-photo.jpg" {
		t.Fatalf("updated file mismatch: err=%v file=%#v", err, updatedFile)
	}

	existsRoot, err := fileRepo.ExistsByName(ctx, userID, nil, "alpha.txt")
	if err != nil || !existsRoot {
		t.Fatalf("exists by name root failed: exists=%v err=%v", existsRoot, err)
	}
	existsChild, err := fileRepo.ExistsByName(ctx, userID, &childID, "vacation-photo.jpg")
	if err != nil || !existsChild {
		t.Fatalf("exists by name child failed: exists=%v err=%v", existsChild, err)
	}

	foundBySearch, err := fileRepo.Search(ctx, userID, "vacation-photo.jpg", 10)
	if err != nil || len(foundBySearch) == 0 {
		t.Fatalf("search failed: err=%v files=%#v", err, foundBySearch)
	}

	if err := fileRepo.SoftDelete(ctx, userID, fileChildID); err != nil {
		t.Fatalf("soft delete file: %v", err)
	}
	if _, err := fileRepo.FindByID(ctx, userID, fileChildID); err == nil {
		t.Fatal("expected soft deleted file not found")
	}
	trashedFile, err := fileRepo.FindTrashedByID(ctx, userID, fileChildID)
	if err != nil || trashedFile.ID != fileChildID {
		t.Fatalf("find trashed by id failed: err=%v file=%#v", err, trashedFile)
	}
	if _, err := fileRepo.FindTrashedByID(ctx, userID, "ffffffff-ffff-ffff-ffff-ffffffffffff"); err == nil {
		t.Fatal("expected missing trashed file to return not found error")
	}
	trashedFiles, err := fileRepo.ListTrashed(ctx, userID)
	if err != nil || len(trashedFiles) != 1 {
		t.Fatalf("list trashed files failed: err=%v files=%#v", err, trashedFiles)
	}
	if err := fileRepo.Restore(ctx, userID, fileChildID); err != nil {
		t.Fatalf("restore file: %v", err)
	}
	if _, err := fileRepo.FindByID(ctx, userID, fileChildID); err != nil {
		t.Fatalf("find restored file: %v", err)
	}

	if err := folderRepo.SoftDelete(ctx, userID, childID); err != nil {
		t.Fatalf("soft delete folder: %v", err)
	}
	if _, err := folderRepo.FindByID(ctx, userID, childID); err == nil {
		t.Fatal("expected soft deleted folder not found")
	}
	trashedFolder, err := folderRepo.FindTrashedByID(ctx, userID, childID)
	if err != nil || trashedFolder.ID != childID {
		t.Fatalf("find trashed folder by id failed: err=%v folder=%#v", err, trashedFolder)
	}
	if _, err := folderRepo.FindTrashedByID(ctx, userID, "ffffffff-ffff-ffff-ffff-ffffffffffff"); err == nil {
		t.Fatal("expected missing trashed folder to return not found error")
	}
	trashedFolders, err := folderRepo.ListTrashed(ctx, userID)
	if err != nil || len(trashedFolders) != 1 {
		t.Fatalf("list trashed folders failed: err=%v folders=%#v", err, trashedFolders)
	}
	existsFolderName, err := folderRepo.ExistsByName(ctx, userID, nil, "Child Renamed")
	if err != nil || existsFolderName {
		t.Fatalf("expected trashed folder name not to count as active: exists=%v err=%v", existsFolderName, err)
	}
	if err := folderRepo.Restore(ctx, userID, childID); err != nil {
		t.Fatalf("restore folder: %v", err)
	}
	if _, err := folderRepo.FindByID(ctx, userID, childID); err != nil {
		t.Fatalf("find restored folder: %v", err)
	}
	existsFolderName, err = folderRepo.ExistsByName(ctx, userID, nil, "Child Renamed")
	if err != nil || !existsFolderName {
		t.Fatalf("expected restored folder name to count as active: exists=%v err=%v", existsFolderName, err)
	}
	existsChildFolderName, err := folderRepo.ExistsByName(ctx, userID, &rootID, "Child Renamed")
	if err != nil || existsChildFolderName {
		t.Fatalf("expected no child-named folder under original root after parent move: exists=%v err=%v", existsChildFolderName, err)
	}
	grandChildID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	grandChild := &domain.Folder{
		ID:        grandChildID,
		UserID:    userID,
		ParentID:  &rootID,
		Name:      "Grand Child",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := folderRepo.Create(ctx, grandChild); err != nil {
		t.Fatalf("create grand child folder: %v", err)
	}
	existsChildFolderName, err = folderRepo.ExistsByName(ctx, userID, &rootID, "Grand Child")
	if err != nil || !existsChildFolderName {
		t.Fatalf("expected child folder exists by name under parent: exists=%v err=%v", existsChildFolderName, err)
	}
	existsFolderNameInParent, err := folderRepo.ExistsByName(ctx, userID, &rootID, "Child Renamed")
	if err != nil || existsFolderNameInParent {
		t.Fatalf("expected moved child folder name lookup under old parent to be false: exists=%v err=%v", existsFolderNameInParent, err)
	}

	if err := fileRepo.HardDelete(ctx, fileRootID); err != nil {
		t.Fatalf("hard delete file: %v", err)
	}
	if _, err := fileRepo.FindByID(ctx, userID, fileRootID); err == nil {
		t.Fatal("expected hard deleted file not found")
	}

	if err := fileRepo.UpdateStorageUsed(ctx, userID, 123); err != nil {
		t.Fatalf("update storage used: %v", err)
	}
	var storageUsed int64
	if err := db.QueryRow(ctx, `SELECT storage_used_bytes FROM users WHERE id = $1`, userID).Scan(&storageUsed); err != nil {
		t.Fatalf("read storage used: %v", err)
	}
	if storageUsed != 123 {
		t.Fatalf("expected storage_used_bytes=123, got %d", storageUsed)
	}

	if err := folderRepo.HardDelete(ctx, childID); err != nil {
		t.Fatalf("hard delete folder: %v", err)
	}
	if _, err := folderRepo.FindByID(ctx, userID, childID); err == nil {
		t.Fatal("expected hard deleted folder not found")
	}
}

func TestSharedAndStarReposIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const ownerID = "11111111-1111-1111-1111-111111111111"
	const sharedWith = "22222222-2222-2222-2222-222222222222"
	const folderID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO folders (id, user_id, parent_id, name, created_at, updated_at)
		 VALUES ($1, $2, NULL, 'Shared Folder', $3, $3)`,
		folderID, ownerID, now,
	)
	if err != nil {
		t.Fatalf("insert shared folder: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO shares (id, folder_id, owner_id, shared_with, role, created_at, updated_at)
		 VALUES
		   ('share-1', $1, $2, $3, 'viewer', $4, $4),
		   ('share-2', $1, $2, $3, 'editor', $5, $5)`,
		folderID, ownerID, sharedWith, now, now.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("insert shares: %v", err)
	}

	sharedRepo := NewSharedRepo(db)
	folders, err := sharedRepo.ListSharedFolders(ctx, sharedWith)
	if err != nil || len(folders) != 1 || folders[0].ID != folderID {
		t.Fatalf("list shared folders failed: err=%v folders=%#v", err, folders)
	}

	starRepo := NewStarRepo(db)
	if err := starRepo.Set(ctx, sharedWith, "file-1", "file"); err != nil {
		t.Fatalf("set file star: %v", err)
	}
	if err := starRepo.Set(ctx, sharedWith, "folder-1", "folder"); err != nil {
		t.Fatalf("set folder star: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, 'cloud_star:invalid', '1', $2)`,
		sharedWith, now,
	)
	if err != nil {
		t.Fatalf("insert invalid star key: %v", err)
	}

	refs, err := starRepo.List(ctx, sharedWith)
	if err != nil {
		t.Fatalf("list stars: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 valid star refs, got %#v", refs)
	}

	if err := starRepo.Unset(ctx, sharedWith, "file-1", "file"); err != nil {
		t.Fatalf("unset file star: %v", err)
	}
	refs, err = starRepo.List(ctx, sharedWith)
	if err != nil || len(refs) != 1 {
		t.Fatalf("list stars after unset failed: err=%v refs=%#v", err, refs)
	}
}

func TestStarKeyHelpers(t *testing.T) {
	if got := buildStarKey("file", "abc"); got != "cloud_star:file:abc" {
		t.Fatalf("unexpected star key: %s", got)
	}

	ref, ok := parseStarKey("cloud_star:file:item-1")
	if !ok || ref.ItemType != "file" || ref.ItemID != "item-1" {
		t.Fatalf("parse valid file key failed: ok=%v ref=%#v", ok, ref)
	}
	ref, ok = parseStarKey("cloud_star:folder:item-2")
	if !ok || ref.ItemType != "folder" || ref.ItemID != "item-2" {
		t.Fatalf("parse valid folder key failed: ok=%v ref=%#v", ok, ref)
	}

	if _, ok := parseStarKey("wrongprefix:file:item"); ok {
		t.Fatal("expected wrong prefix to fail")
	}
	if _, ok := parseStarKey("cloud_star:file"); ok {
		t.Fatal("expected wrong part count to fail")
	}
	if _, ok := parseStarKey("cloud_star:image:item"); ok {
		t.Fatal("expected invalid item type to fail")
	}
	if _, ok := parseStarKey("cloud_star:file:"); ok {
		t.Fatal("expected empty item id to fail")
	}
}

func TestBuildOrderClauseVariants(t *testing.T) {
	cases := []struct {
		sortBy  string
		sortDir string
		want    string
	}{
		{sortBy: "name", sortDir: "asc", want: "name ASC, name ASC"},
		{sortBy: "size", sortDir: "desc", want: "size_bytes DESC, name ASC"},
		{sortBy: "updated_at", sortDir: "asc", want: "updated_at ASC, name ASC"},
		{sortBy: "created_at", sortDir: "desc", want: "created_at DESC, name ASC"},
		{sortBy: "unknown", sortDir: "unknown", want: "name ASC, name ASC"},
	}
	for _, tc := range cases {
		if got := buildOrderClause(tc.sortBy, tc.sortDir); got != tc.want {
			t.Fatalf("buildOrderClause(%q,%q)=%q want=%q", tc.sortBy, tc.sortDir, got, tc.want)
		}
	}
}

func TestCloudReposClosedPoolErrors(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	fileRepo := NewFileRepo(db)
	folderRepo := NewFolderRepo(db)
	sharedRepo := NewSharedRepo(db)
	starRepo := NewStarRepo(db)
	db.Close()

	if _, err := fileRepo.ListByFolder(ctx, "u1", nil, "name", "asc"); err == nil {
		t.Fatal("expected list by folder query error")
	}
	if _, err := fileRepo.ListRecent(ctx, "u1", 1); err == nil {
		t.Fatal("expected list recent query error")
	}
	if _, err := fileRepo.ListTrashed(ctx, "u1"); err == nil {
		t.Fatal("expected list trashed query error")
	}
	if _, err := fileRepo.Search(ctx, "u1", "a", 1); err == nil {
		t.Fatal("expected search query error")
	}
	if _, err := fileRepo.FindTrashedByID(ctx, "u1", "x"); err == nil {
		t.Fatal("expected find trashed query error")
	}
	if _, err := folderRepo.ListByParent(ctx, "u1", nil); err == nil {
		t.Fatal("expected folder list by parent query error")
	}
	if _, err := folderRepo.ListTrashed(ctx, "u1"); err == nil {
		t.Fatal("expected folder list trashed query error")
	}
	if _, err := sharedRepo.ListSharedFolders(ctx, "u1"); err == nil {
		t.Fatal("expected shared folders query error")
	}
	if _, err := starRepo.List(ctx, "u1"); err == nil {
		t.Fatal("expected stars query error")
	}
}

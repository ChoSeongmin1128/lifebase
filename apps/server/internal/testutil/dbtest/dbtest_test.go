package dbtest

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeT struct {
	skipped bool
	fatal   string
}

func (f *fakeT) Helper() {}
func (f *fakeT) Skip(args ...any) { f.skipped = true }
func (f *fakeT) Fatalf(format string, args ...any) { f.fatal = fmt.Sprintf(format, args...) }

type fakeRows struct {
	nextResults []bool
	scanErr     error
	err         error
}

func (r *fakeRows) Close() {}
func (r *fakeRows) Err() error { return r.err }
func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	if len(r.nextResults) == 0 {
		return false
	}
	next := r.nextResults[0]
	r.nextResults = r.nextResults[1:]
	return next
}
func (r *fakeRows) Scan(dest ...any) error { return r.scanErr }
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte { return nil }
func (r *fakeRows) Conn() *pgx.Conn { return nil }

func TestOpenPoolBranches(t *testing.T) {
	db, skip, err := openPool(context.Background(), "")
	if err != nil || !skip || db != nil {
		t.Fatalf("expected skip branch, got db=%v skip=%v err=%v", db, skip, err)
	}

	if _, skip, err := openPool(context.Background(), "://bad"); err == nil || skip {
		t.Fatalf("expected invalid dsn error, got skip=%v err=%v", skip, err)
	}

	if _, skip, err := openPool(context.Background(), "postgres://seongmin@localhost:1/lifebase_test?sslmode=disable"); err == nil || skip {
		t.Fatalf("expected ping/open error, got skip=%v err=%v", skip, err)
	}
}

func TestMigrationDirBranches(t *testing.T) {
	prev := callerFn
	t.Cleanup(func() { callerFn = prev })

	callerFn = func(skip int) (uintptr, string, int, bool) {
		return 0, "", 0, false
	}
	if _, err := migrationDir(); err == nil {
		t.Fatal("expected migrationDir error when runtime caller is unavailable")
	}

	callerFn = prev
	dir, err := migrationDir()
	if err != nil {
		t.Fatalf("migrationDir: %v", err)
	}
	if !strings.HasSuffix(filepath.ToSlash(dir), "apps/server/migrations") {
		t.Fatalf("unexpected migration dir: %s", dir)
	}
}

func TestOpenWrapperAndResetWrapperBranches(t *testing.T) {
	prevOnce, prevErr := migrateOnce, migrateErr
	migrateOnce = sync.Once{}
	migrateErr = nil
	t.Cleanup(func() {
		migrateOnce = prevOnce
		migrateErr = prevErr
	})

	prevDSN := os.Getenv("LIFEBASE_TEST_DATABASE_URL")
	t.Cleanup(func() { _ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", prevDSN) })

	ft := &fakeT{}
	_ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", "")
	if db := Open(ft); db != nil || !ft.skipped {
		t.Fatalf("expected Open skip wrapper path, got db=%v skipped=%v", db, ft.skipped)
	}

	ft = &fakeT{}
	_ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", "postgres://seongmin@localhost:1/lifebase_test?sslmode=disable")
	if db := Open(ft); db != nil || ft.fatal == "" {
		t.Fatalf("expected Open fatal wrapper path, got db=%v fatal=%q", db, ft.fatal)
	}

	dsn := strings.TrimSpace(prevDSN)
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL for reset wrapper branch")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	db.Close()

	ft = &fakeT{}
	Reset(ft, db)
	if ft.fatal == "" {
		t.Fatal("expected Reset fatal wrapper path on closed pool")
	}
}

func TestOpenAndResetWrappersWithInjectedHooks(t *testing.T) {
	prevOpenPool, prevApply, prevReset := openPoolFn, applyMigrationsFn, resetTablesFn
	prevOnce, prevErr := migrateOnce, migrateErr
	migrateOnce = sync.Once{}
	migrateErr = nil
	t.Cleanup(func() {
		openPoolFn = prevOpenPool
		applyMigrationsFn = prevApply
		resetTablesFn = prevReset
		migrateOnce = prevOnce
		migrateErr = prevErr
	})

	openPoolFn = func(context.Context, string) (*pgxpool.Pool, bool, error) {
		return &pgxpool.Pool{}, false, nil
	}
	applyMigrationsFn = func(context.Context, *pgxpool.Pool) error { return nil }

	ft := &fakeT{}
	db := Open(ft)
	if db == nil || ft.fatal != "" || ft.skipped {
		t.Fatalf("expected Open success via injected hooks, got db=%v fatal=%q skipped=%v", db, ft.fatal, ft.skipped)
	}

	resetTablesFn = func(context.Context, *pgxpool.Pool) error { return nil }
	ft = &fakeT{}
	Reset(ft, nil)
	if ft.fatal != "" {
		t.Fatalf("expected Reset success via injected hooks, got fatal=%q", ft.fatal)
	}

	resetTablesFn = func(context.Context, *pgxpool.Pool) error { return errors.New("reset boom") }
	ft = &fakeT{}
	Reset(ft, nil)
	if !strings.Contains(ft.fatal, "reset boom") {
		t.Fatalf("expected Reset fatal via injected hooks, got %q", ft.fatal)
	}
}

func TestOpenWrapperMigrationFailureBranch(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	prevOnce, prevErr := migrateOnce, migrateErr
	migrateOnce = sync.Once{}
	migrateOnce.Do(func() {})
	want := errors.New("forced migration failure")
	migrateErr = want
	t.Cleanup(func() {
		migrateOnce = prevOnce
		migrateErr = prevErr
	})

	prevDSN := os.Getenv("LIFEBASE_TEST_DATABASE_URL")
	t.Cleanup(func() { _ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", prevDSN) })
	_ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", dsn)

	ft := &fakeT{}
	db := Open(ft)
	if db != nil {
		t.Fatalf("expected nil db on migration failure, got %v", db)
	}
	if !strings.Contains(ft.fatal, want.Error()) {
		t.Fatalf("expected fatal to mention migration failure, got %q", ft.fatal)
	}
}

func TestOpenAndReset(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	db := Open(t)
	defer db.Close()
	Reset(t, db)

	_, err := db.Exec(context.Background(),
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ('u1', 'k1', 'v1', $1)`,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("insert setting before reset: %v", err)
	}
	Reset(t, db)

	var count int
	if err := db.QueryRow(context.Background(), `SELECT COUNT(*) FROM user_settings`).Scan(&count); err != nil {
		t.Fatalf("count user_settings after reset: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 rows after reset, got %d", count)
	}
}

func TestResetTablesBranches(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer db.Close()

	if err := resetTables(ctx, db); err != nil {
		t.Fatalf("resetTables should succeed: %v", err)
	}

	if _, err := db.Exec(ctx, `DROP TABLE IF EXISTS temp_reset_error`); err != nil {
		t.Fatalf("drop temp table: %v", err)
	}
	if _, err := db.Exec(ctx, `CREATE TEMP TABLE temp_reset_error(x INT)`); err != nil {
		t.Fatalf("create temp table: %v", err)
	}
	if err := resetTables(ctx, db); err != nil {
		t.Fatalf("resetTables with temp table should still succeed: %v", err)
	}

	db.Close()
	if err := resetTables(ctx, db); err == nil {
		t.Fatal("expected resetTables error on closed pool")
	}
}

func TestResetTablesRowIterationBranches(t *testing.T) {
	prevQuery, prevExec := queryTablesFn, execSQLFn
	t.Cleanup(func() {
		queryTablesFn = prevQuery
		execSQLFn = prevExec
	})

	t.Run("scan_error", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return &fakeRows{nextResults: []bool{true, false}, scanErr: errors.New("scan boom")}, nil
		}
		if err := resetTables(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "scan table name") {
			t.Fatalf("expected scan table name error, got %v", err)
		}
	})

	t.Run("rows_err", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return &fakeRows{nextResults: []bool{false}, err: errors.New("rows boom")}, nil
		}
		if err := resetTables(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "iterate table names") {
			t.Fatalf("expected iterate table names error, got %v", err)
		}
	})

	t.Run("empty_table_list", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return &fakeRows{nextResults: []bool{false}}, nil
		}
		execSQLFn = func(context.Context, *pgxpool.Pool, string) error {
			t.Fatal("exec should not be called when there are no tables")
			return nil
		}
		if err := resetTables(context.Background(), nil); err != nil {
			t.Fatalf("expected no-op when table list is empty, got %v", err)
		}
	})

	t.Run("truncate_error", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return &fakeRows{nextResults: []bool{true, false}}, nil
		}
		execSQLFn = func(context.Context, *pgxpool.Pool, string) error {
			return errors.New("truncate boom")
		}
		if err := resetTables(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "truncate tables") {
			t.Fatalf("expected truncate tables error, got %v", err)
		}
	})
}

func TestResetWrapperUnitBranches(t *testing.T) {
	prevQuery, prevExec := queryTablesFn, execSQLFn
	t.Cleanup(func() {
		queryTablesFn = prevQuery
		execSQLFn = prevExec
	})

	t.Run("success", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return &fakeRows{nextResults: []bool{false}}, nil
		}
		execSQLFn = func(context.Context, *pgxpool.Pool, string) error { return nil }

		ft := &fakeT{}
		Reset(ft, nil)
		if ft.fatal != "" {
			t.Fatalf("expected no fatal on Reset success path, got %q", ft.fatal)
		}
	})

	t.Run("error", func(t *testing.T) {
		queryTablesFn = func(context.Context, *pgxpool.Pool) (pgx.Rows, error) {
			return nil, errors.New("list fail")
		}

		ft := &fakeT{}
		Reset(ft, nil)
		if ft.fatal == "" {
			t.Fatal("expected fatal on Reset error path")
		}
	})
}

func TestApplyMigrationsUnitBranches(t *testing.T) {
	prevUsersExists := usersTableExistsFn
	prevReadDir := readDirFn
	prevReadFile := readFileFn
	prevCaller := callerFn
	prevExec := execSQLFn
	t.Cleanup(func() {
		usersTableExistsFn = prevUsersExists
		readDirFn = prevReadDir
		readFileFn = prevReadFile
		callerFn = prevCaller
		execSQLFn = prevExec
	})

	t.Run("users_exists_short_circuit", func(t *testing.T) {
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) { return true, nil }
		if err := applyMigrations(context.Background(), nil); err != nil {
			t.Fatalf("expected short-circuit success, got %v", err)
		}
	})

	t.Run("users_exists_query_error", func(t *testing.T) {
		want := errors.New("users lookup fail")
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) { return false, want }
		if err := applyMigrations(context.Background(), nil); !errors.Is(err, want) {
			t.Fatalf("expected users lookup error, got %v", err)
		}
	})

	t.Run("empty_up_sql_file", func(t *testing.T) {
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) { return false, nil }
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		filePath := filepath.Join(migrationsDir, "001_empty_up.sql")
		if err := os.WriteFile(filePath, []byte("-- +goose Up\n\n-- +goose Down\nSELECT 1;"), 0o644); err != nil {
			t.Fatalf("write migration: %v", err)
		}

		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		execCalled := false
		execSQLFn = func(context.Context, *pgxpool.Pool, string) error {
			execCalled = true
			return nil
		}
		if err := applyMigrations(context.Background(), nil); err != nil {
			t.Fatalf("expected empty-up migration success, got %v", err)
		}
		if execCalled {
			t.Fatal("expected no exec for empty up migration")
		}
	})
}

func TestOpenSkipsWithoutDatabaseURL(t *testing.T) {
	prev := os.Getenv("LIFEBASE_TEST_DATABASE_URL")
	if err := os.Setenv("LIFEBASE_TEST_DATABASE_URL", ""); err != nil {
		t.Fatalf("clear env: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Setenv("LIFEBASE_TEST_DATABASE_URL", prev)
	})

	t.Run("skip", func(t *testing.T) {
		_ = Open(t)
		t.Fatal("Open should have skipped when DB URL is empty")
	})
}

func TestApplyMigrationsWhenUsersTableExists(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL DEFAULT '',
			picture TEXT NOT NULL DEFAULT '',
			storage_quota_bytes BIGINT NOT NULL DEFAULT 0,
			storage_used_bytes BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		t.Fatalf("ensure users table: %v", err)
	}

	if err := applyMigrations(ctx, db); err != nil {
		t.Fatalf("apply migrations with existing users table: %v", err)
	}
}

func TestApplyMigrationsFromScratch(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	adminDSN, err := withDatabase(dsn, "postgres")
	if err != nil {
		t.Fatalf("admin dsn: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	defer adminPool.Close()

	dbName := fmt.Sprintf("lifebase_cov_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
		t.Fatalf("create temp db: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cleanupCancel()
		_, _ = adminPool.Exec(cleanupCtx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)
		_, _ = adminPool.Exec(cleanupCtx, `DROP DATABASE IF EXISTS `+dbName)
	})

	tempDSN, err := withDatabase(dsn, dbName)
	if err != nil {
		t.Fatalf("temp dsn: %v", err)
	}
	tempPool, err := pgxpool.New(ctx, tempDSN)
	if err != nil {
		t.Fatalf("open temp pool: %v", err)
	}
	defer tempPool.Close()

	if err := applyMigrations(ctx, tempPool); err != nil {
		t.Fatalf("apply migrations from scratch: %v", err)
	}

	var exists bool
	if err := tempPool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			  FROM information_schema.tables
			 WHERE table_schema = 'public' AND table_name = 'users'
		)`).
		Scan(&exists); err != nil {
		t.Fatalf("check users table: %v", err)
	}
	if !exists {
		t.Fatal("expected users table after migrations")
	}
}

func TestApplyMigrationsErrorBranches(t *testing.T) {
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("requires LIFEBASE_TEST_DATABASE_URL")
	}

	adminDSN, err := withDatabase(dsn, "postgres")
	if err != nil {
		t.Fatalf("admin dsn: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		t.Fatalf("open admin pool: %v", err)
	}
	defer adminPool.Close()

	makeTempDB := func(t *testing.T) *pgxpool.Pool {
		t.Helper()
		dbName := fmt.Sprintf("lifebase_cov_%d", time.Now().UnixNano())
		if _, err := adminPool.Exec(ctx, `CREATE DATABASE `+dbName); err != nil {
			t.Fatalf("create temp db: %v", err)
		}
		t.Cleanup(func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cleanupCancel()
			_, _ = adminPool.Exec(cleanupCtx, `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)
			_, _ = adminPool.Exec(cleanupCtx, `DROP DATABASE IF EXISTS `+dbName)
		})

		tempDSN, err := withDatabase(dsn, dbName)
		if err != nil {
			t.Fatalf("temp dsn: %v", err)
		}
		tempPool, err := pgxpool.New(ctx, tempDSN)
		if err != nil {
			t.Fatalf("open temp pool: %v", err)
		}
		t.Cleanup(tempPool.Close)
		return tempPool
	}

	t.Run("closed_pool_query_error", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempPool.Close()
		if err := applyMigrations(ctx, tempPool); err == nil {
			t.Fatal("expected applyMigrations error on closed pool")
		}
	})

	t.Run("read_dir_error", func(t *testing.T) {
		tempPool := makeTempDB(t)
		prevReadDir, prevCaller := readDirFn, callerFn
		t.Cleanup(func() {
			readDirFn = prevReadDir
			callerFn = prevCaller
		})
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, "/tmp/does-not-matter.go", 0, true
		}
		want := errors.New("read dir boom")
		readDirFn = func(string) ([]fs.DirEntry, error) { return nil, want }
		if err := applyMigrations(ctx, tempPool); !errors.Is(err, want) {
			t.Fatalf("expected readDir error, got %v", err)
		}
	})

	t.Run("migration_dir_error", func(t *testing.T) {
		tempPool := makeTempDB(t)
		prevCaller := callerFn
		t.Cleanup(func() { callerFn = prevCaller })
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, "", 0, false
		}
		if err := applyMigrations(ctx, tempPool); err == nil || !strings.Contains(err.Error(), "resolve migration dir") {
			t.Fatalf("expected migration dir error, got %v", err)
		}
	})

	t.Run("read_file_error", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		filePath := filepath.Join(migrationsDir, "001_test.sql")
		if err := os.WriteFile(filePath, []byte("-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 2;"), 0o644); err != nil {
			t.Fatalf("write migration: %v", err)
		}

		prevReadFile, prevCaller := readFileFn, callerFn
		t.Cleanup(func() {
			readFileFn = prevReadFile
			callerFn = prevCaller
		})
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		want := errors.New("read file boom")
		readFileFn = func(string) ([]byte, error) { return nil, want }
		if err := applyMigrations(ctx, tempPool); !errors.Is(err, want) {
			t.Fatalf("expected readFile error, got %v", err)
		}
	})

	t.Run("invalid_goose_file", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		filePath := filepath.Join(migrationsDir, "001_invalid.sql")
		if err := os.WriteFile(filePath, []byte("SELECT 1;"), 0o644); err != nil {
			t.Fatalf("write invalid migration: %v", err)
		}

		prevCaller := callerFn
		t.Cleanup(func() { callerFn = prevCaller })
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		if err := applyMigrations(ctx, tempPool); err == nil || !strings.Contains(err.Error(), "invalid goose section") {
			t.Fatalf("expected invalid goose section error, got %v", err)
		}
	})

	t.Run("exec_up_migration_error", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		filePath := filepath.Join(migrationsDir, "001_bad.sql")
		if err := os.WriteFile(filePath, []byte("-- +goose Up\nTHIS IS NOT SQL;\n-- +goose Down\nSELECT 1;"), 0o644); err != nil {
			t.Fatalf("write bad migration: %v", err)
		}

		prevCaller := callerFn
		t.Cleanup(func() { callerFn = prevCaller })
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		if err := applyMigrations(ctx, tempPool); err == nil || !strings.Contains(err.Error(), "exec up migration") {
			t.Fatalf("expected exec up migration error, got %v", err)
		}
	})

	t.Run("blank_up_sql_is_skipped", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		filePath := filepath.Join(migrationsDir, "001_blank.sql")
		if err := os.WriteFile(filePath, []byte("-- +goose Up\n-- +goose Down\nSELECT 1;"), 0o644); err != nil {
			t.Fatalf("write blank migration: %v", err)
		}

		prevCaller := callerFn
		t.Cleanup(func() { callerFn = prevCaller })
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		if err := applyMigrations(ctx, tempPool); err != nil {
			t.Fatalf("expected blank up sql to be skipped, got %v", err)
		}
	})

	t.Run("skip_dirs_and_non_sql_files", func(t *testing.T) {
		tempPool := makeTempDB(t)
		tempDir := t.TempDir()
		migrationsDir := filepath.Join(tempDir, "migrations")
		if err := os.MkdirAll(filepath.Join(migrationsDir, "subdir"), 0o755); err != nil {
			t.Fatalf("mkdir migrations subdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(migrationsDir, "README.txt"), []byte("ignore"), 0o644); err != nil {
			t.Fatalf("write readme: %v", err)
		}

		prevCaller := callerFn
		t.Cleanup(func() { callerFn = prevCaller })
		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tempDir, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		if err := applyMigrations(ctx, tempPool); err != nil {
			t.Fatalf("expected directory and non-sql files to be skipped, got %v", err)
		}
	})
}

func TestApplyMigrationsWithInjectedHooks(t *testing.T) {
	prevUsers, prevReadDir, prevReadFile, prevCaller, prevExec := usersTableExistsFn, readDirFn, readFileFn, callerFn, execSQLFn
	t.Cleanup(func() {
		usersTableExistsFn = prevUsers
		readDirFn = prevReadDir
		readFileFn = prevReadFile
		callerFn = prevCaller
		execSQLFn = prevExec
	})

	t.Run("users_table_check_error", func(t *testing.T) {
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) {
			return false, errors.New("users check boom")
		}
		if err := applyMigrations(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "users check boom") {
			t.Fatalf("expected users table check error, got %v", err)
		}
		usersTableExistsFn = prevUsers
	})

	t.Run("users_table_exists_short_circuit", func(t *testing.T) {
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) {
			return true, nil
		}
		readDirFn = func(string) ([]fs.DirEntry, error) {
			t.Fatal("readDir should not be called when users table already exists")
			return nil, nil
		}
		if err := applyMigrations(context.Background(), nil); err != nil {
			t.Fatalf("expected nil when users table already exists, got %v", err)
		}
		usersTableExistsFn = prevUsers
		readDirFn = prevReadDir
	})

	t.Run("success_executes_non_empty_up_sql", func(t *testing.T) {
		usersTableExistsFn = func(context.Context, *pgxpool.Pool) (bool, error) {
			return false, nil
		}
		tmp := t.TempDir()
		migrationsDir := filepath.Join(tmp, "migrations")
		if err := os.MkdirAll(migrationsDir, 0o755); err != nil {
			t.Fatalf("mkdir migrations: %v", err)
		}
		files := map[string]string{
			"001_first.sql": "-- +goose Up\nSELECT 1;\n-- +goose Down\nSELECT 2;",
			"002_empty.sql": "-- +goose Up\n-- +goose NO TRANSACTION\n\n-- +goose Down\nSELECT 2;",
			"003_second.sql": "-- +goose Up\nSELECT 3;\n-- +goose Down\nSELECT 4;",
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(migrationsDir, name), []byte(body), 0o644); err != nil {
				t.Fatalf("write migration %s: %v", name, err)
			}
		}

		callerFn = func(skip int) (uintptr, string, int, bool) {
			return 0, filepath.Join(tmp, "internal", "testutil", "dbtest", "dbtest.go"), 0, true
		}
		executed := make([]string, 0, 2)
		execSQLFn = func(_ context.Context, _ *pgxpool.Pool, sql string) error {
			executed = append(executed, strings.TrimSpace(sql))
			return nil
		}

		if err := applyMigrations(context.Background(), nil); err != nil {
			t.Fatalf("expected injected success, got %v", err)
		}
		if len(executed) != 2 || executed[0] != "SELECT 1;" || executed[1] != "SELECT 3;" {
			t.Fatalf("unexpected executed migrations: %#v", executed)
		}
		usersTableExistsFn = prevUsers
		callerFn = prevCaller
		execSQLFn = prevExec
	})
}

func TestExtractUpSQL(t *testing.T) {
	sqlText := `
-- +goose Up
-- +goose NO TRANSACTION
CREATE TABLE example (id INT);

-- +goose Down
DROP TABLE example;
`
	up, err := extractUpSQL(sqlText)
	if err != nil {
		t.Fatalf("extract valid up sql: %v", err)
	}
	if strings.Contains(up, "NO TRANSACTION") {
		t.Fatalf("expected marker removed, got: %q", up)
	}
	if !strings.Contains(up, "CREATE TABLE example") {
		t.Fatalf("expected up sql content, got: %q", up)
	}

	cases := []string{
		"",
		"-- +goose Up\nSELECT 1;",
		"-- +goose Down\nSELECT 1;",
		"-- +goose Down\nx\n-- +goose Up\ny",
	}
	for _, tc := range cases {
		if _, err := extractUpSQL(tc); err == nil {
			t.Fatalf("expected invalid goose section error for: %q", tc)
		}
	}
}

func withDatabase(dsn, dbName string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + dbName
	return u.String(), nil
}

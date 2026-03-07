package dbtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	migrateOnce sync.Once
	migrateErr  error
	readDirFn   = os.ReadDir
	readFileFn  = os.ReadFile
	callerFn    = runtime.Caller
	queryTablesFn = func(ctx context.Context, db *pgxpool.Pool) (pgx.Rows, error) {
		return db.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	}
	execSQLFn = func(ctx context.Context, db *pgxpool.Pool, sql string) error {
		_, err := db.Exec(ctx, sql)
		return err
	}
)

type testingT interface {
	Helper()
	Fatalf(string, ...any)
	Skip(...any)
}

// Open connects to integration test database.
// It skips the test when LIFEBASE_TEST_DATABASE_URL is not configured.
func Open(t testingT) *pgxpool.Pool {
	t.Helper()

	db, skip, err := openPool(context.Background(), strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL")))
	if skip {
		t.Skip("skip integration DB test: LIFEBASE_TEST_DATABASE_URL is empty")
		return nil
	}
	if err != nil {
		t.Fatalf("%v", err)
		return nil
	}

	migrateOnce.Do(func() {
		migrateErr = applyMigrations(context.Background(), db)
	})
	if migrateErr != nil {
		db.Close()
		t.Fatalf("apply migrations: %v", migrateErr)
		return nil
	}

	return db
}

// Reset truncates all public tables to isolate each test.
func Reset(t testingT, db *pgxpool.Pool) {
	t.Helper()
	if err := resetTables(context.Background(), db); err != nil {
		t.Fatalf("%v", err)
		return
	}
}

func openPool(ctx context.Context, dsn string) (*pgxpool.Pool, bool, error) {
	if dsn == "" {
		return nil, true, nil
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := pgxpool.New(timeoutCtx, dsn)
	if err != nil {
		return nil, false, fmt.Errorf("open test db: %w", err)
	}
	if err := db.Ping(timeoutCtx); err != nil {
		db.Close()
		return nil, false, fmt.Errorf("ping test db: %w", err)
	}
	return db, false, nil
}

func resetTables(ctx context.Context, db *pgxpool.Pool) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	rows, err := queryTablesFn(timeoutCtx, db)
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	tables := make([]string, 0, 64)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table names: %w", err)
	}
	if len(tables) == 0 {
		return nil
	}

	quoted := make([]string, 0, len(tables))
	for _, tbl := range tables {
		quoted = append(quoted, `"`+tbl+`"`)
	}
	query := "TRUNCATE TABLE " + strings.Join(quoted, ", ") + " RESTART IDENTITY CASCADE"
	if err := execSQLFn(timeoutCtx, db, query); err != nil {
		return fmt.Errorf("truncate tables: %w", err)
	}
	return nil
}

func applyMigrations(ctx context.Context, db *pgxpool.Pool) error {
	var usersTableExists bool
	if err := db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'users'
		)`,
	).Scan(&usersTableExists); err != nil {
		return err
	}
	if usersTableExists {
		return nil
	}

	migDir, err := migrationDir()
	if err != nil {
		return err
	}
	entries, err := readDirFn(migDir)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(migDir, e.Name()))
	}
	sort.Strings(files)

	for _, file := range files {
		b, err := readFileFn(file)
		if err != nil {
			return err
		}
		upSQL, err := extractUpSQL(string(b))
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		upSQL = strings.TrimSpace(upSQL)
		if upSQL == "" {
			continue
		}
		if err := execSQLFn(ctx, db, upSQL); err != nil {
			return fmt.Errorf("%s: exec up migration: %w", file, err)
		}
	}
	return nil
}

func migrationDir() (string, error) {
	_, thisFile, _, ok := callerFn(0)
	if !ok {
		return "", fmt.Errorf("resolve migration dir: runtime caller unavailable")
	}
	rootDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	return filepath.Join(rootDir, "migrations"), nil
}

func extractUpSQL(sqlText string) (string, error) {
	upIdx := strings.Index(sqlText, "-- +goose Up")
	downIdx := strings.Index(sqlText, "-- +goose Down")
	if upIdx < 0 || downIdx < 0 || downIdx <= upIdx {
		return "", fmt.Errorf("invalid goose section")
	}
	up := sqlText[upIdx+len("-- +goose Up") : downIdx]
	// Strip goose-specific NO TRANSACTION markers if present.
	up = strings.ReplaceAll(up, "-- +goose NO TRANSACTION", "")
	return up, nil
}

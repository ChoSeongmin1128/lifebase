package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/home/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestHomeRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewHomeRepo(db)
	repoImpl, ok := repo.(*homeRepo)
	if !ok {
		t.Fatalf("expected *homeRepo concrete type, got %T", repo)
	}

	userID := "11111111-1111-1111-1111-111111111111"
	otherUser := "22222222-2222-2222-2222-222222222222"
	now := time.Now().UTC().Truncate(time.Second)
	today := now.Format("2006-01-02")

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, storage_quota_bytes, storage_used_bytes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$6), ($7,$8,$9,$10,$11,$6,$6)`,
		userID, "u1@example.com", "u1", int64(1000), int64(300), now,
		otherUser, "u2@example.com", "u2", int64(1000), int64(10),
	)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, title, start_time, end_time, timezone, is_all_day, created_at, updated_at)
		 VALUES
		 ('ev1','cal1',$1,'event1',$2,$3,'UTC',false,$4,$4),
		 ('ev2','cal1',$1,'event2',$5,$6,'UTC',true,$4,$4),
		 ('ev3','cal1',$7,'other-event',$2,$3,'UTC',false,$4,$4)`,
		userID,
		now.Add(-2*time.Hour), now.Add(-time.Hour),
		now,
		now.Add(1*time.Hour), now.Add(2*time.Hour),
		otherUser,
	)
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, name, sort_order, created_at, updated_at)
		 VALUES ('list1',$1,'L1',0,$2,$2), ('list2',$3,'L2',0,$2,$2)`,
		userID, now, otherUser,
	)
	if err != nil {
		t.Fatalf("insert todo lists: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, title, due, priority, is_done, is_pinned, sort_order, created_at, updated_at)
		 VALUES
		 ('td1','list1',$1,'overdue',$2::date - INTERVAL '1 day','high',false,true,1,$3,$3),
		 ('td2','list1',$1,'today',$2::date,'normal',false,false,2,$3,$3),
		 ('td3','list1',$1,'done',$2::date,'low',true,false,3,$3,$3),
		 ('td4','list2',$4,'other-user',$2::date,'low',false,false,1,$3,$3)`,
		userID, today, now, otherUser,
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, created_at, updated_at)
		 VALUES
		 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa1',$1,NULL,'img.png','image/png',100,'u1/f1','done',$2,$2),
		 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2',$1,NULL,'video.mp4','video/mp4',200,'u1/f2','pending',$2,$2),
		 ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa3',$1,NULL,'doc.pdf','application/pdf',300,'u1/f3','done',$2,$2),
		 ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbb1',$3,NULL,'other.txt','text/plain',400,'u2/f4','done',$2,$2)`,
		userID, now, otherUser,
	)
	if err != nil {
		t.Fatalf("insert files: %v", err)
	}

	events, total, err := repo.ListEventsInRange(ctx, userID, now.Add(-24*time.Hour).Format(time.RFC3339), now.Add(24*time.Hour).Format(time.RFC3339), 10)
	if err != nil {
		t.Fatalf("list events in range: %v", err)
	}
	if total != 2 || len(events) != 2 {
		t.Fatalf("unexpected events result total=%d len=%d", total, len(events))
	}

	overdue, overdueTotal, err := repo.ListOverdueTodos(ctx, userID, today, 10)
	if err != nil {
		t.Fatalf("list overdue todos: %v", err)
	}
	if overdueTotal != 1 || len(overdue) != 1 || overdue[0].DueDate == nil {
		t.Fatalf("unexpected overdue todos: total=%d items=%#v", overdueTotal, overdue)
	}

	todayTodos, todayTotal, err := repo.ListTodayTodos(ctx, userID, today, 10)
	if err != nil {
		t.Fatalf("list today todos: %v", err)
	}
	if todayTotal != 1 || len(todayTodos) != 1 {
		t.Fatalf("unexpected today todos: total=%d len=%d", todayTotal, len(todayTodos))
	}

	if _, _, err := repoImpl.listTodosByDueScope(ctx, userID, today, 10, "!="); err == nil {
		t.Fatal("expected invalid due scope operator error")
	}

	recentFiles, fileTotal, err := repo.ListRecentFiles(ctx, userID, 10)
	if err != nil {
		t.Fatalf("list recent files: %v", err)
	}
	if fileTotal != 3 || len(recentFiles) != 3 {
		t.Fatalf("unexpected recent files result total=%d len=%d", fileTotal, len(recentFiles))
	}

	summary, err := repo.GetStorageSummary(ctx, userID)
	if err != nil {
		t.Fatalf("get storage summary: %v", err)
	}
	if summary.UsedBytes != 300 || summary.QuotaBytes != 1000 {
		t.Fatalf("unexpected storage summary: %#v", summary)
	}
	if _, err := repo.GetStorageSummary(ctx, "33333333-3333-3333-3333-333333333333"); err == nil {
		t.Fatal("expected user not found for missing user")
	}

	typeUsage, err := repo.ListStorageTypeUsage(ctx, userID)
	if err != nil {
		t.Fatalf("list storage type usage: %v", err)
	}
	if len(typeUsage) < 3 {
		t.Fatalf("expected at least image/video/document usage rows, got %#v", typeUsage)
	}
}

func TestHomeRepoErrorPaths(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewHomeRepo(db)

	db.Close()

	if _, _, err := repo.ListEventsInRange(ctx, "u1", time.Now().Add(-time.Hour).Format(time.RFC3339), time.Now().Format(time.RFC3339), 10); err == nil {
		t.Fatal("expected ListEventsInRange query error on closed pool")
	}
	if _, _, err := repo.ListOverdueTodos(ctx, "u1", time.Now().Format("2006-01-02"), 10); err == nil {
		t.Fatal("expected ListOverdueTodos query error on closed pool")
	}
	if _, _, err := repo.ListRecentFiles(ctx, "u1", 10); err == nil {
		t.Fatal("expected ListRecentFiles query error on closed pool")
	}
	if _, err := repo.GetStorageSummary(ctx, "u1"); err == nil {
		t.Fatal("expected GetStorageSummary query error on closed pool")
	}
	if _, err := repo.ListStorageTypeUsage(ctx, "u1"); err == nil {
		t.Fatal("expected ListStorageTypeUsage query error on closed pool")
	}
}

func TestHomeRepoInputValidationErrors(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewHomeRepo(db)

	if _, _, err := repo.ListEventsInRange(ctx, "u1", "bad-start", "bad-end", 10); err == nil {
		t.Fatal("expected invalid time cast error")
	}
	if _, _, err := repo.ListOverdueTodos(ctx, "u1", "bad-date", 10); err == nil {
		t.Fatal("expected invalid date cast error")
	}
	if _, _, err := repo.ListTodayTodos(ctx, "u1", "bad-date", 10); err == nil {
		t.Fatal("expected invalid date cast error")
	}
}

func TestHomeRepoListQueryAndScannerErrorBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewHomeRepo(db)
	repoImpl, ok := repo.(*homeRepo)
	if !ok {
		t.Fatalf("expected *homeRepo concrete type, got %T", repo)
	}

	const userID = "11111111-1111-1111-1111-111111111111"
	now := time.Now().UTC().Truncate(time.Second)
	today := now.Format("2006-01-02")

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, storage_quota_bytes, storage_used_bytes, created_at, updated_at)
		 VALUES ($1, 'u1@example.com', 'u1', 1000, 0, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	prevQuery := queryHomeRowsFn
	prevEventScan := scanEventSummariesFn
	prevTodoScan := scanTodoSummariesFn
	prevRecentScan := scanRecentFileSummariesFn
	t.Cleanup(func() {
		queryHomeRowsFn = prevQuery
		scanEventSummariesFn = prevEventScan
		scanTodoSummariesFn = prevTodoScan
		scanRecentFileSummariesFn = prevRecentScan
	})

	t.Run("events_query_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return nil, errors.New("events query failed")
		}
		if _, _, err := repoImpl.ListEventsInRange(ctx, userID, now.Add(-time.Hour).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339), 10); err == nil {
			t.Fatal("expected events list query error")
		}
		queryHomeRowsFn = prevQuery
	})

	t.Run("events_scan_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakeRows{}, nil
		}
		scanEventSummariesFn = func(pgx.Rows, int) ([]domain.EventSummary, error) {
			return nil, errors.New("events scan failed")
		}
		if _, _, err := repoImpl.ListEventsInRange(ctx, userID, now.Add(-time.Hour).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339), 10); err == nil {
			t.Fatal("expected events scan error")
		}
		queryHomeRowsFn = prevQuery
		scanEventSummariesFn = prevEventScan
	})

	t.Run("todos_query_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return nil, errors.New("todos query failed")
		}
		if _, _, err := repoImpl.ListOverdueTodos(ctx, userID, today, 10); err == nil {
			t.Fatal("expected todo list query error")
		}
		queryHomeRowsFn = prevQuery
	})

	t.Run("todos_scan_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakeRows{}, nil
		}
		scanTodoSummariesFn = func(pgx.Rows, int) ([]domain.TodoSummary, error) {
			return nil, errors.New("todos scan failed")
		}
		if _, _, err := repoImpl.ListTodayTodos(ctx, userID, today, 10); err == nil {
			t.Fatal("expected todo scan error")
		}
		queryHomeRowsFn = prevQuery
		scanTodoSummariesFn = prevTodoScan
	})

	t.Run("recent_query_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return nil, errors.New("recent files query failed")
		}
		if _, _, err := repoImpl.ListRecentFiles(ctx, userID, 10); err == nil {
			t.Fatal("expected recent files query error")
		}
		queryHomeRowsFn = prevQuery
	})

	t.Run("recent_scan_error", func(t *testing.T) {
		queryHomeRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakeRows{}, nil
		}
		scanRecentFileSummariesFn = func(pgx.Rows, int) ([]domain.RecentFileSummary, error) {
			return nil, errors.New("recent scan failed")
		}
		if _, _, err := repoImpl.ListRecentFiles(ctx, userID, 10); err == nil {
			t.Fatal("expected recent files scan error")
		}
		queryHomeRowsFn = prevQuery
		scanRecentFileSummariesFn = prevRecentScan
	})
}

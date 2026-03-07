package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/home/domain"
	portout "lifebase/internal/home/port/out"
)

type homeRepo struct {
	db *pgxpool.Pool
}

var (
	queryHomeRowsFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
		return db.Query(ctx, sql, args...)
	}
	scanEventSummariesFn      = scanEventSummaries
	scanTodoSummariesFn       = scanTodoSummaries
	scanRecentFileSummariesFn = scanRecentFileSummaries
)

func NewHomeRepo(db *pgxpool.Pool) portout.HomeRepository {
	return &homeRepo{db: db}
}

func (r *homeRepo) ListEventsInRange(ctx context.Context, userID string, startISO, endISO string, limit int) ([]domain.EventSummary, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM events
		 WHERE user_id = $1
		   AND deleted_at IS NULL
		   AND start_time < $3::timestamptz
		   AND end_time > $2::timestamptz`,
		userID, startISO, endISO,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := queryHomeRowsFn(ctx, r.db,
		`SELECT id, calendar_id, title, start_time, end_time, is_all_day, color_id
		 FROM events
		 WHERE user_id = $1
		   AND deleted_at IS NULL
		   AND start_time < $3::timestamptz
		   AND end_time > $2::timestamptz
		 ORDER BY start_time ASC, end_time DESC
		 LIMIT $4`,
		userID, startISO, endISO, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanEventSummariesFn(rows, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *homeRepo) ListOverdueTodos(ctx context.Context, userID, todayDate string, limit int) ([]domain.TodoSummary, int, error) {
	return r.listTodosByDueScope(ctx, userID, todayDate, limit, "<")
}

func (r *homeRepo) ListTodayTodos(ctx context.Context, userID, todayDate string, limit int) ([]domain.TodoSummary, int, error) {
	return r.listTodosByDueScope(ctx, userID, todayDate, limit, "=")
}

func (r *homeRepo) listTodosByDueScope(ctx context.Context, userID, todayDate string, limit int, op string) ([]domain.TodoSummary, int, error) {
	if op != "<" && op != "=" {
		return nil, 0, fmt.Errorf("invalid due scope operator")
	}

	countQuery := fmt.Sprintf(
		`SELECT COUNT(*)
		 FROM todos
		 WHERE user_id = $1
		   AND deleted_at IS NULL
		   AND is_done = FALSE
		   AND due IS NOT NULL
		   AND due %s $2::date`,
		op,
	)
	listQuery := fmt.Sprintf(
		`SELECT id, list_id, title, due, priority, is_pinned
		 FROM todos
		 WHERE user_id = $1
		   AND deleted_at IS NULL
		   AND is_done = FALSE
		   AND due IS NOT NULL
		   AND due %s $2::date
		 ORDER BY due ASC, is_pinned DESC, sort_order ASC, created_at ASC
		 LIMIT $3`,
		op,
	)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, userID, todayDate).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := queryHomeRowsFn(ctx, r.db, listQuery, userID, todayDate, limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanTodoSummariesFn(rows, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *homeRepo) ListRecentFiles(ctx context.Context, userID string, limit int) ([]domain.RecentFileSummary, int, error) {
	var total int
	if err := r.db.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM files
		 WHERE user_id = $1
		   AND deleted_at IS NULL`,
		userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := queryHomeRowsFn(ctx, r.db,
		`SELECT id, folder_id, name, mime_type, size_bytes, thumb_status, updated_at
		 FROM files
		 WHERE user_id = $1
		   AND deleted_at IS NULL
		 ORDER BY updated_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanRecentFileSummariesFn(rows, limit)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *homeRepo) GetStorageSummary(ctx context.Context, userID string) (domain.StorageSummary, error) {
	out := domain.StorageSummary{}
	if err := r.db.QueryRow(ctx,
		`SELECT storage_used_bytes, storage_quota_bytes
		 FROM users
		 WHERE id = $1`,
		userID,
	).Scan(&out.UsedBytes, &out.QuotaBytes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.StorageSummary{}, fmt.Errorf("user not found")
		}
		return domain.StorageSummary{}, err
	}
	return out, nil
}

func (r *homeRepo) ListStorageTypeUsage(ctx context.Context, userID string) ([]domain.StorageTypeUsage, error) {
	rows, err := queryHomeRowsFn(ctx, r.db,
		`SELECT type_key, COALESCE(SUM(size_bytes), 0) AS used_bytes
		 FROM (
		   SELECT
		     CASE
		       WHEN lower(name) ~ '\.(svg|png|jpe?g|gif|webp|bmp|avif|heic|heif)$' THEN 'image'
		       WHEN lower(name) ~ '\.(mp4|mov|m4v|webm|avi|mkv|wmv)$' THEN 'video'
		       WHEN lower(name) ~ '\.(pdf|docx?|xlsx?|pptx?|txt|md|csv|json|xml|rtf)$' THEN 'document'
		       WHEN mime_type LIKE 'image/%' THEN 'image'
		       WHEN mime_type LIKE 'video/%' THEN 'video'
		       WHEN mime_type LIKE 'text/%'
		         OR mime_type IN (
		           'application/pdf',
		           'application/msword',
		           'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
		           'application/vnd.ms-excel',
		           'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
		           'application/vnd.ms-powerpoint',
		           'application/vnd.openxmlformats-officedocument.presentationml.presentation',
		           'application/rtf',
		           'application/json',
		           'application/xml',
		           'text/markdown'
		         ) THEN 'document'
		       ELSE 'other'
		     END AS type_key,
		     size_bytes
		   FROM files
		   WHERE user_id = $1
		     AND deleted_at IS NULL
		 ) t
		 GROUP BY type_key`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanStorageTypeUsage(rows)
}

func scanEventSummaries(rows pgx.Rows, limit int) ([]domain.EventSummary, error) {
	items := make([]domain.EventSummary, 0, limit)
	for rows.Next() {
		var item domain.EventSummary
		if err := rows.Scan(
			&item.ID,
			&item.CalendarID,
			&item.Title,
			&item.StartTime,
			&item.EndTime,
			&item.IsAllDay,
			&item.ColorID,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanTodoSummaries(rows pgx.Rows, limit int) ([]domain.TodoSummary, error) {
	items := make([]domain.TodoSummary, 0, limit)
	for rows.Next() {
		var item domain.TodoSummary
		var dueDate *time.Time
		if err := rows.Scan(
			&item.ID,
			&item.ListID,
			&item.Title,
			&dueDate,
			&item.Priority,
			&item.IsPinned,
		); err != nil {
			return nil, err
		}
		if dueDate != nil {
			s := dueDate.Format("2006-01-02")
			item.DueDate = &s
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanRecentFileSummaries(rows pgx.Rows, limit int) ([]domain.RecentFileSummary, error) {
	items := make([]domain.RecentFileSummary, 0, limit)
	for rows.Next() {
		var item domain.RecentFileSummary
		if err := rows.Scan(
			&item.ID,
			&item.FolderID,
			&item.Name,
			&item.MimeType,
			&item.SizeBytes,
			&item.ThumbStatus,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func scanStorageTypeUsage(rows pgx.Rows) ([]domain.StorageTypeUsage, error) {
	out := make([]domain.StorageTypeUsage, 0, 3)
	for rows.Next() {
		var item domain.StorageTypeUsage
		if err := rows.Scan(&item.Type, &item.Bytes); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

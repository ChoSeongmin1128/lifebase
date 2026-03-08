package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	calendarportout "lifebase/internal/calendar/port/out"
)

type daySummaryHolidayRepo struct {
	db *pgxpool.Pool
}

var (
	scanDaySummaryHolidayRowsFn = scanDaySummaryHolidayRows
	scanDaySummaryTodoRowsFn    = scanDaySummaryTodoRows
	queryDaySummaryRowsFn       = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
		return db.Query(ctx, sql, args...)
	}
)

func NewDaySummaryHolidayRepo(db *pgxpool.Pool) *daySummaryHolidayRepo {
	return &daySummaryHolidayRepo{db: db}
}

func (r *daySummaryHolidayRepo) ListByDateRange(
	ctx context.Context,
	start,
	end time.Time,
) ([]calendarportout.DaySummaryHoliday, error) {
	rows, err := queryDaySummaryRowsFn(ctx, r.db,
		`SELECT locdate, name
		   FROM public_holidays_kr
		  WHERE locdate >= $1::date
		    AND locdate <= $2::date
		  ORDER BY locdate ASC, name ASC`,
		start,
		end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDaySummaryHolidayRowsFn(rows)
}

type daySummaryTodoRepo struct {
	db *pgxpool.Pool
}

func NewDaySummaryTodoRepo(db *pgxpool.Pool) *daySummaryTodoRepo {
	return &daySummaryTodoRepo{db: db}
}

func (r *daySummaryTodoRepo) ListByDueDate(
	ctx context.Context,
	userID,
	date string,
	includeDone bool,
) ([]calendarportout.DaySummaryTodo, error) {
	query := `SELECT id, list_id, title,
	                 CASE WHEN due_date IS NULL THEN NULL ELSE to_char(due_date, 'YYYY-MM-DD') END AS due_date,
	                 CASE WHEN due_time IS NULL THEN NULL ELSE to_char(due_time, 'HH24:MI') END AS due_time,
	                 is_done
	            FROM todos
	           WHERE user_id = $1
	             AND deleted_at IS NULL
	             AND due_date = $2::date`
	if !includeDone {
		query += " AND is_done = FALSE"
	}
	query += ` ORDER BY
	             due_time IS NULL ASC,
	             due_time ASC,
	             is_done ASC,
	             sort_order ASC,
	             created_at ASC`

	rows, err := queryDaySummaryRowsFn(ctx, r.db, query, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDaySummaryTodoRowsFn(rows)
}

var _ calendarportout.DaySummaryHolidayRepository = (*daySummaryHolidayRepo)(nil)
var _ calendarportout.DaySummaryTodoRepository = (*daySummaryTodoRepo)(nil)

func scanDaySummaryHolidayRows(rows pgx.Rows) ([]calendarportout.DaySummaryHoliday, error) {
	var holidays []calendarportout.DaySummaryHoliday
	for rows.Next() {
		var item calendarportout.DaySummaryHoliday
		if err := rows.Scan(&item.Date, &item.Name); err != nil {
			return nil, err
		}
		holidays = append(holidays, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return holidays, nil
}

func scanDaySummaryTodoRows(rows pgx.Rows) ([]calendarportout.DaySummaryTodo, error) {
	var todos []calendarportout.DaySummaryTodo
	for rows.Next() {
		var item calendarportout.DaySummaryTodo
		if err := rows.Scan(
			&item.ID,
			&item.ListID,
			&item.Title,
			&item.DueDate,
			&item.DueTime,
			&item.IsDone,
		); err != nil {
			return nil, err
		}
		todos = append(todos, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}

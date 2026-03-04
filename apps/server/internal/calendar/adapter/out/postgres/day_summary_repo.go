package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	calendarportout "lifebase/internal/calendar/port/out"
)

type daySummaryHolidayRepo struct {
	db *pgxpool.Pool
}

func NewDaySummaryHolidayRepo(db *pgxpool.Pool) *daySummaryHolidayRepo {
	return &daySummaryHolidayRepo{db: db}
}

func (r *daySummaryHolidayRepo) ListByDateRange(
	ctx context.Context,
	start,
	end time.Time,
) ([]calendarportout.DaySummaryHoliday, error) {
	rows, err := r.db.Query(ctx,
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

	var holidays []calendarportout.DaySummaryHoliday
	for rows.Next() {
		var item calendarportout.DaySummaryHoliday
		if err := rows.Scan(&item.Date, &item.Name); err != nil {
			return nil, err
		}
		holidays = append(holidays, item)
	}
	return holidays, nil
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
	                 CASE WHEN due IS NULL THEN NULL ELSE to_char(due, 'YYYY-MM-DD') END AS due,
	                 priority, is_done
	            FROM todos
	           WHERE user_id = $1
	             AND deleted_at IS NULL
	             AND due = $2::date`
	if !includeDone {
		query += " AND is_done = FALSE"
	}
	query += ` ORDER BY
	             CASE priority
	               WHEN 'urgent' THEN 0
	               WHEN 'high' THEN 1
	               WHEN 'normal' THEN 2
	               WHEN 'low' THEN 3
	               ELSE 4
	             END ASC,
	             is_done ASC,
	             sort_order ASC,
	             created_at ASC`

	rows, err := r.db.Query(ctx, query, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []calendarportout.DaySummaryTodo
	for rows.Next() {
		var item calendarportout.DaySummaryTodo
		if err := rows.Scan(
			&item.ID,
			&item.ListID,
			&item.Title,
			&item.Due,
			&item.Priority,
			&item.IsDone,
		); err != nil {
			return nil, err
		}
		todos = append(todos, item)
	}
	return todos, nil
}

var _ calendarportout.DaySummaryHolidayRepository = (*daySummaryHolidayRepo)(nil)
var _ calendarportout.DaySummaryTodoRepository = (*daySummaryTodoRepo)(nil)

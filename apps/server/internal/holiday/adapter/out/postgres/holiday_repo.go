package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/holiday/domain"
	portout "lifebase/internal/holiday/port/out"
)

type holidayRepo struct {
	db *pgxpool.Pool
}

var scanHolidayRowsFn = scanHolidayRows
var queryHolidayRowsFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
	return db.Query(ctx, sql, args...)
}
var acquireHolidayConnFn = func(ctx context.Context, db *pgxpool.Pool) (*pgxpool.Conn, error) {
	return db.Acquire(ctx)
}

var queryAdvisoryLock = func(ctx context.Context, conn *pgxpool.Conn, lockKey int64) (bool, error) {
	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		return false, err
	}
	return locked, nil
}

type holidayTx interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

func NewHolidayRepo(db *pgxpool.Pool) *holidayRepo {
	return &holidayRepo{db: db}
}

func (r *holidayRepo) ListByDateRange(ctx context.Context, start, end time.Time) ([]domain.Holiday, error) {
	rows, err := queryHolidayRowsFn(ctx, r.db,
		`SELECT locdate, name, year, month, date_kind, is_holiday, fetched_at
		   FROM public_holidays_kr
		  WHERE locdate BETWEEN $1 AND $2
		  ORDER BY locdate ASC, name ASC`,
		start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHolidayRowsFn(rows)
}

func (r *holidayRepo) GetMonthSyncState(ctx context.Context, year, month int) (*domain.MonthSyncState, error) {
	var state domain.MonthSyncState
	err := r.db.QueryRow(ctx,
		`SELECT year, month, last_synced_at, COALESCE(last_result_code, '')
		   FROM public_holiday_sync_state
		  WHERE year = $1 AND month = $2`,
		year, month,
	).Scan(&state.Year, &state.Month, &state.LastSyncedAt, &state.ResultCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, err
	}
	return &state, nil
}

func (r *holidayRepo) ReplaceMonth(
	ctx context.Context,
	year, month int,
	holidays []domain.Holiday,
	fetchedAt time.Time,
	resultCode string,
) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	return replaceMonthTx(ctx, tx, year, month, holidays, fetchedAt, resultCode)
}

func replaceMonthTx(
	ctx context.Context,
	tx holidayTx,
	year, month int,
	holidays []domain.Holiday,
	fetchedAt time.Time,
	resultCode string,
) error {

	if _, err := tx.Exec(ctx,
		`DELETE FROM public_holidays_kr WHERE year = $1 AND month = $2`,
		year, month,
	); err != nil {
		return err
	}

	for _, holiday := range holidays {
		if _, err := tx.Exec(ctx,
			`INSERT INTO public_holidays_kr
			   (locdate, name, year, month, date_kind, is_holiday, fetched_at, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
			holiday.Date,
			holiday.Name,
			year,
			month,
			holiday.DateKind,
			holiday.IsHoliday,
			fetchedAt,
			fetchedAt,
		); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO public_holiday_sync_state
		   (year, month, last_synced_at, last_result_code, updated_at)
		 VALUES ($1, $2, $3, $4, $3)
		 ON CONFLICT (year, month)
		 DO UPDATE SET
		   last_synced_at = EXCLUDED.last_synced_at,
		   last_result_code = EXCLUDED.last_result_code,
		   updated_at = EXCLUDED.updated_at`,
		year,
		month,
		fetchedAt,
		resultCode,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *holidayRepo) TryAdvisoryMonthLock(ctx context.Context, year, month int) (bool, portout.UnlockFunc, error) {
	conn, err := acquireHolidayConnFn(ctx, r.db)
	if err != nil {
		return false, nil, err
	}

	lockKey := advisoryMonthLockKey(year, month)
	locked, err := queryAdvisoryLock(ctx, conn, lockKey)
	if err != nil {
		conn.Release()
		return false, nil, err
	}
	if !locked {
		conn.Release()
		return false, nil, nil
	}

	unlock := func() {
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)
		conn.Release()
	}
	return true, unlock, nil
}

func advisoryMonthLockKey(year, month int) int64 {
	return int64(91000000 + (year * 100) + month)
}

func scanHolidayRows(rows pgx.Rows) ([]domain.Holiday, error) {
	items := make([]domain.Holiday, 0, 32)
	for rows.Next() {
		var item domain.Holiday
		if err := rows.Scan(
			&item.Date,
			&item.Name,
			&item.Year,
			&item.Month,
			&item.DateKind,
			&item.IsHoliday,
			&item.FetchedAt,
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

var _ portout.HolidayCacheRepository = (*holidayRepo)(nil)

func validateYearMonth(year, month int) error {
	if year < 1900 || year > 2200 {
		return fmt.Errorf("invalid year")
	}
	if month < 1 || month > 12 {
		return fmt.Errorf("invalid month")
	}
	return nil
}

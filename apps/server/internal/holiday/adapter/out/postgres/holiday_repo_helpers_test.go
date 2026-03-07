package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/holiday/domain"
	"lifebase/internal/testutil/dbtest"
)

type fakeHolidayRows struct {
	data    [][]any
	i       int
	scanErr error
	err     error
}

func (r *fakeHolidayRows) Close()                                       {}
func (r *fakeHolidayRows) Err() error                                   { return r.err }
func (r *fakeHolidayRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeHolidayRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeHolidayRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeHolidayRows) RawValues() [][]byte                          { return nil }
func (r *fakeHolidayRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeHolidayRows) Next() bool {
	if r.i >= len(r.data) {
		return false
	}
	r.i++
	return true
}

func (r *fakeHolidayRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.i-1]
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if dv.Kind() != reflect.Ptr || dv.IsNil() {
			return errors.New("dest must be non-nil pointer")
		}
		if i >= len(row) || row[i] == nil {
			dv.Elem().Set(reflect.Zero(dv.Elem().Type()))
			continue
		}
		dv.Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}

func TestScanHolidayRowsBranches(t *testing.T) {
	now := time.Now().UTC()
	items, err := scanHolidayRows(&fakeHolidayRows{data: [][]any{
		{now, "삼일절", 2026, 3, "01", true, now},
	}})
	if err != nil || len(items) != 1 || items[0].Name != "삼일절" {
		t.Fatalf("scan holiday rows failed: len=%d err=%v", len(items), err)
	}
	if _, err := scanHolidayRows(&fakeHolidayRows{data: [][]any{{now}}, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scan error")
	}
	if _, err := scanHolidayRows(&fakeHolidayRows{err: errors.New("rows err")}); err == nil {
		t.Fatal("expected rows.Err error")
	}
}

type fakeHolidayTx struct {
	execErrAt int
	commitErr error
	execCalls int
}

func (f *fakeHolidayTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	f.execCalls++
	if f.execErrAt > 0 && f.execCalls == f.execErrAt {
		return pgconn.CommandTag{}, errors.New("exec fail")
	}
	return pgconn.CommandTag{}, nil
}
func (f *fakeHolidayTx) Commit(context.Context) error   { return f.commitErr }
func (f *fakeHolidayTx) Rollback(context.Context) error { return nil }

func TestReplaceMonthTxBranches(t *testing.T) {
	now := time.Now().UTC()
	holidays := []domain.Holiday{
		{Date: now, Name: "A", DateKind: "01", IsHoliday: true},
	}

	if err := replaceMonthTx(context.Background(), &fakeHolidayTx{execErrAt: 1}, 2026, 3, holidays, now, "00"); err == nil {
		t.Fatal("expected delete exec error")
	}
	if err := replaceMonthTx(context.Background(), &fakeHolidayTx{execErrAt: 2}, 2026, 3, holidays, now, "00"); err == nil {
		t.Fatal("expected insert exec error")
	}
	if err := replaceMonthTx(context.Background(), &fakeHolidayTx{execErrAt: 3}, 2026, 3, holidays, now, "00"); err == nil {
		t.Fatal("expected sync-state upsert error")
	}
	if err := replaceMonthTx(context.Background(), &fakeHolidayTx{commitErr: errors.New("commit fail")}, 2026, 3, holidays, now, "00"); err == nil {
		t.Fatal("expected commit error")
	}
	if err := replaceMonthTx(context.Background(), &fakeHolidayTx{}, 2026, 3, holidays, now, "00"); err != nil {
		t.Fatalf("replaceMonthTx success failed: %v", err)
	}
}

func TestTryAdvisoryMonthLockQueryErrorBranch(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	repo := NewHolidayRepo(db)

	prev := queryAdvisoryLock
	queryAdvisoryLock = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
		return false, errors.New("query fail")
	}
	t.Cleanup(func() { queryAdvisoryLock = prev })

	if _, _, err := repo.TryAdvisoryMonthLock(context.Background(), 2026, 7); err == nil {
		t.Fatal("expected advisory lock query error")
	}
}

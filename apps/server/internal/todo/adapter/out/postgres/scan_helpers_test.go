package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeScanRows struct {
	nextTotal int
	nextSeen  int
	scanErr   error
	rowsErr   error
}

func (r *fakeScanRows) Close() {}
func (r *fakeScanRows) Err() error { return r.rowsErr }
func (r *fakeScanRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeScanRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeScanRows) Next() bool {
	r.nextSeen++
	return r.nextSeen <= r.nextTotal
}
func (r *fakeScanRows) Scan(dest ...any) error { return r.scanErr }
func (r *fakeScanRows) Values() ([]any, error) { return nil, nil }
func (r *fakeScanRows) RawValues() [][]byte { return nil }
func (r *fakeScanRows) Conn() *pgx.Conn { return nil }

func TestScanTodoRowsHelpersErrors(t *testing.T) {
	if _, err := scanTodoListRows(&fakeScanRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanTodoListRows scan error")
	}
	if _, err := scanTodoListRows(&fakeScanRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanTodoListRows rows error")
	}
	if _, err := scanTodosRows(&fakeScanRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanTodosRows scan error")
	}
	if _, err := scanTodosRows(&fakeScanRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanTodosRows rows error")
	}
}

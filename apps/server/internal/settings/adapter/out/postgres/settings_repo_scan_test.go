package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRows struct {
	nextTotal int
	nextSeen  int
	scanErr   error
	rowsErr   error
}

func (r *fakeRows) Close() {}
func (r *fakeRows) Err() error { return r.rowsErr }
func (r *fakeRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Next() bool {
	r.nextSeen++
	return r.nextSeen <= r.nextTotal
}
func (r *fakeRows) Scan(dest ...any) error { return r.scanErr }
func (r *fakeRows) Values() ([]any, error) { return nil, nil }
func (r *fakeRows) RawValues() [][]byte { return nil }
func (r *fakeRows) Conn() *pgx.Conn { return nil }

func TestScanSettingsRowsErrors(t *testing.T) {
	if _, err := scanSettingsRows(&fakeRows{nextTotal: 1, scanErr: errors.New("scan fail")}, "u1"); err == nil {
		t.Fatal("expected scanSettingsRows to return scan error")
	}
	if _, err := scanSettingsRows(&fakeRows{rowsErr: errors.New("rows fail")}, "u1"); err == nil {
		t.Fatal("expected scanSettingsRows to return rows error")
	}
}

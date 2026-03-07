package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeAdminRows struct {
	nextTotal int
	nextSeen  int
	scanErr   error
	rowsErr   error
}

func (r *fakeAdminRows) Close() {}
func (r *fakeAdminRows) Err() error { return r.rowsErr }
func (r *fakeAdminRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeAdminRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeAdminRows) Next() bool {
	r.nextSeen++
	return r.nextSeen <= r.nextTotal
}
func (r *fakeAdminRows) Scan(dest ...any) error { return r.scanErr }
func (r *fakeAdminRows) Values() ([]any, error) { return nil, nil }
func (r *fakeAdminRows) RawValues() [][]byte { return nil }
func (r *fakeAdminRows) Conn() *pgx.Conn { return nil }

func TestAdminScanHelpersErrors(t *testing.T) {
	if _, err := scanAdminRows(&fakeAdminRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanAdminRows scan error")
	}
	if _, err := scanAdminRows(&fakeAdminRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanAdminRows rows error")
	}
	if _, err := scanGoogleAccountRows(&fakeAdminRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanGoogleAccountRows scan error")
	}
	if _, err := scanGoogleAccountRows(&fakeAdminRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanGoogleAccountRows rows error")
	}
	if _, err := scanFileRefsRows(&fakeAdminRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanFileRefsRows scan error")
	}
	if _, err := scanFileRefsRows(&fakeAdminRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanFileRefsRows rows error")
	}
}

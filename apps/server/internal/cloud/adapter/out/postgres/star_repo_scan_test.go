package postgres

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeStarRows struct {
	data    [][]any
	i       int
	scanErr error
	err     error
}

func (r *fakeStarRows) Close()                                       {}
func (r *fakeStarRows) Err() error                                   { return r.err }
func (r *fakeStarRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeStarRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeStarRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeStarRows) RawValues() [][]byte                          { return nil }
func (r *fakeStarRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeStarRows) Next() bool {
	if r.i >= len(r.data) {
		return false
	}
	r.i++
	return true
}

func (r *fakeStarRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	row := r.data[r.i-1]
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if i >= len(row) || row[i] == nil {
			dv.Elem().Set(reflect.Zero(dv.Elem().Type()))
			continue
		}
		dv.Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}

func TestScanStarRowsBranches(t *testing.T) {
	refs, err := scanStarRows(&fakeStarRows{data: [][]any{
		{"cloud_star:file:file-1"},
		{"cloud_star:invalid"},
	}})
	if err != nil {
		t.Fatalf("scan star rows: %v", err)
	}
	if len(refs) != 1 || refs[0].ItemID != "file-1" {
		t.Fatalf("unexpected refs: %#v", refs)
	}

	if _, err := scanStarRows(&fakeStarRows{data: [][]any{{"x"}}, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected star scan error")
	}
	if _, err := scanStarRows(&fakeStarRows{err: errors.New("rows fail")}); err == nil {
		t.Fatal("expected star rows.Err")
	}
}

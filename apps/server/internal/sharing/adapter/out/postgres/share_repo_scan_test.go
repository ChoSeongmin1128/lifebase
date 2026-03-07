package postgres

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeRows struct {
	data    [][]any
	i       int
	scanErr error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return nil }
func (r *fakeRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeRows) RawValues() [][]byte                          { return nil }
func (r *fakeRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeRows) Next() bool {
	if r.i >= len(r.data) {
		return false
	}
	r.i++
	return true
}

func (r *fakeRows) Scan(dest ...any) error {
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

func TestScanSharesSuccessAndEmpty(t *testing.T) {
	now := time.Now().UTC()
	row := []any{"s1", "f1", "o1", "u1", "viewer", now, now}
	items, err := scanShares(&fakeRows{data: [][]any{row}})
	if err != nil {
		t.Fatalf("scan shares success: %v", err)
	}
	if len(items) != 1 || items[0].ID != "s1" {
		t.Fatalf("unexpected shares result: %#v", items)
	}

	empty, err := scanShares(&fakeRows{})
	if err != nil {
		t.Fatalf("scan empty shares: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty shares, got %#v", empty)
	}
}

func TestScanSharesScanError(t *testing.T) {
	_, err := scanShares(&fakeRows{
		data:    [][]any{{"s1"}},
		scanErr: errors.New("scan fail"),
	})
	if err == nil {
		t.Fatal("expected scan error")
	}
}

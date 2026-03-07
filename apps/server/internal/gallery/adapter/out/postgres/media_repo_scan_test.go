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
	err     error
}

func (r *fakeRows) Close()                                       {}
func (r *fakeRows) Err() error                                   { return r.err }
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

func TestScanFilesSuccessAndEmpty(t *testing.T) {
	now := time.Now().UTC()
	folderID := "f-1"
	thumb := "done"
	name := "a.png"
	row := []any{
		"id-1", "u-1", &folderID, name, "image/png", int64(10),
		"u-1/id-1", thumb, &now, now, now, (*time.Time)(nil),
	}
	items, err := scanFiles(&fakeRows{data: [][]any{row}})
	if err != nil {
		t.Fatalf("scan files success: %v", err)
	}
	if len(items) != 1 || items[0].ID != "id-1" {
		t.Fatalf("unexpected scan result: %#v", items)
	}

	empty, err := scanFiles(&fakeRows{})
	if err != nil {
		t.Fatalf("scan empty rows: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected empty result, got %#v", empty)
	}
}

func TestScanFilesScanError(t *testing.T) {
	_, err := scanFiles(&fakeRows{
		data:    [][]any{{"id-1"}},
		scanErr: errors.New("scan fail"),
	})
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestScanFilesRowsError(t *testing.T) {
	_, err := scanFiles(&fakeRows{err: errors.New("rows fail")})
	if err == nil {
		t.Fatal("expected rows error")
	}
}

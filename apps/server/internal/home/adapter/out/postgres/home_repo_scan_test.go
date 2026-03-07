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

func TestScanEventSummariesBranches(t *testing.T) {
	now := time.Now().UTC()
	colorID := "1"
	files, err := scanEventSummaries(&fakeRows{data: [][]any{
		{"e1", "c1", "title", now, now.Add(time.Hour), false, &colorID},
	}}, 10)
	if err != nil || len(files) != 1 || files[0].ID != "e1" {
		t.Fatalf("scan event summaries failed: len=%d err=%v", len(files), err)
	}
	if _, err := scanEventSummaries(&fakeRows{data: [][]any{{"e1"}}, scanErr: errors.New("scan fail")}, 10); err == nil {
		t.Fatal("expected scan error")
	}
	if _, err := scanEventSummaries(&fakeRows{err: errors.New("rows err")}, 10); err == nil {
		t.Fatal("expected rows.Err error")
	}
}

func TestScanTodoSummariesBranches(t *testing.T) {
	now := time.Now().UTC()
	items, err := scanTodoSummaries(&fakeRows{data: [][]any{
		{"t1", "l1", "todo", &now, "high", true},
		{"t2", "l1", "todo2", (*time.Time)(nil), "low", false},
	}}, 10)
	if err != nil || len(items) != 2 {
		t.Fatalf("scan todo summaries failed: len=%d err=%v", len(items), err)
	}
	if items[0].DueDate == nil || *items[0].DueDate == "" {
		t.Fatal("expected due date formatting")
	}
	if items[1].DueDate != nil {
		t.Fatal("expected nil due date")
	}
	if _, err := scanTodoSummaries(&fakeRows{data: [][]any{{"t1"}}, scanErr: errors.New("scan fail")}, 10); err == nil {
		t.Fatal("expected scan error")
	}
	if _, err := scanTodoSummaries(&fakeRows{err: errors.New("rows err")}, 10); err == nil {
		t.Fatal("expected rows.Err error")
	}
}

func TestScanRecentAndStorageUsageBranches(t *testing.T) {
	now := time.Now().UTC()
	folderID := "folder1"
	recent, err := scanRecentFileSummaries(&fakeRows{data: [][]any{
		{"f1", &folderID, "a.txt", "text/plain", int64(1), "done", now},
	}}, 10)
	if err != nil || len(recent) != 1 || recent[0].ID != "f1" {
		t.Fatalf("scan recent files failed: len=%d err=%v", len(recent), err)
	}
	if _, err := scanRecentFileSummaries(&fakeRows{data: [][]any{{"f1"}}, scanErr: errors.New("scan fail")}, 10); err == nil {
		t.Fatal("expected recent file scan error")
	}
	if _, err := scanRecentFileSummaries(&fakeRows{err: errors.New("rows err")}, 10); err == nil {
		t.Fatal("expected recent file rows.Err error")
	}

	usage, err := scanStorageTypeUsage(&fakeRows{data: [][]any{
		{"image", int64(100)},
	}})
	if err != nil || len(usage) != 1 || usage[0].Type != "image" {
		t.Fatalf("scan usage failed: len=%d err=%v", len(usage), err)
	}
	if _, err := scanStorageTypeUsage(&fakeRows{data: [][]any{{"image"}}, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected storage usage scan error")
	}
	if _, err := scanStorageTypeUsage(&fakeRows{err: errors.New("rows err")}); err == nil {
		t.Fatal("expected storage usage rows.Err error")
	}
}

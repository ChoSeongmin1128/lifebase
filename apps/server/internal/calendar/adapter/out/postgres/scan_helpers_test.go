package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeCalendarRows struct {
	nextTotal int
	nextSeen  int
	scanErr   error
	rowsErr   error
}

func (r *fakeCalendarRows) Close() {}
func (r *fakeCalendarRows) Err() error { return r.rowsErr }
func (r *fakeCalendarRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeCalendarRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeCalendarRows) Next() bool {
	r.nextSeen++
	return r.nextSeen <= r.nextTotal
}
func (r *fakeCalendarRows) Scan(dest ...any) error { return r.scanErr }
func (r *fakeCalendarRows) Values() ([]any, error) { return nil, nil }
func (r *fakeCalendarRows) RawValues() [][]byte { return nil }
func (r *fakeCalendarRows) Conn() *pgx.Conn { return nil }

func TestCalendarScanHelpersErrors(t *testing.T) {
	if _, err := scanEventRows(&fakeCalendarRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanEventRows scan error")
	}
	if _, err := scanEventRows(&fakeCalendarRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanEventRows rows error")
	}
	if _, err := scanCalendarRows(&fakeCalendarRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanCalendarRows scan error")
	}
	if _, err := scanCalendarRows(&fakeCalendarRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanCalendarRows rows error")
	}
	if _, err := scanReminderRows(&fakeCalendarRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanReminderRows scan error")
	}
	if _, err := scanReminderRows(&fakeCalendarRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanReminderRows rows error")
	}
	if _, err := scanDaySummaryHolidayRows(&fakeCalendarRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanDaySummaryHolidayRows scan error")
	}
	if _, err := scanDaySummaryHolidayRows(&fakeCalendarRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanDaySummaryHolidayRows rows error")
	}
	if _, err := scanDaySummaryTodoRows(&fakeCalendarRows{nextTotal: 1, scanErr: errors.New("scan fail")}); err == nil {
		t.Fatal("expected scanDaySummaryTodoRows scan error")
	}
	if _, err := scanDaySummaryTodoRows(&fakeCalendarRows{rowsErr: errors.New("rows fail")}); err == nil {
		t.Fatal("expected scanDaySummaryTodoRows rows error")
	}
}

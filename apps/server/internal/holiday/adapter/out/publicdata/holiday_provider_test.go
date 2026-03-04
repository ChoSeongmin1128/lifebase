package publicdata

import (
	"encoding/json"
	"testing"
)

func TestParseHolidayItems_Array(t *testing.T) {
	raw := json.RawMessage(`[
	  {"dateKind":"01","dateName":"삼일절","isHoliday":"Y","locdate":20260301},
	  {"dateKind":"01","dateName":"평일","isHoliday":"N","locdate":20260303}
	]`)
	items, err := parseHolidayItems(raw, 2026, 3)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Name != "삼일절" {
		t.Fatalf("unexpected holiday name: %s", items[0].Name)
	}
}

func TestParseHolidayItems_SingleObject(t *testing.T) {
	raw := json.RawMessage(`{"dateKind":"01","dateName":"광복절","isHoliday":"Y","locdate":"20260815"}`)
	items, err := parseHolidayItems(raw, 2026, 8)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("unexpected item count: %d", len(items))
	}
	if items[0].Name != "광복절" {
		t.Fatalf("unexpected holiday name: %s", items[0].Name)
	}
}

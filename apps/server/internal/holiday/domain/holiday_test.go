package domain

import (
	"testing"
	"time"
)

func TestMonthKeysBetween(t *testing.T) {
	start := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	months := MonthKeysBetween(start, end)
	if len(months) != 3 {
		t.Fatalf("unexpected month count: %d", len(months))
	}
	if months[0].Year != 2026 || months[0].Month != 1 {
		t.Fatalf("unexpected first month: %+v", months[0])
	}
	if months[2].Year != 2026 || months[2].Month != 3 {
		t.Fatalf("unexpected last month: %+v", months[2])
	}
}

func TestMonthKeysInYearRange(t *testing.T) {
	months := MonthKeysInYearRange(2025, 2026)
	if len(months) != 24 {
		t.Fatalf("unexpected month count: %d", len(months))
	}
	if months[0].Year != 2025 || months[0].Month != 1 {
		t.Fatalf("unexpected first month: %+v", months[0])
	}
	if months[len(months)-1].Year != 2026 || months[len(months)-1].Month != 12 {
		t.Fatalf("unexpected last month: %+v", months[len(months)-1])
	}
}

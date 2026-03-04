package domain

import "time"

type Holiday struct {
	Date      time.Time
	Name      string
	Year      int
	Month     int
	DateKind  string
	IsHoliday bool
	FetchedAt time.Time
}

type MonthKey struct {
	Year  int
	Month int
}

type MonthSyncState struct {
	Year         int
	Month        int
	LastSyncedAt time.Time
	ResultCode   string
}

func MonthKeysBetween(start, end time.Time) []MonthKey {
	cursor := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	last := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	months := make([]MonthKey, 0, 12)
	for !cursor.After(last) {
		months = append(months, MonthKey{Year: cursor.Year(), Month: int(cursor.Month())})
		cursor = cursor.AddDate(0, 1, 0)
	}
	return months
}

func MonthKeysInYearRange(fromYear, toYear int) []MonthKey {
	months := make([]MonthKey, 0, (toYear-fromYear+1)*12)
	for year := fromYear; year <= toYear; year += 1 {
		for month := 1; month <= 12; month += 1 {
			months = append(months, MonthKey{Year: year, Month: month})
		}
	}
	return months
}

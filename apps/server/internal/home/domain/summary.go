package domain

import "time"

type TimeWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type EventSummary struct {
	ID         string    `json:"id"`
	CalendarID string    `json:"calendar_id"`
	Title      string    `json:"title"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	IsAllDay   bool      `json:"is_all_day"`
	ColorID    *string   `json:"color_id"`
}

type TodoSummary struct {
	ID       string  `json:"id"`
	ListID   string  `json:"list_id"`
	Title    string  `json:"title"`
	DueDate  *string `json:"due_date"`
	DueTime  *string `json:"due_time"`
	IsPinned bool    `json:"is_pinned"`
}

type RecentFileSummary struct {
	ID          string    `json:"id"`
	FolderID    *string   `json:"folder_id"`
	Name        string    `json:"name"`
	MimeType    string    `json:"mime_type"`
	SizeBytes   int64     `json:"size_bytes"`
	ThumbStatus string    `json:"thumb_status"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StorageSummary struct {
	UsedBytes    int64              `json:"used_bytes"`
	QuotaBytes   int64              `json:"quota_bytes"`
	UsagePercent float64            `json:"usage_percent"`
	Breakdown    []StorageTypeUsage `json:"breakdown"`
}

type StorageTypeUsage struct {
	Type    string  `json:"type"`
	Bytes   int64   `json:"bytes"`
	Percent float64 `json:"percent"`
}

type Summary struct {
	Window TimeWindow `json:"window"`
	Events struct {
		Items      []EventSummary `json:"items"`
		TotalCount int            `json:"total_count"`
	} `json:"events"`
	Todos struct {
		Overdue      []TodoSummary `json:"overdue"`
		Today        []TodoSummary `json:"today"`
		OverdueCount int           `json:"overdue_count"`
		TodayCount   int           `json:"today_count"`
	} `json:"todos"`
	Files struct {
		Recent     []RecentFileSummary `json:"recent"`
		TotalCount int                 `json:"total_count"`
	} `json:"files"`
	Storage StorageSummary `json:"storage"`
}

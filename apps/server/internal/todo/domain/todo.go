package domain

import "time"

type TodoList struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	GoogleID           *string   `json:"google_id,omitempty"`
	GoogleAccountID    *string   `json:"google_account_id,omitempty"`
	GoogleAccountEmail *string   `json:"google_account_email,omitempty"`
	Name               string    `json:"name"`
	SortOrder          int       `json:"sort_order"`
	ActiveCount        int       `json:"active_count"`
	DoneCount          int       `json:"done_count"`
	TotalCount         int       `json:"total_count"`
	Source             string    `json:"source"` // google | local
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Todo struct {
	ID        string     `json:"id"`
	ListID    string     `json:"list_id"`
	UserID    string     `json:"user_id"`
	ParentID  *string    `json:"parent_id"`
	GoogleID  *string    `json:"google_id,omitempty"`
	Title     string     `json:"title"`
	Notes     string     `json:"notes"`
	Due       *string    `json:"due"` // YYYY-MM-DD
	Priority  string     `json:"priority"`
	IsDone    bool       `json:"is_done"`
	IsPinned  bool       `json:"is_pinned"`
	SortOrder int        `json:"sort_order"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

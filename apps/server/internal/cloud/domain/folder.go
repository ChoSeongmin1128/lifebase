package domain

import "time"

type Folder struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ParentID  *string    `json:"parent_id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

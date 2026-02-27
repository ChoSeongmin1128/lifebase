package domain

import "time"

type Media struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	FolderID    *string    `json:"folder_id"`
	Name        string     `json:"name"`
	MimeType    string     `json:"mime_type"`
	SizeBytes   int64      `json:"size_bytes"`
	StoragePath string     `json:"-"`
	ThumbStatus string     `json:"thumb_status"`
	TakenAt     *time.Time `json:"taken_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

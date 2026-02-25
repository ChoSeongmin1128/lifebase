package domain

import "time"

type Share struct {
	ID         string    `json:"id"`
	FolderID   string    `json:"folder_id"`
	OwnerID    string    `json:"owner_id"`
	SharedWith string    `json:"shared_with"`
	Role       string    `json:"role"` // viewer, editor
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type ShareInvite struct {
	ID         string     `json:"id"`
	FolderID   string     `json:"folder_id"`
	OwnerID    string     `json:"owner_id"`
	Token      string     `json:"token,omitempty"`
	Role       string     `json:"role"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

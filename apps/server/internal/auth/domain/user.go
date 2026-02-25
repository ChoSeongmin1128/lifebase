package domain

import "time"

type User struct {
	ID                string
	Email             string
	Name              string
	Picture           string
	StorageQuotaBytes int64
	StorageUsedBytes  int64
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type GoogleAccount struct {
	ID             string
	UserID         string
	GoogleEmail    string
	GoogleID       string
	AccessToken    string
	RefreshToken   string
	TokenExpiresAt *time.Time
	Scopes         string
	Status         string // active, reauth_required, revoked
	IsPrimary      bool
	ConnectedAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

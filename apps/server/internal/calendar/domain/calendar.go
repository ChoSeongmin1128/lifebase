package domain

import "time"

type Calendar struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	GoogleID        *string    `json:"google_id,omitempty"`
	GoogleAccountID *string    `json:"google_account_id,omitempty"`
	Name            string     `json:"name"`
	Kind            string     `json:"kind"`
	ColorID         *string    `json:"color_id,omitempty"`
	IsPrimary       bool       `json:"is_primary"`
	IsVisible       bool       `json:"is_visible"`
	IsReadOnly      bool       `json:"is_readonly"`
	IsSpecial       bool       `json:"is_special"`
	SyncToken       *string    `json:"-"`
	SyncedStart     *time.Time `json:"synced_start,omitempty"`
	SyncedEnd       *time.Time `json:"synced_end,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

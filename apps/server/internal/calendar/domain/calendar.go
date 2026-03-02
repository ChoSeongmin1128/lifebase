package domain

import "time"

type Calendar struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	GoogleID        *string   `json:"google_id,omitempty"`
	GoogleAccountID *string   `json:"google_account_id,omitempty"`
	Name            string    `json:"name"`
	ColorID         *string   `json:"color_id,omitempty"`
	IsPrimary       bool      `json:"is_primary"`
	IsVisible       bool      `json:"is_visible"`
	SyncToken       *string   `json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

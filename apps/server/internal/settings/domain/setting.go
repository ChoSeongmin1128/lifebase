package domain

import "time"

type Setting struct {
	UserID    string
	Key       string
	Value     string
	UpdatedAt time.Time
}

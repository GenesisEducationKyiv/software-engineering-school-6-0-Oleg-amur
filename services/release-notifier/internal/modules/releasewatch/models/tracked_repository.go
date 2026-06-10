package models

import "time"

type TrackedRepository struct {
	ID          int
	Name        string
	LastSeenTag string
	CreatedAt   time.Time
}

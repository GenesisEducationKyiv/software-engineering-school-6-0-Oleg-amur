package model

import "time"

type Repository struct {
	ID          int
	Name        string
	LastSeenTag string
	CreatedAt   time.Time
}

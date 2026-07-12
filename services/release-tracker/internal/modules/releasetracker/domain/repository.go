package domain

import "time"

type Repository struct {
	ID          int64
	Name        string
	LastSeenTag string
	CreatedAt   time.Time
}

type ActiveSubscription struct {
	Email            string
	UnsubscribeToken string
}

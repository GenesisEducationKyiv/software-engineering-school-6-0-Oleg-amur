package models

import (
	"time"
)

const (
	StatusPending = iota
	StatusActive
)

type Repository struct {
	ID          int
	Name        string
	LastSeenTag string
	CreatedAt   time.Time
}

type Subscriber struct {
	ID        int
	Email     string
	CreatedAt time.Time
}

type Subscription struct {
	ID                 int
	SubscriberID       int
	RepositoryID       int
	SubscriptionStatus int
	Token              string
	CreatedAt          time.Time

	Subscriber *Subscriber
	Repository *Repository
}

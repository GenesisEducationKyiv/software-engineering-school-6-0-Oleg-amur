package domain

import "time"

const (
	StatusPending = iota
	StatusActive
)

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

type RepositorySubscription struct {
	Email            string
	UnsubscribeToken string
}

type Repository struct {
	Name        string
	LastSeenTag string
}

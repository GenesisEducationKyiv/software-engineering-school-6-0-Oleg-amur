package model

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

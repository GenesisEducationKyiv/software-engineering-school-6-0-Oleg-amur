package domain

import (
	"time"

	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
)

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
	Repository *releasetrackerdomain.Repository
}

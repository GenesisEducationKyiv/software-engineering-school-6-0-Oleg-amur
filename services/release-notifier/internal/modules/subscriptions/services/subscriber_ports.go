package services

import "context"

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"

type SubscriberRepo interface {
	GetByEmail(ctx context.Context, email string) (*models.Subscriber, error)
	Create(ctx context.Context, email string) (*models.Subscriber, error)
}

type subscriberService interface {
	GetOrCreate(ctx context.Context, email string) (*models.Subscriber, error)
}

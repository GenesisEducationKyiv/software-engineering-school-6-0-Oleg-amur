package subscriptiongrpc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/usecase"
	subscriptionsv1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerListsActiveSubscriptions(t *testing.T) {
	queries := &activeSubscriptionsQueryFake{
		subscriptions: []domain.RepositorySubscription{{
			Email:            "user@example.com",
			UnsubscribeToken: "token",
		}},
	}
	handler := NewHandler(discardLogger(), queries)

	response, err := handler.ListActiveSubscriptionsByRepository(
		t.Context(),
		&subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest{RepositoryId: 7},
	)
	if err != nil {
		t.Fatalf("list active subscriptions: %v", err)
	}
	if queries.repositoryID != 7 {
		t.Fatalf("repository ID = %d, want 7", queries.repositoryID)
	}
	if len(response.GetSubscriptions()) != 1 {
		t.Fatalf("subscriptions length = %d, want 1", len(response.GetSubscriptions()))
	}
	got := response.GetSubscriptions()[0]
	if got.GetEmail() != "user@example.com" || got.GetUnsubscribeToken() != "token" {
		t.Fatalf("unexpected subscription: %+v", got)
	}
}

func TestHandlerRejectsMissingRepository(t *testing.T) {
	handler := NewHandler(discardLogger(), &activeSubscriptionsQueryFake{})

	_, err := handler.ListActiveSubscriptionsByRepository(
		t.Context(),
		&subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest{},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerMapsQueryFailureToInternal(t *testing.T) {
	handler := NewHandler(discardLogger(), &activeSubscriptionsQueryFake{err: errors.New("query failed")})

	_, err := handler.ListActiveSubscriptionsByRepository(
		t.Context(),
		&subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest{RepositoryId: 7},
	)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Internal)
	}
}

type activeSubscriptionsQueryFake struct {
	repositoryID  int64
	subscriptions []domain.RepositorySubscription
	err           error
}

func (f *activeSubscriptionsQueryFake) GetActiveSubscriptionsByRepository(
	_ context.Context,
	repositoryID int64,
) ([]domain.RepositorySubscription, error) {
	f.repositoryID = repositoryID
	return f.subscriptions, f.err
}

func (f *activeSubscriptionsQueryFake) Subscribe(context.Context, subscriptionusecase.SubscribeRequest) error {
	return nil
}

func (f *activeSubscriptionsQueryFake) Confirm(context.Context, string) error {
	return nil
}

func (f *activeSubscriptionsQueryFake) Unsubscribe(context.Context, string) error {
	return nil
}

func (f *activeSubscriptionsQueryFake) GetSubscriptions(
	context.Context,
	string,
) ([]subscriptionusecase.SubscriptionView, error) {
	return nil, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

package subscriptiongrpc

import (
	"context"
	"errors"
	"testing"
	"time"

	subscriptionsv1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClientListsActiveSubscriptions(t *testing.T) {
	rpc := &subscriptionQueryClientFake{
		response: &subscriptionsv1.ListActiveSubscriptionsByRepositoryResponse{
			Subscriptions: []*subscriptionsv1.ActiveSubscription{{
				Email:            "user@example.com",
				UnsubscribeToken: "token",
			}},
		},
	}
	client := NewClient(rpc, time.Second)

	subscriptions, err := client.ListActiveByRepository(t.Context(), 7)
	if err != nil {
		t.Fatalf("list active subscriptions: %v", err)
	}
	if rpc.repositoryID != 7 {
		t.Fatalf("repository ID = %d, want 7", rpc.repositoryID)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("subscriptions length = %d, want 1", len(subscriptions))
	}
	if subscriptions[0].Email != "user@example.com" || subscriptions[0].UnsubscribeToken != "token" {
		t.Fatalf("unexpected subscription: %+v", subscriptions[0])
	}
}

func TestClientPreservesGRPCStatus(t *testing.T) {
	rpc := &subscriptionQueryClientFake{err: status.Error(codes.Unavailable, "service unavailable")}
	client := NewClient(rpc, time.Second)

	_, err := client.ListActiveByRepository(t.Context(), 7)
	if !errors.Is(err, rpc.err) {
		t.Fatalf("error does not wrap RPC failure: %v", err)
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("status code = %s, want %s", status.Code(err), codes.Unavailable)
	}
}

type subscriptionQueryClientFake struct {
	subscriptionsv1.SubscriptionServiceClient
	repositoryID int64
	response     *subscriptionsv1.ListActiveSubscriptionsByRepositoryResponse
	err          error
}

func (f *subscriptionQueryClientFake) ListActiveSubscriptionsByRepository(
	_ context.Context,
	req *subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest,
	_ ...grpc.CallOption,
) (*subscriptionsv1.ListActiveSubscriptionsByRepositoryResponse, error) {
	f.repositoryID = req.GetRepositoryId()
	return f.response, f.err
}

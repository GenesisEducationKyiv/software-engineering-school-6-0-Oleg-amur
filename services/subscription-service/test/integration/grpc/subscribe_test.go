//go:build integration

package grpc_test

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/api/grpc/pb"
	"google.golang.org/grpc/codes"
)

func (s *GRPCSuite) TestSubscribe_CreatesPendingSubscription() {
	_, err := s.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	s.assertGRPCCode(err, codes.OK)

	event := s.receiveSubscriptionEvent()
	s.Equal("user@example.com", event.Email)
	s.NotEmpty(event.ConfirmationToken)

	resp, err := s.client.GetSubscriptions(context.Background(), &pb.GetSubscriptionsRequest{
		Email: "user@example.com",
	})
	s.assertGRPCCode(err, codes.OK)
	s.Empty(resp.GetSubscriptions(), "active subscriptions before confirmation")
}

func (s *GRPCSuite) TestSubscribe_ReturnsInvalidArgumentForInvalidRepoFormat() {
	_, err := s.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "invalid",
	})

	s.assertGRPCCode(err, codes.InvalidArgument)
}

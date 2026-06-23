//go:build integration

package grpc_test

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	"google.golang.org/grpc/codes"
)

func (s *GRPCSuite) TestGetSubscriptions_ReturnsActiveSubscription() {
	_, err := s.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	s.assertGRPCCode(err, codes.OK)

	token := s.receiveSubscriptionToken()
	_, err = s.client.Confirm(context.Background(), &pb.ConfirmRequest{Token: token})
	s.assertGRPCCode(err, codes.OK)

	resp, err := s.client.GetSubscriptions(context.Background(), &pb.GetSubscriptionsRequest{
		Email: "user@example.com",
	})
	s.assertGRPCCode(err, codes.OK)
	s.Require().Len(resp.GetSubscriptions(), 1, "active subscriptions")

	sub := resp.GetSubscriptions()[0]
	s.Equal("user@example.com", sub.GetEmail())
	s.Equal("owner/repo", sub.GetRepo())
	s.True(sub.GetConfirmed())
	s.Equal("v1.0.0", sub.GetLastSeenTag())
}

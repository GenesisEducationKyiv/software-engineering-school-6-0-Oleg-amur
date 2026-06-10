//go:build integration

package grpc_test

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
	"google.golang.org/grpc/codes"
)

func (s *GRPCSuite) TestConfirm_ActivatesSubscription() {
	_, err := s.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	s.assertGRPCCode(err, codes.OK)

	token := s.receiveSubscriptionToken()

	_, err = s.client.Confirm(context.Background(), &pb.ConfirmRequest{Token: token})
	s.assertGRPCCode(err, codes.OK)

	s.assertSubscriptionStatus(token, subscriptionmodels.StatusActive)
}

func (s *GRPCSuite) assertSubscriptionStatus(token string, want int) {
	s.T().Helper()

	var got int
	err := s.app.DB.QueryRowContext(
		context.Background(),
		`SELECT subscription_status FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&got)
	s.Require().NoError(err, "get subscription status by token")
	s.Equal(want, got)
}

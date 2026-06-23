//go:build integration

package grpc_test

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/test/integration/testkit"
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

	testkit.AssertSubscriptionStatus(s.T(), s.app.DB, token, subscriptionmodels.StatusActive)
}

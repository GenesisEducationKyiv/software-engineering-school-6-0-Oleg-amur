//go:build integration

package grpc_test

import (
	"context"

	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/test/integration/testkit"
	pb "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
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

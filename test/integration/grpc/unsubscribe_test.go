//go:build integration

package grpc_test

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc/pb"
	"google.golang.org/grpc/codes"
)

func (s *GRPCSuite) TestUnsubscribe_RemovesSubscription() {
	_, err := s.client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	})
	s.assertGRPCCode(err, codes.OK)

	token := s.receiveSubscriptionToken()

	_, err = s.client.Unsubscribe(context.Background(), &pb.UnsubscribeRequest{Token: token})
	s.assertGRPCCode(err, codes.OK)

	s.assertSubscriptionMissing(token)
}

func (s *GRPCSuite) assertSubscriptionMissing(token string) {
	s.T().Helper()

	var id int
	err := s.app.DB.QueryRowContext(
		context.Background(),
		`SELECT id FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&id)
	s.Require().True(errors.Is(err, sql.ErrNoRows), "expected subscription to be deleted, got id=%d err=%v", id, err)
}

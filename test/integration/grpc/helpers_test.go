//go:build integration

package grpc_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *GRPCSuite) receiveSubscriptionEvent() model.SubscriptionEvent {
	s.T().Helper()

	select {
	case event := <-s.app.Events:
		return event
	default:
		s.T().Fatal("want subscription event to be queued")
	}

	return model.SubscriptionEvent{}
}

func (s *GRPCSuite) receiveSubscriptionToken() string {
	s.T().Helper()

	event := s.receiveSubscriptionEvent()
	s.Require().NotEmpty(event.Token, "subscription event token")
	return event.Token
}

func (s *GRPCSuite) assertGRPCCode(gotErr error, wantCode codes.Code) {
	s.T().Helper()

	if wantCode == codes.OK {
		s.NoError(gotErr)
		return
	}
	gotCode := status.Code(gotErr)
	s.Equal(wantCode, gotCode, "gRPC error: %v", gotErr)
}

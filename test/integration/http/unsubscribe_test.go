//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
)

func (s *HTTPSuite) TestUnsubscribe_RemovesSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := s.createSubscription(subscribeRequest)
	s.get("/api/v1/confirm/"+token, http.StatusOK, nil)

	s.get("/api/v1/unsubscribe/"+token, http.StatusOK, nil)

	s.assertSubscriptions(subscribeRequest.Email, func(response []dto.Subscription) {
		s.Empty(response, "active subscriptions after unsubscribe")
	})
}

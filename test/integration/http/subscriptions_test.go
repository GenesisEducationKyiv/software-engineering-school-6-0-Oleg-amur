//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
)

func (s *HTTPSuite) TestGetSubscriptions_ReturnsActiveSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := s.createSubscription(subscribeRequest)
	s.get("/api/v1/confirm/"+token, http.StatusOK, nil)

	s.assertActiveSubscription(dto.Subscription{
		Email:       subscribeRequest.Email,
		Repo:        subscribeRequest.Repo,
		Confirmed:   true,
		LastSeenTag: "v1.0.0",
	})
}

func (s *HTTPSuite) TestGetSubscriptions_ReturnsBadRequestForMissingEmail() {
	s.get("/api/v1/subscriptions", http.StatusBadRequest, nil)
}

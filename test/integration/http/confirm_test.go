//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func (s *HTTPSuite) TestConfirm_ActivatesSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	token := s.createSubscription(subscribeRequest)

	s.get("/api/v1/confirm/"+token, http.StatusOK, nil)

	s.assertSubscriptionStatus(token, model.StatusActive)
}

func (s *HTTPSuite) TestConfirm_ReturnsNotFoundForMissingToken() {
	s.get("/api/v1/confirm/missing-token", http.StatusNotFound, nil)
}

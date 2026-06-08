//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/model"
)

func (s *HTTPSuite) TestConfirm_ActivatesSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	s.postSubscribe(subscribeRequest, http.StatusOK)

	token := s.receiveSubscriptionToken()

	s.getConfirm(token, http.StatusOK)

	s.assertSubscriptionStatus(token, model.StatusActive)
}

func (s *HTTPSuite) TestConfirm_ReturnsNotFoundForMissingToken() {
	s.getConfirm("missing-token", http.StatusNotFound)
}

func (s *HTTPSuite) assertSubscriptionStatus(token string, want int) {
	s.T().Helper()

	var got int
	err := s.app.DB.QueryRowContext(
		s.T().Context(),
		`SELECT subscription_status FROM subscriptions WHERE token = $1`,
		token,
	).Scan(&got)
	s.Require().NoError(err, "get subscription status by token")
	s.Equal(want, got)
}

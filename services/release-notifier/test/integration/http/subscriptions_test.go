//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http/dto"
)

func (s *HTTPSuite) TestGetSubscriptions_ReturnsActiveSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	s.postSubscribe(subscribeRequest, http.StatusOK)

	token := s.receiveSubscriptionToken()
	s.getConfirm(token, http.StatusOK)

	s.assertActiveSubscription(dto.Subscription{
		Email:       subscribeRequest.Email,
		Repo:        subscribeRequest.Repo,
		Confirmed:   true,
		LastSeenTag: "v1.0.0",
	})
}

func (s *HTTPSuite) TestGetSubscriptions_ReturnsBadRequestForMissingEmail() {
	s.getSubscriptionsWithoutEmail(http.StatusBadRequest)
}

func (s *HTTPSuite) assertActiveSubscription(want dto.Subscription) {
	s.T().Helper()

	response := s.getSubscriptions(want.Email)
	s.Require().Len(response, 1, "active subscriptions")

	got := response[0]
	s.Equal(want.Email, got.Email)
	s.Equal(want.Repo, got.Repo)
	s.Equal(want.Confirmed, got.Confirmed)
	s.Equal(want.LastSeenTag, got.LastSeenTag)
}

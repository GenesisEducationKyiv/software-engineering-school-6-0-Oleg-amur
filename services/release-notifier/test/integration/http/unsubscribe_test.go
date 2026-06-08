//go:build integration

package http_test

import (
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http/dto"
)

func (s *HTTPSuite) TestUnsubscribe_RemovesSubscription() {
	subscribeRequest := dto.SubscribeRequest{
		Email: "user@example.com",
		Repo:  "owner/repo",
	}
	s.postSubscribe(subscribeRequest, http.StatusOK)

	token := s.receiveSubscriptionToken()
	s.getConfirm(token, http.StatusOK)

	s.getUnsubscribe(token, http.StatusOK)

	response := s.getSubscriptions(subscribeRequest.Email)
	s.Empty(response, "active subscriptions after unsubscribe")
}

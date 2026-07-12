package subscriptiongrpc

import (
	"context"
	"fmt"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
	subscriptionsv1 "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/gen/subscriptions/v1"
)

type Client struct {
	rpc     subscriptionsv1.SubscriptionServiceClient
	timeout time.Duration
}

func NewClient(rpc subscriptionsv1.SubscriptionServiceClient, timeout time.Duration) *Client {
	return &Client{rpc: rpc, timeout: timeout}
}

func (c *Client) ListActiveByRepository(
	ctx context.Context,
	repositoryID int64,
) ([]domain.ActiveSubscription, error) {
	callCtx := ctx
	if c.timeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	response, err := c.rpc.ListActiveSubscriptionsByRepository(
		callCtx,
		&subscriptionsv1.ListActiveSubscriptionsByRepositoryRequest{RepositoryId: repositoryID},
	)
	if err != nil {
		return nil, fmt.Errorf("list active subscriptions over gRPC: %w", err)
	}

	result := make([]domain.ActiveSubscription, 0, len(response.GetSubscriptions()))
	for _, subscription := range response.GetSubscriptions() {
		result = append(result, domain.ActiveSubscription{
			Email:            subscription.GetEmail(),
			UnsubscribeToken: subscription.GetUnsubscribeToken(),
		})
	}

	return result, nil
}

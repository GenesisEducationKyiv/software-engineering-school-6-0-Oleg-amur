package subscriptionhttp

type ActiveSubscription struct {
	Email            string `json:"email"`
	UnsubscribeToken string `json:"unsubscribe_token"`
}

type ActiveSubscriptionsResponse struct {
	Subscriptions []ActiveSubscription `json:"subscriptions"`
}

type SubscribeRequest struct {
	Email string `json:"email"`
	Repo  string `json:"repo"`
}

type Subscription struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

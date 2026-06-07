package events

const (
	SubscriptionConfirmationRequestedType = "notification.subscription_confirmation_requested"
	ReleaseNotificationRequestedType      = "notification.release_notification_requested"
)

type SubscriptionConfirmationRequested struct {
	EventID string `json:"event_id"`
	Email   string `json:"email"`
	Token   string `json:"token"`
}

type ReleaseNotificationRequested struct {
	EventID          string `json:"event_id"`
	Email            string `json:"email"`
	Repo             string `json:"repo"`
	Tag              string `json:"tag"`
	UnsubscribeToken string `json:"unsubscribe_token"`
}

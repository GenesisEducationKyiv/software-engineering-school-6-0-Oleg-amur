package events

import "fmt"

const (
	SubscriptionConfirmationRequestedType = "notification.subscription_confirmation_requested"
	ReleaseNotificationRequestedType      = "notification.release_notification_requested"
	NotificationSchemaVersion             = 1
)

type SubscriptionConfirmationRequested struct {
	EventID       string `json:"event_id"`
	SchemaVersion int    `json:"schema_version"`
	Email         string `json:"email"`
	Token         string `json:"token"`
}

type ReleaseNotificationRequested struct {
	EventID          string `json:"event_id"`
	SchemaVersion    int    `json:"schema_version"`
	Email            string `json:"email"`
	Repo             string `json:"repo"`
	Tag              string `json:"tag"`
	UnsubscribeToken string `json:"unsubscribe_token"`
}

func ValidateNotificationSchemaVersion(version int) error {
	if version != NotificationSchemaVersion {
		return fmt.Errorf("unsupported notification schema version: %d", version)
	}
	return nil
}

package events

import "fmt"

const (
	SubscriptionConfirmationRequestedType = "notification.subscription_confirmation_requested"
	SubscriptionConfirmationSucceededType = "notification.subscription_confirmation_succeeded"
	SubscriptionConfirmationFailedType    = "notification.subscription_confirmation_failed"
	ReleaseNotificationRequestedType      = "notification.release_notification_requested"
	NotificationSchemaVersion             = 1
)

type SubscriptionConfirmationRequested struct {
	EventID           string `json:"event_id"`
	SchemaVersion     int    `json:"schema_version"`
	SagaID            int    `json:"saga_id"`
	SubscriptionID    int    `json:"subscription_id"`
	Email             string `json:"email"`
	ConfirmationToken string `json:"confirmation_token"`
}

type SubscriptionConfirmationSucceeded struct {
	EventID           string `json:"event_id"`
	SchemaVersion     int    `json:"schema_version"`
	SagaID            int    `json:"saga_id"`
	SubscriptionID    int    `json:"subscription_id"`
	Email             string `json:"email"`
	ConfirmationToken string `json:"confirmation_token"`
}

type SubscriptionConfirmationFailed struct {
	EventID           string `json:"event_id"`
	SchemaVersion     int    `json:"schema_version"`
	SagaID            int    `json:"saga_id"`
	SubscriptionID    int    `json:"subscription_id"`
	Email             string `json:"email"`
	ConfirmationToken string `json:"confirmation_token"`
	Reason            string `json:"reason"`
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

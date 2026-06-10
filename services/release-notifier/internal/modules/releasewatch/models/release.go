package models

type ReleaseEvent struct {
	RepoID   int
	RepoName string
	Tag      string
}

type NotificationRecipient struct {
	Email            string
	UnsubscribeToken string
}

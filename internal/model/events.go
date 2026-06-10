package model

type ReleaseEvent struct {
	RepoID   int
	RepoName string
	Tag      string
	Token    string
}

type SubscriptionEvent struct {
	Email string
	Token string
}

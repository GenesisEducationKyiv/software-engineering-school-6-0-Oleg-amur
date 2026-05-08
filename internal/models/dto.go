package models

type SubscribeRequest struct {
	Email string
	Repo  string
}

type SubscriptionDTO struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag string
}

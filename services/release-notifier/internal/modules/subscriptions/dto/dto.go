package dto

type SubscriptionDTO struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag string
}

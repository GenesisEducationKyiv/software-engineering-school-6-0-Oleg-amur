package usecase

type SubscriptionView struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag string
}

type RepositoryView struct {
	Name        string
	LastSeenTag string
}

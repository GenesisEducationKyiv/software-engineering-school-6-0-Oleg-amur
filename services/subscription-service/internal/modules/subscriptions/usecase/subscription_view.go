package usecase

type SubscriptionView struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag string
}

type RepositoryView struct {
	ID          int64
	Name        string
	LastSeenTag string
}

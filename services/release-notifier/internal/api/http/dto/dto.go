package dto

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

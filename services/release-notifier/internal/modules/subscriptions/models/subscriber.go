package models

import "time"

type Subscriber struct {
	ID        int
	Email     string
	CreatedAt time.Time
}

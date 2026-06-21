package worker

import (
	"context"
	"time"
)

type Scanner interface {
	Scan(context.Context)
}

type Scheduler struct {
	scanner  Scanner
	interval time.Duration
}

func NewScheduler(scanner Scanner, interval time.Duration) *Scheduler {
	return &Scheduler{scanner: scanner, interval: interval}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.scanner.Scan(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.scanner.Scan(ctx)
		}
	}
}

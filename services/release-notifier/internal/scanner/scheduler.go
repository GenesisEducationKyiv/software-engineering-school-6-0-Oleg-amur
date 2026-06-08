package scanner

import (
	"context"
	"log/slog"
	"time"
)

type releaseScanner interface {
	Scan(ctx context.Context)
}

type Scheduler struct {
	log      *slog.Logger
	scanner  releaseScanner
	interval time.Duration
}

func NewScheduler(log *slog.Logger, scanner releaseScanner, interval time.Duration) *Scheduler {
	return &Scheduler{
		log:      log,
		scanner:  scanner,
		interval: interval,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.log.Info("background scheduler started", "interval", s.interval)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.scanner.Scan(ctx)

	for {
		select {
		case <-ctx.Done():
			s.log.Info("background scheduler stopping")
			return
		case <-ticker.C:
			s.scanner.Scan(ctx)
		}
	}
}

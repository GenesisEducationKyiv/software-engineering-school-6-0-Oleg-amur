package worker

import (
	"context"
	"log/slog"
	"time"
)

type Job interface {
	Execute(context.Context)
}

type Scheduler struct {
	log      *slog.Logger
	job      Job
	interval time.Duration
}

func NewScheduler(log *slog.Logger, job Job, interval time.Duration) *Scheduler {
	return &Scheduler{log: log, job: job, interval: interval}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.job.Execute(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.job.Execute(ctx)
		}
	}
}

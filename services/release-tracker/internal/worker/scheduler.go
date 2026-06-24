package worker

import (
	"context"
	"time"
)

type Job interface {
	Execute(context.Context)
}

type Scheduler struct {
	job      Job
	interval time.Duration
}

func NewScheduler(job Job, interval time.Duration) *Scheduler {
	return &Scheduler{job: job, interval: interval}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.job.Execute(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.job.Execute(ctx)
		}
	}
}

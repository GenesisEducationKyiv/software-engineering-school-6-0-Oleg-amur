package observability

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type countingDatabasePinger struct {
	calls atomic.Int32
}

func (p *countingDatabasePinger) PingContext(context.Context) error {
	p.calls.Add(1)
	return nil
}

func TestDatabaseCollectorCachesPingWithinTTL(t *testing.T) {
	pinger := &countingDatabasePinger{}
	collector := &databaseCollector{
		db:           pinger,
		log:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		pingTimeout:  time.Second,
		pingCacheTTL: time.Minute,
	}

	firstUp, _ := collector.ping()
	secondUp, _ := collector.ping()

	if firstUp != 1 || secondUp != 1 {
		t.Fatalf("ping up values = %v, %v; want 1, 1", firstUp, secondUp)
	}
	if got := pinger.calls.Load(); got != 1 {
		t.Fatalf("PingContext calls = %d; want 1", got)
	}
}

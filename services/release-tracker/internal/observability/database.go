package observability

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	databasePingTimeout  = time.Second
	databasePingCacheTTL = 5 * time.Second
)

type DatabasePinger interface {
	PingContext(context.Context) error
}

type databaseCollector struct {
	db           DatabasePinger
	log          *slog.Logger
	up           *prometheus.Desc
	pingDuration *prometheus.Desc
	pingTimeout  time.Duration
	pingCacheTTL time.Duration

	mu             sync.Mutex
	cached         bool
	cachedAt       time.Time
	cachedUp       float64
	cachedDuration float64
}

func RegisterDatabaseMetrics(db DatabasePinger, log *slog.Logger) error {
	collector := &databaseCollector{
		db:  db,
		log: log,
		up: prometheus.NewDesc(
			"release_tracker_database_up",
			"Whether the application can ping the database: 1 for up, 0 for down.",
			nil,
			nil,
		),
		pingDuration: prometheus.NewDesc(
			"release_tracker_database_ping_duration_seconds",
			"Database ping duration in seconds.",
			nil,
			nil,
		),
		pingTimeout:  databasePingTimeout,
		pingCacheTTL: databasePingCacheTTL,
	}

	if err := prometheus.Register(collector); err != nil {
		var alreadyRegistered prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegistered) {
			return nil
		}
		return err
	}
	return nil
}

func (c *databaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.up
	ch <- c.pingDuration
}

func (c *databaseCollector) Collect(ch chan<- prometheus.Metric) {
	up, duration := c.ping()

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, up)
	ch <- prometheus.MustNewConstMetric(c.pingDuration, prometheus.GaugeValue, duration)
}

func (c *databaseCollector) ping() (float64, float64) {
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached && c.pingCacheTTL > 0 && now.Sub(c.cachedAt) < c.pingCacheTTL {
		return c.cachedUp, c.cachedDuration
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.pingTimeout)
	defer cancel()

	start := time.Now()
	err := c.db.PingContext(ctx)
	duration := time.Since(start).Seconds()
	up := 1.0
	if err != nil {
		up = 0
		c.log.Error("database ping metric collection failed", "err", err)
	}

	c.cached = true
	c.cachedAt = now
	c.cachedUp = up
	c.cachedDuration = duration

	return up, duration
}

package observability

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const databasePingTimeout = time.Second

type DatabasePinger interface {
	PingContext(context.Context) error
}

type databaseCollector struct {
	db             DatabasePinger
	log            *slog.Logger
	up             *prometheus.Desc
	pingDuration   *prometheus.Desc
	pingTimeoutSec time.Duration
}

func RegisterDatabaseMetrics(db DatabasePinger, log *slog.Logger) error {
	collector := &databaseCollector{
		db:  db,
		log: log,
		up: prometheus.NewDesc(
			"release_notifier_database_up",
			"Whether the application can ping the database: 1 for up, 0 for down.",
			nil,
			nil,
		),
		pingDuration: prometheus.NewDesc(
			"release_notifier_database_ping_duration_seconds",
			"Database ping duration in seconds.",
			nil,
			nil,
		),
		pingTimeoutSec: databasePingTimeout,
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
	ctx, cancel := context.WithTimeout(context.Background(), c.pingTimeoutSec)
	defer cancel()

	start := time.Now()
	err := c.db.PingContext(ctx)
	duration := time.Since(start).Seconds()

	up := 1.0
	if err != nil {
		up = 0
		c.log.Error("database ping metric collection failed", "err", err)
	}

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, up)
	ch <- prometheus.MustNewConstMetric(c.pingDuration, prometheus.GaugeValue, duration)
}

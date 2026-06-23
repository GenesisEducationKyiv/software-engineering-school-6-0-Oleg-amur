//go:build integration

package eventbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/test/integration/testkit"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/suite"
)

type EventBusSuite struct {
	suite.Suite

	ctx       context.Context
	cancel    context.CancelFunc
	broker    *testkit.RabbitMQ
	cfg       rabbitmq.Config
	publisher *rabbitmq.Publisher
	conn      *amqp.Connection
	ch        *amqp.Channel
}

func TestEventBusSuite(t *testing.T) {
	suite.Run(t, new(EventBusSuite))
}

func (s *EventBusSuite) SetupSuite() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	startCtx, cancel := context.WithTimeout(s.ctx, 2*time.Minute)
	defer cancel()
	s.broker = testkit.NewRabbitMQ(startCtx, s.T())
}

func (s *EventBusSuite) TearDownSuite() {
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *EventBusSuite) SetupTest() {
	s.cfg = notificationConfig(s.broker.URL)

	var err error
	s.publisher, err = rabbitmq.NewNotificationPublisher(s.cfg)
	s.Require().NoError(err)

	s.conn, s.ch = openRabbitMQChannel(s.T(), s.broker.URL)
}

func (s *EventBusSuite) TearDownTest() {
	if s.publisher != nil {
		s.Require().NoError(s.publisher.Close())
	}
	if s.conn != nil && s.ch != nil {
		closeRabbitMQ(s.T(), s.conn, s.ch)
	}
}

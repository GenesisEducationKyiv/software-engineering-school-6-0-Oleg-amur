package usecase

import (
	"context"
	"io"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type repositoryStoreFake struct {
	getName      string
	repository   *domain.Repository
	getErr       error
	repositories []domain.Repository
	getAllErr    error
	created      string
	createdTag   string
	createErr    error
	updatedID    int
	updatedTag   string
	updateErr    error
}

func (f *repositoryStoreFake) Create(
	_ context.Context,
	name string,
	tag string,
) (*domain.Repository, error) {
	f.created = name
	f.createdTag = tag
	return &domain.Repository{Name: name, LastSeenTag: tag}, f.createErr
}

func (f *repositoryStoreFake) GetByName(
	_ context.Context,
	name string,
) (*domain.Repository, error) {
	f.getName = name
	return f.repository, f.getErr
}

func (f *repositoryStoreFake) GetAll(context.Context) ([]domain.Repository, error) {
	return f.repositories, f.getAllErr
}

func (f *repositoryStoreFake) UpdateTag(_ context.Context, id int, tag string) error {
	f.updatedID = id
	f.updatedTag = tag
	return f.updateErr
}

type githubClientFake struct {
	exists    bool
	latestTag string
	err       error
}

func (f *githubClientFake) RepositoryExists(context.Context, string) (bool, error) {
	return f.exists, f.err
}

func (f *githubClientFake) LatestTag(context.Context, string) (string, error) {
	return f.latestTag, f.err
}

type subscriptionClientFake struct {
	subscriptions []domain.ActiveSubscription
	err           error
}

func (f *subscriptionClientFake) ListActiveByRepository(
	context.Context,
	string,
) ([]domain.ActiveSubscription, error) {
	return f.subscriptions, f.err
}

type notificationPublisherFake struct {
	events []events.ReleaseNotificationRequested
	err    error
}

func (f *notificationPublisherFake) Publish(
	_ context.Context,
	event events.ReleaseNotificationRequested,
) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, event)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

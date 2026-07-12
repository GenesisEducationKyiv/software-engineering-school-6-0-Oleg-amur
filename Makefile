SHELL := /bin/sh

COVERAGE_DIR := coverage
SUBSCRIPTION_SERVICE_DIR := services/subscription-service
RELEASE_TRACKER_DIR := services/release-tracker
NOTIFICATION_WORKER_DIR := services/notification-worker
CONTRACTS_DIR := shared/contracts
MESSAGING_DIR := shared/messaging
FAKE_GITHUB_DIR := test/e2e/fakes/github
E2E_COMPOSE := test/e2e/docker-compose-e2e-tests.yaml
GO_PACKAGES := ./...
INTERNAL_PACKAGES := ./internal/...
INTEGRATION_PACKAGES := ./test/integration/...

.PHONY: proto-lint proto-generate arch-lint lint test test-unit test-integration test-e2e coverage coverage-unit coverage-integration coverage-summary benchmark-seed benchmark-http benchmark-grpc benchmark-client clean-test-artifacts

proto-lint:
	buf lint

proto-generate:
	buf generate

arch-lint:
	cd $(SUBSCRIPTION_SERVICE_DIR) && go-arch-lint check
	cd $(RELEASE_TRACKER_DIR) && go-arch-lint check
	cd $(NOTIFICATION_WORKER_DIR) && go-arch-lint check

lint: arch-lint
	cd $(CONTRACTS_DIR) && golangci-lint run --config=../../.golangci.yml $(GO_PACKAGES)
	cd $(MESSAGING_DIR) && golangci-lint run --config=../../.golangci.yml $(GO_PACKAGES)
	cd $(SUBSCRIPTION_SERVICE_DIR) && golangci-lint run --config=../../.golangci.yml $(GO_PACKAGES)
	cd $(RELEASE_TRACKER_DIR) && golangci-lint run --config=../../.golangci.yml $(GO_PACKAGES)
	cd $(NOTIFICATION_WORKER_DIR) && golangci-lint run --config=../../.golangci.yml $(GO_PACKAGES)
	cd $(FAKE_GITHUB_DIR) && golangci-lint run --config=../../../../.golangci.yml $(GO_PACKAGES)

test: test-unit test-integration test-e2e

test-unit:
	cd $(CONTRACTS_DIR) && go test -v $(GO_PACKAGES)
	cd $(MESSAGING_DIR) && go test -v $(GO_PACKAGES)
	cd $(SUBSCRIPTION_SERVICE_DIR) && go test -v $(GO_PACKAGES)
	cd $(RELEASE_TRACKER_DIR) && go test -v $(GO_PACKAGES)
	cd $(NOTIFICATION_WORKER_DIR) && go test -v $(GO_PACKAGES)
	cd $(FAKE_GITHUB_DIR) && go test -v $(GO_PACKAGES)

test-integration:
	cd $(SUBSCRIPTION_SERVICE_DIR) && go test -tags=integration -count=1 -v $(INTEGRATION_PACKAGES)
	cd $(RELEASE_TRACKER_DIR) && go test -tags=integration -count=1 -v $(INTEGRATION_PACKAGES)

test-e2e:
	docker compose -f $(E2E_COMPOSE) up --build --abort-on-container-exit --exit-code-from playwright; \
	code=$$?; \
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans; \
	exit $$code

coverage: coverage-unit coverage-integration coverage-summary

coverage-unit:
	mkdir -p $(COVERAGE_DIR)
	cd $(SUBSCRIPTION_SERVICE_DIR) && go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/unit-subscription-service.out $(GO_PACKAGES)
	cd $(RELEASE_TRACKER_DIR) && go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/unit-release-tracker.out $(GO_PACKAGES)
	cd $(NOTIFICATION_WORKER_DIR) && go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/unit-notification-worker.out $(GO_PACKAGES)
	cd $(SUBSCRIPTION_SERVICE_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/unit-subscription-service.out
	cd $(RELEASE_TRACKER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/unit-release-tracker.out
	cd $(NOTIFICATION_WORKER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/unit-notification-worker.out

coverage-integration:
	mkdir -p $(COVERAGE_DIR)
	cd $(SUBSCRIPTION_SERVICE_DIR) && go test -tags=integration -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/integration-subscription-service.out $(INTEGRATION_PACKAGES)
	cd $(RELEASE_TRACKER_DIR) && go test -tags=integration -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/integration-release-tracker.out $(INTEGRATION_PACKAGES)
	cd $(SUBSCRIPTION_SERVICE_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/integration-subscription-service.out
	cd $(RELEASE_TRACKER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/integration-release-tracker.out

coverage-summary:
	@echo "Coverage profiles:"
	@echo "  $(COVERAGE_DIR)/unit-subscription-service.out"
	@echo "  $(COVERAGE_DIR)/unit-release-tracker.out"
	@echo "  $(COVERAGE_DIR)/unit-notification-worker.out"
	@echo "  $(COVERAGE_DIR)/integration-subscription-service.out"
	@echo "  $(COVERAGE_DIR)/integration-release-tracker.out"

benchmark-seed:
	./scripts/benchmark/seed-subscription-data.sh

benchmark-http:
	./scripts/benchmark/http-autocannon.sh

benchmark-grpc:
	./scripts/benchmark/grpc-ghz.sh

benchmark-client:
	go run ./benchmarks/subscription-client-compare

clean-test-artifacts:
	rm -rf $(COVERAGE_DIR)
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans

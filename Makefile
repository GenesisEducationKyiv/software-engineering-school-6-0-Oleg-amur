SHELL := /bin/sh

COVERAGE_DIR := coverage
RELEASE_NOTIFIER_DIR := services/release-notifier
NOTIFICATION_WORKER_DIR := services/notification-worker
CONTRACTS_DIR := shared/contracts
FAKE_GITHUB_DIR := test/e2e/fakes/github
E2E_COMPOSE := test/e2e/docker-compose-e2e-tests.yaml
GO_PACKAGES := ./...
INTERNAL_PACKAGES := ./internal/...
INTEGRATION_PACKAGES := ./test/integration/...

.PHONY: lint test test-unit test-integration test-e2e coverage coverage-unit coverage-integration coverage-summary clean-test-artifacts

lint:
	golangci-lint run ./$(CONTRACTS_DIR)/...
	golangci-lint run ./$(RELEASE_NOTIFIER_DIR)/...
	golangci-lint run ./$(NOTIFICATION_WORKER_DIR)/...
	golangci-lint run ./$(FAKE_GITHUB_DIR)/...

test: test-unit test-integration test-e2e

test-unit:
	cd $(CONTRACTS_DIR) && go test -v $(GO_PACKAGES)
	cd $(RELEASE_NOTIFIER_DIR) && go test -v $(GO_PACKAGES)
	cd $(NOTIFICATION_WORKER_DIR) && go test -v $(GO_PACKAGES)
	cd $(FAKE_GITHUB_DIR) && go test -v $(GO_PACKAGES)

test-integration:
	cd $(RELEASE_NOTIFIER_DIR) && go test -tags=integration -count=1 -v $(INTEGRATION_PACKAGES)

test-e2e:
	docker compose -f $(E2E_COMPOSE) up --build --abort-on-container-exit --exit-code-from playwright; \
	code=$$?; \
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans; \
	exit $$code

coverage: coverage-unit coverage-integration coverage-summary

coverage-unit:
	mkdir -p $(COVERAGE_DIR)
	cd $(RELEASE_NOTIFIER_DIR) && go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/unit-release-notifier.out $(GO_PACKAGES)
	cd $(NOTIFICATION_WORKER_DIR) && go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/unit-notification-worker.out $(GO_PACKAGES)
	cd $(RELEASE_NOTIFIER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/unit-release-notifier.out
	cd $(NOTIFICATION_WORKER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/unit-notification-worker.out

coverage-integration:
	mkdir -p $(COVERAGE_DIR)
	cd $(RELEASE_NOTIFIER_DIR) && go test -tags=integration -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=../../$(COVERAGE_DIR)/integration.out $(INTEGRATION_PACKAGES)
	cd $(RELEASE_NOTIFIER_DIR) && go tool cover -func=../../$(COVERAGE_DIR)/integration.out

coverage-summary:
	@echo "Coverage profiles:"
	@echo "  $(COVERAGE_DIR)/unit-release-notifier.out"
	@echo "  $(COVERAGE_DIR)/unit-notification-worker.out"
	@echo "  $(COVERAGE_DIR)/integration.out"

clean-test-artifacts:
	rm -rf $(COVERAGE_DIR)
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans

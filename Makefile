SHELL := /bin/sh

COVERAGE_DIR := coverage
E2E_COMPOSE := test/e2e/docker-compose-e2e-tests.yaml
GO_PACKAGES := ./...
INTERNAL_PACKAGES := ./internal/...
INTEGRATION_PACKAGES := ./test/integration/...

.PHONY: test test-unit test-integration test-e2e coverage coverage-unit coverage-integration coverage-summary clean-test-artifacts

test: test-unit test-integration test-e2e

test-unit:
	go test -v $(GO_PACKAGES)

test-integration:
	go test -tags=integration -count=1 -v $(INTEGRATION_PACKAGES)

test-e2e:
	docker compose -f $(E2E_COMPOSE) up --build --abort-on-container-exit --exit-code-from playwright; \
	code=$$?; \
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans; \
	exit $$code

coverage: coverage-unit coverage-integration coverage-summary

coverage-unit:
	mkdir -p $(COVERAGE_DIR)
	go test -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=$(COVERAGE_DIR)/unit.out $(GO_PACKAGES)
	go tool cover -func=$(COVERAGE_DIR)/unit.out

coverage-integration:
	mkdir -p $(COVERAGE_DIR)
	go test -tags=integration -count=1 -covermode=atomic -coverpkg=$(INTERNAL_PACKAGES) -coverprofile=$(COVERAGE_DIR)/integration.out $(INTEGRATION_PACKAGES)
	go tool cover -func=$(COVERAGE_DIR)/integration.out

coverage-summary:
	@echo "Coverage profiles:"
	@echo "  $(COVERAGE_DIR)/unit.out"
	@echo "  $(COVERAGE_DIR)/integration.out"

clean-test-artifacts:
	rm -rf $(COVERAGE_DIR)
	docker compose -f $(E2E_COMPOSE) down -v --remove-orphans

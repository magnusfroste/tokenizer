.PHONY: build dev test test-unit test-integration test-eval test-policy test-regression eval-report lint fmt vet tidy clean run run-mock migrate seed

GO ?= go
PSQL ?= psql
DATABASE_URL ?= postgres://tokenizer:tokenizer@localhost:5432/tokenizer?sslmode=disable
BIN_DIR := bin
ROUTER_BIN := $(BIN_DIR)/router
MOCK_BIN := $(BIN_DIR)/mock-provider
WORKER_BIN := $(BIN_DIR)/worker
CTL_BIN := $(BIN_DIR)/routerctl

build:
	$(GO) build -o $(ROUTER_BIN) ./cmd/router
	$(GO) build -o $(MOCK_BIN) ./cmd/mock-provider
	$(GO) build -o $(WORKER_BIN) ./cmd/worker
	$(GO) build -o $(CTL_BIN) ./cmd/routerctl

run: build
	$(ROUTER_BIN)

run-mock: build
	$(MOCK_BIN)

dev:
	$(GO) run ./cmd/router

migrate:
	$(PSQL) "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/migrations/001_foundation.sql

seed:
	$(PSQL) "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f db/seeds/local.sql

test:
	$(GO) test ./... -race -count=1

test-unit:
	$(GO) test ./internal/... -race -count=1

test-integration:
	$(GO) test ./test/integration/... -race -count=1

test-eval:
	$(GO) test ./internal/evals/... -count=1 -run 'TestEvalSmoke|TestDataset' -v

test-policy:
	$(GO) test ./internal/policy/... -count=1 -run 'TestParsePolicyTestCases|TestRunPolicyTests'

test-regression:
	$(GO) test ./internal/evals/... -count=1 -run TestRegressionSuite -v

eval-report:
	$(GO) run ./cmd/eval-report -out eval-report

lint: vet
	@out=$$(gofmt -l . | grep -v '^vendor/' || true); \
	if [ -n "$$out" ]; then echo "gofmt issues:"; echo "$$out"; exit 1; fi

vet:
	$(GO) vet ./...

fmt:
	gofmt -w .

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)

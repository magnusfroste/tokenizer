.PHONY: build dev test test-unit test-integration lint fmt vet tidy clean run run-mock

GO ?= go
BIN_DIR := bin
ROUTER_BIN := $(BIN_DIR)/router
MOCK_BIN := $(BIN_DIR)/mock-provider
WORKER_BIN := $(BIN_DIR)/worker

build:
	$(GO) build -o $(ROUTER_BIN) ./cmd/router
	$(GO) build -o $(MOCK_BIN) ./cmd/mock-provider
	$(GO) build -o $(WORKER_BIN) ./cmd/worker

run: build
	$(ROUTER_BIN)

run-mock: build
	$(MOCK_BIN)

dev:
	$(GO) run ./cmd/router

test:
	$(GO) test ./... -race -count=1

test-unit:
	$(GO) test ./internal/... -race -count=1

test-integration:
	$(GO) test ./test/integration/... -race -count=1

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

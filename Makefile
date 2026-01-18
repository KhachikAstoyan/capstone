SHELL := /bin/bash

# Color output
NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m

# Go settings
GO ?= go
GOFLAGS ?=
CGO_ENABLED ?= 0

# Build settings
BUILD_DIR := build

# Version info
TIMESTAMP := $(shell date +"%Y%m%d.%H%M")
GIT_HASHID := $(shell git rev-parse --short HEAD 2> /dev/null || echo "dev")
VERSION ?= dev
BUILDSTAMP := $(TIMESTAMP)~$(GIT_HASHID)

# Linker flags
GO_LINKER_FLAGS := -ldflags="-s -w -X main.Version=$(VERSION) -X main.Buildstamp=$(BUILDSTAMP)"

# Binary names and sources
API_BINARY := api
API_BINARY_SRC := ./cmd/api
WORKER_BINARY := worker
WORKER_BINARY_SRC := ./cmd/worker

.PHONY: help
help:
	@printf "$(OK_COLOR)Available targets:$(NO_COLOR)\n"
	@printf "  build              Build both api and worker\n"
	@printf "  build-api          Build api binary\n"
	@printf "  build-worker       Build worker binary\n"
	@printf "  clean              Remove build artifacts\n"
	@printf "  test               Run Go tests\n"
	@printf "  lint               Run golangci-lint\n"
	@printf "  fmt                Format Go code\n"
	@printf "  tidy               Go mod tidy\n"
	@printf "  deps               Download dependencies\n"
	@printf "  ci                 Run CI checks (fmt, lint, test)\n"

.PHONY: all
all: clean deps build

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

.PHONY: deps
deps:
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@$(GO) mod download

.PHONY: build
build: build-api build-worker

.PHONY: build-api
build-api: $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building API binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(API_BINARY) $(API_BINARY_SRC)
	@printf "$(OK_COLOR)✅ API binary built: $(BUILD_DIR)/$(API_BINARY)$(NO_COLOR)\n"

.PHONY: build-worker
build-worker: $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building Worker binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(WORKER_BINARY) $(WORKER_BINARY_SRC)
	@printf "$(OK_COLOR)✅ Worker binary built: $(BUILD_DIR)/$(WORKER_BINARY)$(NO_COLOR)\n"

.PHONY: clean
clean:
	@printf "$(OK_COLOR)==> Cleaning build artifacts$(NO_COLOR)\n"
	@$(GO) clean
	@rm -rf $(BUILD_DIR)

.PHONY: test
test:
	@printf "$(OK_COLOR)==> Running tests$(NO_COLOR)\n"
	@$(GO) test $(GOFLAGS) ./...

.PHONY: lint
lint:
	@printf "$(OK_COLOR)==> Running golangci-lint$(NO_COLOR)\n"
	@golangci-lint run ./...

.PHONY: fmt
fmt:
	@printf "$(OK_COLOR)==> Formatting code$(NO_COLOR)\n"
	@$(GO) fmt ./...

.PHONY: tidy
tidy:
	@printf "$(OK_COLOR)==> Tidying dependencies$(NO_COLOR)\n"
	@$(GO) mod tidy

.PHONY: ci
ci: fmt lint test
	@printf "$(OK_COLOR)✅ All CI checks passed$(NO_COLOR)\n"

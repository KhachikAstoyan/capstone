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
BUILD_DIR := ./build

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
EMAIL_BINARY := email
EMAIL_BINARY_SRC := ./cmd/email
CP_BINARY := control-plane
CP_BINARY_SRC := ./cmd/control-plane

.PHONY: help
help:
	@printf "$(OK_COLOR)Available targets:$(NO_COLOR)\n"
	@printf "  build-all             Build api, worker, email, and control-plane binaries\n"
	@printf "  build-api             Build api binary\n"
	@printf "  build-worker          Build worker binary\n"
	@printf "  build-email           Build email worker binary\n"
	@printf "  build-control-plane   Build execution control-plane binary\n"
	@printf "  run-control-plane     Run control-plane (needs CP_DATABASE_URL)\n"
	@printf "  clean                 Remove build artifacts\n"
	@printf "  test                  Run Go tests\n"
	@printf "  fmt                   Format Go code\n"
	@printf "  tidy                  Go mod tidy\n"
	@printf "  deps                  Download dependencies\n"
	@printf "  run-email             Run email worker (needs EMAIL_RABBITMQ_URL)\n"

.PHONY: all
all: clean deps build-all

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

.PHONY: deps
deps:
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@$(GO) mod download

.PHONY: build-all
build-all: build-api build-worker build-email build-control-plane

.PHONY: build-api
build-api: | $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building API binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(API_BINARY) $(API_BINARY_SRC)
	@printf "$(OK_COLOR)✅ API binary built: $(BUILD_DIR)/$(API_BINARY)$(NO_COLOR)\n"

.PHONY: run-api
run-api:
	@printf "$(OK_COLOR)==> Running API binary$(NO_COLOR)\n"
	@$(GO) run $(API_BINARY_SRC)

.PHONY: run-email
run-email:
	@printf "$(OK_COLOR)==> Running Email worker$(NO_COLOR)\n"
	@$(GO) run $(EMAIL_BINARY_SRC)

.PHONY: build-control-plane
build-control-plane: | $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building Control Plane binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(CP_BINARY) $(CP_BINARY_SRC)
	@printf "$(OK_COLOR)✅ Control Plane binary built: $(BUILD_DIR)/$(CP_BINARY)$(NO_COLOR)\n"

.PHONY: run-control-plane
run-control-plane:
	@printf "$(OK_COLOR)==> Running Control Plane$(NO_COLOR)\n"
	@$(GO) run $(CP_BINARY_SRC)

.PHONY: build-worker
build-worker: | $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building Worker binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(WORKER_BINARY) $(WORKER_BINARY_SRC)
	@printf "$(OK_COLOR)✅ Worker binary built: $(BUILD_DIR)/$(WORKER_BINARY)$(NO_COLOR)\n"

.PHONY: build-email
build-email: | $(BUILD_DIR)
	@printf "$(OK_COLOR)==> Building Email worker binary$(NO_COLOR)\n"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -trimpath $(GO_LINKER_FLAGS) -o $(BUILD_DIR)/$(EMAIL_BINARY) $(EMAIL_BINARY_SRC)
	@printf "$(OK_COLOR)✅ Email binary built: $(BUILD_DIR)/$(EMAIL_BINARY)$(NO_COLOR)\n"

.PHONY: clean
clean:
	@printf "$(OK_COLOR)==> Cleaning build artifacts$(NO_COLOR)\n"
	@$(GO) clean
	@rm -rf $(BUILD_DIR)

.PHONY: test
test:
	@printf "$(OK_COLOR)==> Running tests$(NO_COLOR)\n"
	@$(GO) test $(GOFLAGS) ./...

.PHONY: fmt
fmt:
	@printf "$(OK_COLOR)==> Formatting code$(NO_COLOR)\n"
	@$(GO) fmt ./...

.PHONY: tidy
tidy:
	@printf "$(OK_COLOR)==> Tidying dependencies$(NO_COLOR)\n"
	@$(GO) mod tidy

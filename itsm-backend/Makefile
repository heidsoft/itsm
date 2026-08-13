# ITSM Backend Makefile
#
# This file gives backend developers a single, well-defined Go command surface.
# All `go` invocations explicitly set `GOTOOLCHAIN=auto` so that locally
# installed Go (e.g. 1.25.6) can transparently switch to the version
# declared in itsm-backend/go.mod (1.25.12 at the time of writing).
#
# Cross references:
#   - docs/testing/go-toolchain.md — why `GOTOOLCHAIN=auto` is the contract
#   - docs/testing/coverage-report.md — for the unified coverage entry
#   - .github/workflows/backend-ci.yml — uses `actions/setup-go@v7` with
#     `go-version-file: itsm-backend/go.mod`, which installs 1.25.12
#     directly and does NOT need `GOTOOLCHAIN` at all.

SHELL := /bin/bash
GO ?= go
GOTOOLCHAIN ?= auto

# Defaults match the CI workflow; override on the command line as needed:
#   make test BACKEND_TARGETS=./service/...
BACKEND_TARGETS ?= ./...
COVER_PROFILE ?= coverage.out
COVER_MODE    ?= set

# Allowed: GOCACHE/GOMODCACHE override for sandboxed environments; CI sets
# its own cache via actions/cache@v7 so we keep this empty by default.
export GOTOOLCHAIN
export GOFLAGS

# Make sure the toolchain switch is silent in CI logs but explicit locally.
ifeq ($(GOTOOLCHAIN),auto)
$(info [backend-make] Using GOTOOLCHAIN=auto; backend will auto-download 1.25.12 if missing)
endif

.PHONY: help all tidy verify test test-ci test-race test-short cover cover-html vet build run fmt lint clean

help:           ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

all:            ## tidy + verify + vet + test
	$(MAKE) tidy
	$(MAKE) verify
	$(MAKE) vet
	$(MAKE) test

tidy:           ## Run `go mod tidy` (keeps go.sum in sync)
	$(GO) mod tidy

verify:         ## `go mod verify` (sanity check on go.sum)
	$(GO) mod verify

test:           ## Run all Go tests (output to stdout)
	$(GO) test $(BACKEND_TARGETS) -count=1

test-ci:        ## Run all Go tests with coverage (mirrors backend-ci.yml)
	$(GO) list ./... 2>/dev/null \
	  | grep -vE '^itsm-backend/migrations$|^itsm-backend/migrations/hook$|^itsm-backend/migrations/enttest$|^itsm-backend/migrations/predicate$' \
	  | xargs $(GO) test -coverprofile=$(COVER_PROFILE) -covermode=$(COVER_MODE) -count=1
	@if [ -f $(COVER_PROFILE) ]; then $(GO) tool cover -func=$(COVER_PROFILE) | tail -3; fi

test-race:      ## Run all Go tests with `-race`
	$(GO) test $(BACKEND_TARGETS) -race -count=1

test-short:     ## Run only `testing.Short()`-tagged tests
	$(GO) test $(BACKEND_TARGETS) -short -count=1

cover:          ## Generate coverage profile only
	$(GO) test $(BACKEND_TARGETS) -coverprofile=$(COVER_PROFILE) -covermode=$(COVER_MODE) -count=1

cover-html: cover ## Render coverage HTML (open coverage.html)
	$(GO) tool cover -html=$(COVER_PROFILE) -o coverage.html

vet:            ## `go vet ./...` — static analysis
	$(GO) vet $(BACKEND_TARGETS)

build:          ## Build main binary
	$(GO) build -o itsm ./main.go

run:            ## `go run .` (development)
	$(GO) run .

fmt:            ## `gofmt -w` the whole tree (supplement gofumpt CI)
	gofmt -w .

lint:           ## Run staticcheck locally (matches CI)
	$(GO) install honnef.co/go/tools/cmd/staticcheck@v0.6.1
	@bin="$$($(GO) env GOBIN)/staticcheck"; \
	if [ -z "$$bin" ] || [ ! -x "$$bin" ]; then \
	  bin="$$($(GO) env GOPATH)/bin/staticcheck"; \
	fi; \
	$(GO) list ./... | grep -v -e '^itsm-backend/migrations$' -e '^itsm-backend/migrations/' | xargs "$$bin"

clean:          ## Remove build/coverage/test artifacts
	rm -f itsm coverage.out coverage.html
	$(GO) clean -testcache

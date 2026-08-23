ifeq ($(OS),Windows_NT)
SHELL := powershell.exe
.SHELLFLAGS := -NoLogo -NoProfile -Command
endif

GO            ?= go
MODULE        := github.com/SoheilHasankhani/msc-cli
BIN_DIR       := bin
ifeq ($(OS),Windows_NT)
BIN           := $(BIN_DIR)/msc.exe
else
BIN           := $(BIN_DIR)/msc
endif
PKG           := ./cmd/msc
GOLANGCI_LINT := github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0

ifeq ($(OS),Windows_NT)
VERSION_RAW := $(strip $(shell git describe --tags --always --dirty))
COMMIT_RAW  := $(strip $(shell git rev-parse --short HEAD))
VERSION ?= $(if $(VERSION_RAW),$(VERSION_RAW),dev)
COMMIT  ?= $(if $(COMMIT_RAW),$(COMMIT_RAW),none)
DATE    ?= $(shell powershell -NoLogo -NoProfile -Command "[DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')")
else
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
endif

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: help tidy generate build run test cover fmt lint lint-fix ci clean

ifeq ($(OS),Windows_NT)
MKDIR_P = if (-not (Test-Path -LiteralPath '$(1)')) { New-Item -ItemType Directory -Path '$(1)' -Force | Out-Null }
RM_RF = if (Test-Path -LiteralPath '$(1)') { Remove-Item -Recurse -Force -LiteralPath '$(1)' -ErrorAction SilentlyContinue }
else
MKDIR_P = mkdir -p '$(1)'
RM_RF = rm -rf '$(1)'
endif

help:
	@echo "Targets:"
	@echo "  make run ARGS=\"--help\"   Run from source (no install)"
	@echo "  make build               Build ./bin/msc (or bin/msc.exe on Windows)"
	@echo "  make test                Run all unit tests (isolated config home)"
	@echo "  make cover               Unit tests + coverage report"
	@echo "  make fmt                 Auto-fix gofmt/goimports (same as CI formatters)"
	@echo "  make lint                golangci-lint (same version as CI; needs Go only)"
	@echo "  make lint-fix            golangci-lint --fix"
	@echo "  make generate            go generate ./... (mocks)"
	@echo "  make ci                  tidy + generate + test + lint + build"
	@echo "  make clean               Remove local build artifacts"

tidy:
	$(GO) mod tidy

generate:
	$(GO) generate ./...

build:
	$(call MKDIR_P,$(BIN_DIR))
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

run:
	$(GO) run -ldflags "$(LDFLAGS)" $(PKG) $(ARGS)

test:
	$(GO) run scripts/runtests.go

cover:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -func=coverage.out

fmt:
	$(GO) run $(GOLANGCI_LINT) fmt

lint:
	$(GO) run scripts/lint.go

lint-fix:
	$(GO) run $(GOLANGCI_LINT) run --fix

ci: tidy generate test lint build

clean:
	$(call RM_RF,$(BIN_DIR))
	$(call RM_RF,dist)
	$(call RM_RF,coverage.out)
	$(call RM_RF,coverage.html)

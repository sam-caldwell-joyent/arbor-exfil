SHELL := bash

APP_NAME := arbor-exfil
BIN_DIR ?= bin
GO ?= go
DOCKER ?= docker
GIT ?= git
PKGS := ./...
VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
LDFLAGS := -X 'arbor-exfil/cmd.Version=$(VERSION)'

.PHONY: all build test lint fmt tidy clean coverage check-coverage docker help tag/patch tag/minor tag/major

all: lint test build ## Default: lint, test, build

build: $(BIN_DIR) ## Build the binary into bin/
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(APP_NAME) .

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

TEST_COVER_MIN ?= 80.0

test: ## Run unit tests with coverage and enforce minimum threshold
	# Measure coverage on core package(s) only to avoid skew from test utilities
	$(GO) test ./cmd -cover -coverprofile=coverage.out
	# Run full test suite (no coverage aggregation needed)
	$(GO) test ./...
	@$(MAKE) check-coverage


check-coverage: ## Enforce minimum test coverage threshold ($(TEST_COVER_MIN)%)
	@$(GO) tool cover -func=coverage.out | awk -v min=$(TEST_COVER_MIN) '/^total:/ { gsub("%","",$$3); cov=$$3+0; if (cov < min) { printf("Coverage %.1f%% is below threshold %.1f%%\n", cov, min); exit 1 } else { printf("Coverage OK: %.1f%% (min %.1f%%)\n", cov, min) } }'

coverage: test ## Show coverage summary (requires coverage.out from `make test`)
	$(GO) tool cover -func=coverage.out

docker: ## Build local Docker image as $(APP_NAME):local-test
	$(DOCKER) build -t $(APP_NAME):local-test .

lint: ## Lint using go vet
	$(GO) vet -v $(PKGS)

fmt: ## Format sources
	$(GO) fmt $(PKGS)

tidy: ## Sync go.mod/go.sum
	$(GO) mod tidy

clean: ## Clean build and coverage artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_\-\/]+:.*?##/ {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# -------------------------
# Tagging helpers (semver)
# -------------------------
define bump_tag
    @set -euo pipefail; \
    git fetch --tags >/dev/null 2>&1 || true; \
    LATEST="$$($(GIT) describe --tags --abbrev=0 2>/dev/null || echo v0.0.0)"; \
    V="$${LATEST#v}"; \
    MAJOR="$$(( $$(echo "$$V" | awk -F. '{print $$1+0}') ))"; \
    MINOR="$$(( $$(echo "$$V" | awk -F. '{print $$2+0}') ))"; \
    PATCH="$$(( $$(echo "$$V" | awk -F. '{print $$3+0}') ))"; \
    case "$(1)" in \
      patch) PATCH="$$((PATCH + 1))" ;; \
      minor) MINOR="$$((MINOR + 1))"; PATCH=0 ;; \
      major) MAJOR="$$((MAJOR + 1))"; MINOR=0; PATCH=0 ;; \
      *) echo "Unknown bump: $(1)"; exit 1 ;; \
    esac; \
    NEW="v$${MAJOR}.$${MINOR}.$${PATCH}"; \
    echo "Tagging: $${LATEST} -> $${NEW}"; \
    $(GIT) tag "$$NEW"; \
    $(GIT) push origin "$$NEW"
endef

tag/patch: ## Create and push new patch tag (vX.Y.Z+1)
	$(call bump_tag,patch)

tag/minor: ## Create and push new minor tag (vX.Y+1.0)
	$(call bump_tag,minor)

tag/major: ## Create and push new major tag (vX+1.0.0)
	$(call bump_tag,major)

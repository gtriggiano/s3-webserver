BIN := s3-webserver
SHA := $(shell git rev-parse --short=8 HEAD)
VERSION := $(shell git describe --exact-match 2>/dev/null || basename $$(git describe --all 2>/dev/null))
BUILDDATE := $(shell TZ=UTC0 date '+%Y-%m-%dT%H:%M:%SZ')

GO_BUILD_LDFLAGS := \
	-s \
	-w \
	-X github.com/gtriggiano/s3-webserver/pkg/version.Progname=$(BIN) \
	-X github.com/gtriggiano/s3-webserver/pkg/version.Version=$(VERSION) \
	-X github.com/gtriggiano/s3-webserver/pkg/version.Sha=$(SHA) \
	-X github.com/gtriggiano/s3-webserver/pkg/version.BuildDate=$(BUILDDATE)

.PHONY: build
build: ## Build binary
build: fmt vet
	go build -mod=readonly -ldflags "$(GO_BUILD_LDFLAGS)" -o bin/$(BIN) main.go

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt -mod=readonly ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet -mod=readonly -ldflags "$(GO_BUILD_LDFLAGS)" ./...

.PHONY: lint
lint: ## Run linters
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0 run -v --exclude-use-default=false

.PHONY: clean
clean:
	@rm -rf bin

.PHONY: help
help:
	@echo "$(BIN)"
	@echo
	@echo Targets:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9._-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort
GO ?= go
GOTEST ?=  go test
GOFORMAT ?= go fmt
GOLANGCI_LINT ?= golangci-lint

all: build

.PHONY: build
build:
	$(GO) build

.PHONY: test
test:
	@echo "## TEST"
	@$(GOTEST) ./...

.PHONY: lint
lint:
	@echo "## LINT"
	@$(GOLANGCI_LINT) run

.PHONY: format
format:
	@echo "## FORMAT"
	@$(GOFORMAT) ./...

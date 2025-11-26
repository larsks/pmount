GO ?= go
GOTEST ?=  go test
GOFORMAT ?= go fmt
GOLANGCI_LINT ?= golangci-lint

GOFILES = $(shell go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}}{{"\n"}}{{end}}' ./...)

EXE = pmount

PKG = $(shell grep '^module ' go.mod | cut -f2 -d ' ')
VERSION = $(shell git describe --tags 2> /dev/null || echo dev)
COMPILE =	go build -o $@ -ldflags '-X $(PKG)/internal/version.Version=$(VERSION)'

all: $(EXE)

$(EXE): $(GOFILES)
	$(COMPILE) ./cmd/pmount/

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

.PHONY: clean
clean:
	rm -f pmount

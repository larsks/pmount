GO ?= go
GOTEST ?=  go test
GOFORMAT ?= go fmt
GOLANGCI_LINT ?= golangci-lint

GOFILES = $(shell go list -f '{{range .GoFiles}}{{$$.Dir}}/{{.}}{{"\n"}}{{end}}' ./...)

EXE = pmount

all: $(EXE)

$(EXE): $(GOFILES)
	$(GO) build -o $@ ./cmd/pmount

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

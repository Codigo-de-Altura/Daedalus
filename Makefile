# Daedalus developer Makefile.
#
# Thin wrappers around the Go toolchain so local development and CI share the
# same entry points. `make setup` is the single onboarding command for a clean
# clone; the rest are the standard build/test/lint/run loop.

BINARY := daedalus
CMD    := ./cmd/daedalus
PKG    := ./...

.PHONY: build test lint run fmt tidy setup help

build: ## Compile the daedalus binary
	go build -o $(BINARY) $(CMD)

test: ## Run the test suite
	go test $(PKG)

lint: ## Check formatting (gofmt) and run go vet
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needs to run on:"; echo "$$unformatted"; exit 1; \
	fi
	go vet $(PKG)

run: build ## Build and run the binary
	./$(BINARY)

fmt: ## Format the codebase in place
	gofmt -w .

tidy: ## Tidy and verify module dependencies
	go mod tidy

setup: ## One-command onboarding from a clean clone (deps + build)
	go mod download
	$(MAKE) build

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) \
		| awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

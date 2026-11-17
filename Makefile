GO ?= go
SPEC ?= spec/openapi.json
NORMALIZED ?= spec/openapi.normalized.json

.PHONY: all generate test test-race test-e2e vet fmt-check build clean

all: generate test build

## generate: normalize the vendored OpenAPI document and regenerate ./api.
generate:
	$(GO) run ./cmd/specfix -in $(SPEC) -out $(NORMALIZED)
	$(GO) tool oapi-codegen -config oapi-codegen.yaml $(NORMALIZED)
	rm -f $(NORMALIZED)

## test: run all unit tests.
test:
	$(GO) test ./...

## test-race: run all unit tests with the race detector.
test-race:
	$(GO) test -race ./...

## test-e2e: run read-only end-to-end tests against the live shop.
## Loads .env when it exists; see .env.example.
test-e2e:
	@if [ -f .env ]; then set -a; . ./.env; set +a; fi; \
	$(GO) test -tags=e2e -count=1 -v ./e2e

## vet: run go vet.
vet:
	$(GO) vet ./...

## fmt-check: fail when a file is not gofmt'ed.
fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		echo "gofmt needed for:"; echo "$$files"; exit 1; \
	fi

## build: build the CLI into ./bin/strombauer.
build:
	$(GO) build -o bin/strombauer ./cmd/strombauer

## clean: remove build output and the normalized spec.
clean:
	rm -rf bin $(NORMALIZED)

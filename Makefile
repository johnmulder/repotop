GO ?= go

.PHONY: build check fmt-check test vet

check: fmt-check vet test build

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then \
		printf 'unformatted Go files:\n%s\n' "$$files"; \
		exit 1; \
	fi

vet:
	$(GO) vet ./...

test:
	$(GO) test -count=1 -race ./...

build:
	$(GO) build ./...

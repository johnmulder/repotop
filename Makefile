GO ?= go
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)
RELEASE_VERSION ?= devel
DIST ?= dist
export GOOS GOARCH RELEASE_VERSION DIST

.PHONY: build check fmt-check release test vet

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

release:
	@printf '%s\n' "$$RELEASE_VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$$' || { printf 'RELEASE_VERSION must match vMAJOR.MINOR.PATCH\n' >&2; exit 1; }
	@mkdir -p "$$DIST"
	@set -eu; \
	stage="$$(mktemp -d)"; \
	trap 'rm -rf "$$stage"' 0; \
	name="repotop-$${RELEASE_VERSION}-$${GOOS}-$${GOARCH}"; \
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w -X main.version=$${RELEASE_VERSION}" -o "$$stage/repotop" .; \
	"$$stage/repotop" --help; \
	test "$$("$$stage/repotop" --version)" = "repotop $${RELEASE_VERSION}"; \
	tar -C "$$stage" -czf "$$DIST/$$name.tar.gz" repotop

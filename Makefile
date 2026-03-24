
# Makefile for TFX

.DEFAULT_GOAL := test

GO_TEST_FLAGS := -covermode=atomic -coverprofile=coverage.out
GO_RACE_FLAGS := -race
GO_VERBOSE    := -v
COVERAGE_THRESHOLD := 80
VERSION ?= $(shell cat VERSION 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)
GOLANGCI_LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4
PREFIX ?= $(HOME)/.local
RELEASE_DIR ?= release
ARCHIVE_GOOS ?= $(shell go env GOOS)
ARCHIVE_GOARCH ?= $(shell go env GOARCH)

.PHONY: test test-verbose test-race coverage check-coverage clean demo build-demo build-tfx build-binaries verify release-artifacts release-archive release-archives install-local print-version

test:
	go test ./... -short $(GO_TEST_FLAGS)

#  Test only one package
test-one:
	@echo "Usage: make test-one PKG=path/to/package"
	@echo "Example: make test-one PKG=./logfx"
	@echo "Running tests for package: $(PKG)"
	go test $(PKG) $(GO_TEST_FLAGS)

test-verbose:
	go test ./... $(GO_TEST_FLAGS) $(GO_VERBOSE)

test-race:
	go test ./... $(GO_TEST_FLAGS) $(GO_RACE_FLAGS)

coverage:
	go tool cover -html=coverage.out

check-coverage:
	@coverage=$$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$$3); print $$3}'); \
	echo "Total coverage: $$coverage%"; \
	awk -v cov="$$coverage" -v threshold="$(COVERAGE_THRESHOLD)" 'BEGIN { exit((cov + 0) < (threshold + 0) ? 1 : 0) }'

clean:
	rm -f coverage.out
	rm -f bin/demo
	rm -f bin/tfx
	rm -rf dist
	rm -rf release

print-version:
	@printf "%s\n" "$(VERSION)"

verify:
	bash tools/test.sh
	bash tools/test.sh check
	go build ./...
	go vet ./...
	$(GOLANGCI_LINT) run --timeout=5m

build-tfx:
	mkdir -p bin
	go build -ldflags '$(LDFLAGS)' -o bin/tfx ./cmd/tfx

build-demo:
	mkdir -p bin
	go build -o bin/demo ./cmd/demo

build-binaries: build-tfx build-demo

install-local: build-tfx
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	cp bin/tfx "$(DESTDIR)$(PREFIX)/bin/tfx"

demo: build-demo
	./bin/demo

release-archive:
	VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" \
		bash tools/release-package.sh "$(VERSION)" "$(ARCHIVE_GOOS)" "$(ARCHIVE_GOARCH)" "$(RELEASE_DIR)"

release-archives:
	rm -rf "$(RELEASE_DIR)"
	mkdir -p "$(RELEASE_DIR)"
	for target in \
		"linux amd64" \
		"linux arm64" \
		"darwin amd64" \
		"darwin arm64" \
		"windows amd64" \
		"windows arm64"; do \
		set -- $$target; \
		VERSION="$(VERSION)" COMMIT="$(COMMIT)" BUILD_DATE="$(BUILD_DATE)" \
			bash tools/release-package.sh "$(VERSION)" "$$1" "$$2" "$(RELEASE_DIR)"; \
	done
	@if ls "$(RELEASE_DIR)"/*.tar.gz "$(RELEASE_DIR)"/*.zip >/dev/null 2>&1; then \
		shasum -a 256 "$(RELEASE_DIR)"/*.tar.gz "$(RELEASE_DIR)"/*.zip > "$(RELEASE_DIR)"/SHA256SUMS 2>/dev/null || \
		sha256sum "$(RELEASE_DIR)"/*.tar.gz "$(RELEASE_DIR)"/*.zip > "$(RELEASE_DIR)"/SHA256SUMS; \
	fi

release-artifacts:
	rm -rf dist
	mkdir -p dist
	printf "%s\n" "$(VERSION)" > dist/VERSION
	go build -ldflags '$(LDFLAGS)' -o dist/tfx ./cmd/tfx
	go build -o dist/demo ./cmd/demo
	cp README.md CHANGELOG.md LICENSE dist/
	cp docs/release-notes-$(VERSION).md dist/RELEASE_NOTES.md
	@if command -v shasum >/dev/null 2>&1; then \
		shasum -a 256 dist/tfx dist/demo > dist/SHA256SUMS; \
	else \
		sha256sum dist/tfx dist/demo > dist/SHA256SUMS; \
	fi

fix:
	goimports -w .
	gofumpt -w .
	gci write -s standard -s default -s "prefix($(shell go list -m))" .
	go mod tidy
	go run golang.org/x/tools/gopls/internal/analysis/modernize/cmd/modernize@latest -fix ./...
# 	golangci-lint run --fix || true

tidy:
	go mod tidy

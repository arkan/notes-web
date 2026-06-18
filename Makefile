APP := notes-web
PKG := ./cmd/notes-web
GO_TEST_PKGS := ./cmd/... ./internal/...
BIN_DIR ?= $(HOME)/.local/bin
GO ?= go

.PHONY: run build install clean deps deps-ci test-go lint test-e2e test

run:
	$(GO) run $(PKG)

build:
	$(GO) build -o bin/$(APP) $(PKG)

install:
	install -d $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP) $(PKG)

clean:
	rm -rf bin

deps:
	npm ci
	npx playwright install chromium

deps-ci:
	npm ci
	npx playwright install --with-deps chromium

test-go:
	$(GO) test $(GO_TEST_PKGS)

lint:
	npm run lint

test-e2e:
	npm run test:e2e

test: test-go lint test-e2e

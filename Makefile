APP := notes-web
PKG := ./cmd/notes-web
BIN_DIR ?= $(HOME)/.local/bin
GO ?= go

.PHONY: run build install clean

run:
	$(GO) run $(PKG)

build:
	$(GO) build -o bin/$(APP) $(PKG)

install:
	install -d $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP) $(PKG)

clean:
	rm -rf bin

# bub — Makefile
#
# Common tasks. Run `make` or `make help` to list targets.

BINARY      := bub
PKG         := ./...
INSTALL_DIR := $(shell go env GOPATH)/bin

# Args passed to `make run`, e.g.  make run ARGS="work 25"
ARGS ?=

.DEFAULT_GOAL := help

## help: show this help
.PHONY: help
help:
	@echo "bub — available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/## //' | awk -F': ' '{ printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2 }'

## build: compile the binary into ./$(BINARY)
.PHONY: build
build:
	go build -o $(BINARY) .

## run: build and run, pass flags with ARGS="..."  (e.g. make run ARGS="work 1")
.PHONY: run
run:
	go run . $(ARGS)

## install: install $(BINARY) into $(INSTALL_DIR)
.PHONY: install
install:
	go install .

## uninstall: remove the installed binary
.PHONY: uninstall
uninstall:
	rm -f "$(INSTALL_DIR)/$(BINARY)"

## tidy: sync go.mod / go.sum with the source
.PHONY: tidy
tidy:
	go mod tidy

## fmt: gofmt all packages
.PHONY: fmt
fmt:
	go fmt $(PKG)

## vet: run go vet
.PHONY: vet
vet:
	go vet $(PKG)

## test: run the test suite
.PHONY: test
test:
	go test $(PKG)

## check: fmt, vet, and test in one go
.PHONY: check
check: fmt vet test

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY)
	go clean

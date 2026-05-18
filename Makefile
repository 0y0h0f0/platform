SHELL := /usr/bin/env bash
LOCAL_BIN := $(HOME)/go/bin

GO ?= go
APP_ENV ?= local
CONFIG_FILE ?=
BUF_IMAGE ?= bufbuild/buf:1.59.0
LINT_IMAGE ?= golangci/golangci-lint:v1.64.8

BUF_BIN := $(shell if [ -x "$(LOCAL_BIN)/buf" ]; then echo "$(LOCAL_BIN)/buf"; else command -v buf 2>/dev/null; fi)
LINT_BIN := $(shell if [ -x "$(LOCAL_BIN)/golangci-lint" ]; then echo "$(LOCAL_BIN)/golangci-lint"; else command -v golangci-lint 2>/dev/null; fi)

ifeq ($(BUF_BIN),)
BUF := docker run --rm -v "$(CURDIR):/workspace" -w /workspace $(BUF_IMAGE)
else
BUF := $(BUF_BIN)
endif

ifeq ($(LINT_BIN),)
LINT := docker run --rm -v "$(CURDIR):/app" -w /app $(LINT_IMAGE)
else
LINT := $(LINT_BIN)
endif

.PHONY: proto proto-lint run/% test lint coverage migrate up down

proto:
	$(BUF) generate

proto-lint:
	$(BUF) lint

run/%:
	APP_ENV=$(APP_ENV) CONFIG_FILE="$(CONFIG_FILE)" $(GO) run ./cmd/$*

test:
	$(GO) test ./...

lint:
	$(LINT) run ./...

coverage:
	./scripts/coverage.sh

migrate:
	./scripts/run-migrations.sh

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

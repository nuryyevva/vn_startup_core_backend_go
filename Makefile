SHELL := /bin/bash

MIGRATIONS_DIR := migrations
DB_DSN ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(or $(POSTGRES_HOST),localhost):$(or $(POSTGRES_PORT),5432)/$(or $(POSTGRES_DB),vn_platform)?sslmode=$(or $(POSTGRES_SSLMODE),disable)

include .env
export

.PHONY: up down run run-mockai seed build sqlc-generate migrate migrate-down migrate-force test test-unit test-integration lint tidy

up:
	docker-compose up -d

down:
	docker-compose down

build:
	go build -o bin/api ./cmd/api

run: build
	./bin/api

# run-mockai starts the throwaway AI Orchestrator stand-in (cmd/mockai) used
# to exercise the WebSocket path end-to-end without the real Python service.
# Run it alongside `make run` — see README's "Проверка WebSocket-пути".
run-mockai:
	go run ./cmd/mockai

# seed inserts a couple of sample stories (with scenes and choices) into
# the database so every endpoint can be exercised manually — there's no
# admin panel at this stage. Safe to run more than once (idempotent).
seed:
	go run ./cmd/seed

sqlc-generate:
	sqlc generate

migrate:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

migrate-force:
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" force $(version)

test: test-unit test-integration

test-unit:
	go test ./... -short -count=1

test-integration:
	go test -tags integration -p 1 ./... -run Integration -count=1

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

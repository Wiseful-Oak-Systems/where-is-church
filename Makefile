.PHONY: run build dev db-up db-down up down test test-verbose lint fmt clean docker-build logs coverage help

## help: Show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'

## up: Start the full stack (db + app) with Docker Compose
up:
	docker compose up -d --build

## down: Stop all services
down:
	docker compose down

## logs: Tail logs from all services
logs:
	docker compose logs -f

## run: Start the server locally (requires db-up first)
run:
	go run ./cmd/server

## build: Build the binary to bin/
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/whereischurch ./cmd/server

## dev: Start database only, then run the server locally
dev: db-up run

## db-up: Start only the database
db-up:
	docker compose up -d db

## db-down: Stop the database
db-down:
	docker compose down db

## test: Run all tests
test:
	CGO_ENABLED=1 go test ./internal/... -count=1

## test-verbose: Run all tests with verbose output
test-verbose:
	CGO_ENABLED=1 go test -v ./internal/... -count=1

## test-race: Run tests with race detector
test-race:
	CGO_ENABLED=1 go test -race ./internal/... -count=1

## coverage: Run tests with coverage report
coverage:
	CGO_ENABLED=1 go test -coverprofile=coverage.out -covermode=atomic ./internal/...
	go tool cover -func=coverage.out
	@echo "\nTo view HTML report: go tool cover -html=coverage.out"

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## fmt: Format all Go files
fmt:
	gofmt -s -w .
	goimports -w .

## migrate-up: Run database migrations (requires golang-migrate CLI)
migrate-up:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

## migrate-down: Rollback last migration
migrate-down:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down 1

## migrate-create: Create a new migration (usage: make migrate-create NAME=add_feature)
migrate-create:
	migrate create -ext sql -dir migrations -seq $(NAME)

## clean: Remove build artifacts
clean:
	rm -rf bin/ server coverage.out coverage.*

## docker-build: Build Docker image only
docker-build:
	docker build -t where-is-church:latest .

## seed: Load seed files into database (or download from OSM if no files)
seed:
	go run ./cmd/seed

## seed-gen: Download churches from OSM and save as seed files (run on your machine)
seed-gen:
	go run ./cmd/seedgen BR

## seed-gen-all: Download all supported countries
seed-gen-all:
	go run ./cmd/seedgen ALL

## seed-load: Load all seed files from seeds/ into database
seed-load: seed

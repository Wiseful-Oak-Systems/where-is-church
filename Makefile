.PHONY: run build dev db-up db-down test test-verbose lint fmt clean docker-build docker-run coverage help

## help: Show this help message
help:
	@grep -E '^## ' Makefile | sed 's/## //'

## run: Start the server
run:
	go run ./cmd/server

## build: Build the binary to bin/
build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/whereischurch ./cmd/server

## dev: Start database and server
dev: db-up run

## db-up: Start PostgreSQL with PostGIS
db-up:
	docker compose up -d

## db-down: Stop PostgreSQL
db-down:
	docker compose down

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

## clean: Remove build artifacts
clean:
	rm -rf bin/ server coverage.out coverage.*

## docker-build: Build Docker image
docker-build:
	docker build -t where-is-church:latest .

## docker-run: Run the full stack with Docker Compose
docker-run: db-up
	docker compose up -d

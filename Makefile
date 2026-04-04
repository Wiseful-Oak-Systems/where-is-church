.PHONY: run build dev db-up db-down

run:
	go run ./cmd/server

build:
	go build -o bin/whereischurch ./cmd/server

dev: db-up run

db-up:
	docker compose up -d

db-down:
	docker compose down

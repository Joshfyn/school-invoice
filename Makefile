.PHONY: help build run test clean docker-up docker-down migrate-up migrate-down

# Default target
help:
	@echo "School Invoice API - Available commands:"
	@echo "  make build        - Build the application"
	@echo "  make run          - Run the application locally"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make docker-up    - Start all services with Docker"
	@echo "  make docker-down  - Stop all Docker services"
	@echo "  make docker-build - Build Docker images"
	@echo "  make migrate-up   - Run database migrations"
	@echo "  make migrate-down - Rollback database migrations"
	@echo "  make db-shell     - Open PostgreSQL shell"

# Build the application
build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

# Run the application locally
run:
	go run ./cmd/api

# Run worker locally
run-worker:
	go run ./cmd/worker

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Docker commands
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f

# Database commands
db-shell:
	docker-compose exec db psql -U postgres -d school_invoice

# Migration commands (using golang-migrate)
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir ./migrations -seq $$name

# Development helpers
dev:
	air -c .air.toml

# Install development tools
tools:
	go install github.com/air-verse/air@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

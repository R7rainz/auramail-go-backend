# AuraMail Go Backend Makefile

.PHONY: all build run test clean migrate dev

# Default target
all: build

# Build the backend binary
build:
	go build -o backend ./cmd/backend/

# Run the backend in development mode
run:
	go run ./cmd/backend/

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Air not installed. Running with go run instead..."; \
		go run ./cmd/backend/; \
	fi

# Run all tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f backend coverage.out coverage.html

# Database migrations (requires goose: go install github.com/pressly/goose/v3/cmd/goose@latest)
migrate-up:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" up

migrate-down:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" down

migrate-status:
	goose -dir migrations postgres "$(GOOSE_DBSTRING)" status

migrate-create:
	@read -p "Migration name: " name; \
	goose -dir migrations create $$name sql

# Install development tools
tools:
	go install github.com/cosmtrek/air@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest

# Tidy dependencies
tidy:
	go mod tidy

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Format code
fmt:
	go fmt ./...

# Lint (requires golangci-lint)
lint:
	golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build the backend binary"
	@echo "  run            - Run the backend"
	@echo "  dev            - Run with hot reload (requires air)"
	@echo "  test           - Run all tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  migrate-up     - Run database migrations"
	@echo "  migrate-down   - Rollback last migration"
	@echo "  migrate-status - Show migration status"
	@echo "  migrate-create - Create a new migration"
	@echo "  tools          - Install development tools"
	@echo "  tidy           - Tidy go.mod"
	@echo "  docker-up      - Start Docker containers"
	@echo "  docker-down    - Stop Docker containers"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"

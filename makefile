# Makefile
.PHONY: build test run docker-up docker-down clean lint benchmark deps

# Build the application
build:
	go build -o bin/server cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Run the application
run:
	go run cmd/server/main.go

# Start development environment
docker-up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Services are ready!"

# Stop development environment  
docker-down:
	docker-compose down

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Run linter
lint:
	golangci-lint run

# Run benchmarks
benchmark:
	go test -bench=. -benchmem ./internal/ratelimiter/algorithms/

# Install/update dependencies
deps:
	go mod download
	go mod tidy

# Development with hot reload
dev:
	air

# Database migrations
migrate-up:
	go run cmd/migrate/main.go up

migrate-down:
	go run cmd/migrate/main.go down
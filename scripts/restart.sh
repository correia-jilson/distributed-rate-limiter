#!/bin/bash

echo "🚀 Starting Distributed Rate Limiter..."
echo "======================================"

# Check prerequisites
echo "🔍 Checking prerequisites..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+"
    exit 1
fi

# Check if Docker is running
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker"
    exit 1
fi

# Check if we're in the right directory
if [ ! -f "go.mod" ] || [ ! -f "docker-compose.yml" ]; then
    echo "❌ Not in project directory. Please cd to distributed-rate-limiter"
    exit 1
fi

# Start Docker services
echo "🐳 Starting Docker services..."
docker-compose up -d

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 5

# Check Redis connectivity
echo "🔴 Testing Redis connection..."
REDIS_RETRIES=0
while [ $REDIS_RETRIES -lt 10 ]; do
    if redis-cli ping >/dev/null 2>&1; then
        echo "   ✅ Redis connected"
        break
    fi
    REDIS_RETRIES=$((REDIS_RETRIES + 1))
    echo "   ⏳ Attempt $REDIS_RETRIES/10..."
    sleep 2
done

if [ $REDIS_RETRIES -eq 10 ]; then
    echo "   ❌ Redis connection failed"
    docker-compose logs redis
    exit 1
fi

# Install/update Go dependencies
echo "📦 Updating Go dependencies..."
go mod download
go mod tidy

# Start the Rate Limiter service
echo "🎯 Starting Rate Limiter service..."
echo "   Server will start on http://localhost:8080"
echo "   Press Ctrl+C to stop"
echo ""
echo "🔗 Useful endpoints:"
echo "   Health: curl http://localhost:8080/health"
echo "   Rate Limit: curl -X POST http://localhost:8080/api/v1/ratelimit -d '{\"key\":\"test\",\"limit\":10,\"window\":60,\"algorithm\":\"token_bucket\"}'"
echo ""
echo "========================================"

# Start the Go server
echo "🏃 Starting Go server..."
go run cmd/server/main.go

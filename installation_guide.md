# 🛠️ Installation Requirements & Setup Guide

## 📋 Prerequisites

### Required Software

#### Go Programming Language
```bash
# Version Required: Go 1.21 or higher

# Installation Options:

# macOS (using Homebrew)
brew install go

# Linux (Ubuntu/Debian)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Windows
# Download installer from: https://golang.org/dl/

# Verify Installation
go version  # Should show: go version go1.21.x
```

#### Redis Database
```bash
# Version Required: Redis 6.0 or higher

# macOS (using Homebrew)
brew install redis
brew services start redis

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Windows
# Option 1: Download from https://redis.io/download
# Option 2: Use Docker (recommended)
docker run -d -p 6379:6379 --name redis redis:alpine

# Verify Installation
redis-cli ping  # Should return: PONG
```

#### Git Version Control
```bash
# macOS
brew install git

# Linux (Ubuntu/Debian)  
sudo apt install git

# Windows
# Download from: https://git-scm.com/download/win

# Verify Installation
git --version  # Should show version info
```

### Optional (Recommended) Software

#### Docker & Docker Compose
```bash
# macOS
brew install --cask docker
# Start Docker Desktop manually

# Linux (Ubuntu/Debian)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Log out and back in

# Windows
# Download Docker Desktop from: https://www.docker.com/products/docker-desktop/

# Verify Installation
docker --version
docker-compose --version
```

#### Node.js (for future dashboard development)
```bash
# macOS
brew install node

# Linux (Ubuntu/Debian)
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Windows
# Download from: https://nodejs.org/

# Verify Installation
node --version
npm --version
```

#### PostgreSQL (for metrics storage)
```bash
# macOS
brew install postgresql
brew services start postgresql

# Linux (Ubuntu/Debian)
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Windows
# Download from: https://www.postgresql.org/download/windows/

# Verify Installation
psql --version
```

## 🔧 Development Environment Setup

### 1. Create Project Directory
```bash
# Create and navigate to project directory
mkdir distributed-rate-limiter
cd distributed-rate-limiter

# Initialize Git repository
git init
```

### 2. Initialize Go Module
```bash
# Replace YOUR_USERNAME with your GitHub username
go mod init github.com/YOUR_USERNAME/distributed-rate-limiter

# Verify go.mod creation
cat go.mod
```

### 3. Install Go Dependencies
```bash
# Core application dependencies
go get github.com/gin-gonic/gin@latest          # HTTP web framework
go get github.com/redis/go-redis/v9@latest      # Redis client
go get github.com/sirupsen/logrus@latest        # Structured logging
go get github.com/lib/pq@latest                 # PostgreSQL driver
go get github.com/spf13/viper@latest            # Configuration management

# Development dependencies
go get github.com/stretchr/testify@latest       # Testing framework
go get github.com/golang-migrate/migrate/v4@latest # Database migrations

# Clean up dependencies
go mod tidy

# Verify dependencies
go mod download
```

### 4. Create Project Structure
```bash
# Create directory structure
mkdir -p cmd/server
mkdir -p internal/api
mkdir -p internal/ratelimiter/algorithms
mkdir -p internal/ratelimiter/storage  
mkdir -p internal/config
mkdir -p internal/metrics
mkdir -p internal/health
mkdir -p pkg/sdk/go
mkdir -p tests/unit
mkdir -p tests/integration
mkdir -p tests/performance
mkdir -p configs
mkdir -p docs
mkdir -p scripts
mkdir -p deployments/docker

# Verify structure
tree . -d  # Shows directory tree
```

### 5. Configure Development Services

#### Redis Configuration
```bash
# Option 1: Local Redis
redis-server  # Start Redis server

# Option 2: Docker Redis (recommended)
docker run -d \
  --name redis-rate-limiter \
  -p 6379:6379 \
  redis:alpine

# Test Redis connection
redis-cli ping  # Should return: PONG
```

#### PostgreSQL Setup (for metrics)
```bash
# Create database user and database
sudo -u postgres psql << EOF
CREATE USER ratelimiter WITH PASSWORD 'password123';
CREATE DATABASE ratelimiter OWNER ratelimiter;
GRANT ALL PRIVILEGES ON DATABASE ratelimiter TO ratelimiter;
\q
EOF

# Test connection
psql -h localhost -U ratelimiter -d ratelimiter -c "SELECT version();"
```

### 6. Environment Configuration
```bash
# Create environment file
cat > .env << EOF
# Server Configuration
PORT=8080
GIN_MODE=debug

# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# PostgreSQL Configuration  
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=ratelimiter
POSTGRES_PASSWORD=password123
POSTGRES_DB=ratelimiter

# Logging Configuration
LOG_LEVEL=info
LOG_FORMAT=json
EOF

# Add .env to .gitignore
echo ".env" >> .gitignore
```

## 🧪 Installation Verification

### System Requirements Check
```bash
# Create verification script
cat > scripts/verify-setup.sh << 'EOF'
#!/bin/bash

echo "🔍 Verifying Installation Requirements..."

# Check Go
if command -v go &> /dev/null; then
    echo "✅ Go: $(go version)"
else
    echo "❌ Go: Not installed"
    exit 1
fi

# Check Redis
if command -v redis-cli &> /dev/null; then
    if redis-cli ping &> /dev/null; then
        echo "✅ Redis: Connected and running"
    else
        echo "❌ Redis: Not running"
        exit 1
    fi
else
    echo "❌ Redis: Not installed"
    exit 1
fi

# Check Git
if command -v git &> /dev/null; then
    echo "✅ Git: $(git --version)"
else
    echo "❌ Git: Not installed"
    exit 1
fi

# Check PostgreSQL
if command -v psql &> /dev/null; then
    echo "✅ PostgreSQL: $(psql --version)"
else
    echo "⚠️  PostgreSQL: Not installed (optional for Phase 1)"
fi

# Check Docker
if command -v docker &> /dev/null; then
    echo "✅ Docker: $(docker --version)"
else
    echo "⚠️  Docker: Not installed (optional)"
fi

# Check Node.js
if command -v node &> /dev/null; then
    echo "✅ Node.js: $(node --version)"
else
    echo "⚠️  Node.js: Not installed (needed for Phase 3)"
fi

echo ""
echo "🎯 Phase 1 Requirements: Complete!"
echo "🚀 Ready to build the distributed rate limiter!"
EOF

# Make script executable
chmod +x scripts/verify-setup.sh

# Run verification
./scripts/verify-setup.sh
```

### Go Dependencies Verification
```bash
# Verify all dependencies are properly installed
go mod verify

# List all dependencies
go list -m all

# Check for security vulnerabilities
go list -json -m all | nancy sleuth

# Verify build works
go build -o bin/test cmd/server/main.go
```

### Service Connectivity Test
```bash
# Test Redis connectivity
cat > scripts/test-redis.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "github.com/redis/go-redis/v9"
)

func main() {
    rdb := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    ctx := context.Background()
    pong, err := rdb.Ping(ctx).Result()
    if err != nil {
        fmt.Printf("❌ Redis connection failed: %v\n", err)
        return
    }
    
    fmt.Printf("✅ Redis connected: %s\n", pong)
}
EOF

# Run Redis test
go run scripts/test-redis.go

# Clean up test file
rm scripts/test-redis.go
```

## 🐳 Docker Development Environment

### Docker Compose Setup
```yaml
# docker-compose.yml
version: '3.8'

services:
  redis:
    image: redis:7-alpine  
    container_name: redis-rate-limiter
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3

  postgres:
    image: postgres:15-alpine
    container_name: postgres-rate-limiter
    environment:
      POSTGRES_USER: ratelimiter
      POSTGRES_PASSWORD: password123
      POSTGRES_DB: ratelimiter
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ratelimiter -d ratelimiter"]
      interval: 10s
      timeout: 5s
      retries: 3

volumes:
  redis_data:
  postgres_data:
```

### Development Commands
```bash
# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Clean up (removes data)
docker-compose down -v
```

## 🛠️ Development Tools Setup

### VS Code Configuration
```json
// .vscode/settings.json
{
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "go.buildTags": "integration",
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
        "source.organizeImports": true
    }
}
```

### Recommended VS Code Extensions
```json
// .vscode/extensions.json
{
    "recommendations": [
        "golang.go",
        "ms-vscode.vscode-json",
        "redhat.vscode-yaml",
        "humao.rest-client",
        "ms-azuretools.vscode-docker"
    ]
}
```

### Git Configuration
```bash
# Configure Git (if not already done)
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"

# Create .gitignore
cat > .gitignore << EOF
# Binaries
bin/
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary
*.test
*.out

# Go workspace file
go.work
go.work.sum

# Environment variables
.env
.env.local
.env.*.local

# IDE files
.vscode/
.idea/
*.swp
*.swo

# OS files
.DS_Store
Thumbs.db

# Logs
*.log

# Dependencies
vendor/

# Build artifacts
dist/
build/

# Temporary files
tmp/
temp/
EOF
```

## 🚀 Quick Start Verification

### Complete Setup Test
```bash
# 1. Clone or create project
git clone https://github.com/YOUR_USERNAME/distributed-rate-limiter.git
cd distributed-rate-limiter

# 2. Install dependencies
go mod download

# 3. Start services
docker-compose up -d

# 4. Wait for services to be ready
sleep 10

# 5. Build and run the rate limiter
go run cmd/server/main.go &

# 6. Test the API
curl -X POST http://localhost:8080/api/v1/ratelimit \
  -H "Content-Type: application/json" \
  -d '{
    "key": "test_user",
    "limit": 10,
    "window": 60,
    "algorithm": "token_bucket"
  }'

# Expected response:
# {"allowed":true,"remaining":9,"reset_time":1234567890,"algorithm":"token_bucket","tokens":9}
```

## 🔧 Troubleshooting Common Issues

### Go Installation Issues
```bash
# Check GOPATH and GOROOT
go env GOPATH
go env GOROOT

# Fix PATH issues
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin

# Clear module cache if corrupted
go clean -modcache
```

### Redis Connection Issues
```bash
# Check if Redis is running
redis-cli ping

# Check Redis port
netstat -an | grep 6379

# Start Redis manually
redis-server /usr/local/etc/redis.conf

# Docker Redis alternative
docker run -d -p 6379:6379 redis:alpine
```

### Permission Issues (Linux/macOS)
```bash
# Fix Docker permissions
sudo usermod -aG docker $USER
newgrp docker

# Fix Redis permissions
sudo chown -R $USER:$USER /var/lib/redis
```

### Port Conflicts
```bash
# Check what's using port 8080
sudo lsof -i :8080

# Kill process using port
sudo kill -9 $(lsof -t -i:8080)

# Use different port
export PORT=8081
```

## 📚 Learning Resources

### Go Resources
- [Official Go Documentation](https://golang.org/doc/)
- [Go by Example](https://gobyexample.com/)
- [Effective Go](https://golang.org/doc/effective_go.html)

### Redis Resources
- [Redis Documentation](https://redis.io/documentation)
- [Redis University](https://university.redis.com/)
- [Redis Lua Scripting](https://redis.io/commands/eval)

### Distributed Systems
- [Designing Data-Intensive Applications](https://dataintensive.net/)
- [System Design Primer](https://github.com/donnemartin/system-design-primer)
- [CAP Theorem Explained](https://en.wikipedia.org/wiki/CAP_theorem)

---

**This completes the installation requirements for Phase 1. All prerequisites should be installed and verified before proceeding with the rate limiter implementation.**

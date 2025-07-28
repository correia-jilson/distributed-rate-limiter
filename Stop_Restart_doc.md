# 🔄 Shutdown & Restart Guide

## 🛑 Complete Shutdown Process

### Quick Shutdown (Recommended)
```bash
# Navigate to project directory
cd distributed-rate-limiter

# Run the automated shutdown script
./scripts/shutdown.sh
```

### Manual Shutdown Steps

#### 1. Stop the Rate Limiter Service
```bash
# Method 1: If running in foreground terminal
# Press Ctrl+C in the terminal where server is running

# Method 2: If running in background
pkill -f "go run cmd/server/main.go"

# Method 3: Kill by port (if needed)
kill -9 $(lsof -t -i:8080)

# Verify service is stopped
lsof -i :8080  # Should return no results
curl http://localhost:8080/health  # Should fail to connect
```

#### 2. Stop Docker Services
```bash
# Stop containers but keep data volumes
docker-compose down

# Alternative: Stop and remove everything including data
docker-compose down -v  # ⚠️ This deletes all Redis/PostgreSQL data

# Verify containers are stopped
docker-compose ps  # Should show no running services
docker ps  # Should show no containers (or none related to project)
```

#### 3. Stop Local Services (if not using Docker)
```bash
# Stop Redis (if running locally)
redis-cli shutdown
# OR for macOS with Homebrew
brew services stop redis
# OR for Linux systemd
sudo systemctl stop redis-server

# Stop PostgreSQL (if running locally) 
brew services stop postgresql  # macOS
sudo systemctl stop postgresql  # Linux
```

#### 4. Clean Up Processes
```bash
# Check for any remaining processes
ps aux | grep -E "(redis|postgres|go run)"

# Kill any remaining processes (replace PID with actual process ID)
kill -9 <PID>

# Clean up any stuck ports
sudo lsof -i :8080  # Check port 8080
sudo lsof -i :6379  # Check Redis port
sudo lsof -i :5432  # Check PostgreSQL port
```

### 💾 Save Your Work
```bash
# Check current status
git status

# Add any new files
git add .

# Commit current state
git commit -m "checkpoint: end of development session $(date '+%Y-%m-%d %H:%M')"

# Push to GitHub (if authentication is set up)
git push origin main

# Create local backup (optional)
tar -czf ../rate-limiter-backup-$(date +%Y%m%d).tar.gz .
```

## 🚀 Complete Restart Process

### Quick Restart (Recommended)
```bash
# Navigate to project directory
cd distributed-rate-limiter

# Run the automated restart script
./scripts/restart.sh
```

### Manual Restart Steps

#### 1. Start Infrastructure Services
```bash
# Start Docker services (Redis + PostgreSQL)
docker-compose up -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 10

# Verify services are running
docker-compose ps
```

#### 2. Verify Service Connectivity
```bash
# Test Redis connection
redis-cli ping
# Expected output: PONG

# Test PostgreSQL connection (if using)
psql -h localhost -U ratelimiter -d ratelimiter -c "SELECT version();"

# Check Docker service health
docker-compose logs redis
docker-compose logs postgres
```

#### 3. Start the Rate Limiter Service
```bash
# Method 1: Foreground (recommended for development)
go run cmd/server/main.go

# Method 2: Background (for testing)
nohup go run cmd/server/main.go > server.log 2>&1 &

# Method 3: Using air for hot reload (if installed)
air
```

#### 4. Verify Everything is Working
```bash
# Test health endpoint
curl http://localhost:8080/health
# Expected: {"status":"healthy","redis":"connected","time":...}

# Test rate limiting functionality
curl -X POST http://localhost:8080/api/v1/ratelimit \
  -H "Content-Type: application/json" \
  -d '{
    "key": "test_user",
    "limit": 10,
    "window": 60,
    "algorithm": "token_bucket"
  }'
# Expected: {"allowed":true,"remaining":9,...}
```

## 🔧 Automated Scripts

### Create Shutdown Script
```bash
# Create scripts directory if it doesn't exist
mkdir -p scripts

# Create comprehensive shutdown script
cat > scripts/shutdown.sh << 'EOF'
#!/bin/bash

echo "🛑 Shutting down Distributed Rate Limiter..."
echo "========================================"

# Stop the Go server
echo "📱 Stopping Rate Limiter service..."
pkill -f "go run cmd/server/main.go" 2>/dev/null
pkill -f "air" 2>/dev/null
sleep 2

# Verify service is stopped
if lsof -i :8080 >/dev/null 2>&1; then
    echo "⚠️  Port 8080 still in use, forcing shutdown..."
    kill -9 $(lsof -t -i:8080) 2>/dev/null
fi

# Stop Docker services
echo "🐳 Stopping Docker services..."
docker-compose down

# Verify containers are stopped
RUNNING_CONTAINERS=$(docker-compose ps -q)
if [ -n "$RUNNING_CONTAINERS" ]; then
    echo "⚠️  Some containers still running, force stopping..."
    docker-compose down -t 0
fi

# Stop local services (if running)
echo "🔧 Stopping local services..."
redis-cli shutdown 2>/dev/null || echo "   Redis: not running locally"
brew services stop redis 2>/dev/null || echo "   Redis: not managed by brew"

# Save current work
echo "💾 Saving current work..."
git add . 2>/dev/null
if git diff --staged --quiet; then
    echo "   No changes to commit"
else
    git commit -m "checkpoint: session ended $(date '+%Y-%m-%d %H:%M')" 2>/dev/null
    echo "   ✅ Work committed locally"
fi

# Clean up temporary files
echo "🧹 Cleaning up..."
rm -f server.log
rm -f nohup.out
rm -rf tmp/

# Final verification
echo ""
echo "🔍 Final verification:"
echo "   Port 8080: $(lsof -i :8080 >/dev/null 2>&1 && echo 'OCCUPIED ⚠️' || echo 'FREE ✅')"
echo "   Docker containers: $(docker-compose ps -q | wc -l | tr -d ' ') running"
echo "   Git status: $(git status --porcelain | wc -l | tr -d ' ') uncommitted files"

echo ""
echo "✅ Shutdown complete!"
echo "🚀 Run './scripts/restart.sh' to restart when ready"
echo "========================================"
EOF

# Make script executable
chmod +x scripts/shutdown.sh
```

### Create Restart Script
```bash
cat > scripts/restart.sh << 'EOF'
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

# Check PostgreSQL connectivity (if using)
echo "🐘 Testing PostgreSQL connection..."
if docker-compose ps postgres | grep -q "Up"; then
    PG_RETRIES=0
    while [ $PG_RETRIES -lt 10 ]; do
        if docker-compose exec -T postgres pg_isready -U ratelimiter >/dev/null 2>&1; then
            echo "   ✅ PostgreSQL connected"
            break
        fi
        PG_RETRIES=$((PG_RETRIES + 1))
        echo "   ⏳ Attempt $PG_RETRIES/10..."
        sleep 2
    done
    
    if [ $PG_RETRIES -eq 10 ]; then
        echo "   ❌ PostgreSQL connection failed"
        docker-compose logs postgres
        exit 1
    fi
else
    echo "   ⚠️ PostgreSQL not configured"
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
echo "   Rate Limit: curl -X POST http://localhost:8080/api/v1/ratelimit -d '{...}'"
echo ""
echo "========================================"

# Check if air is available for hot reload
if command -v air &> /dev/null; then
    echo "🔥 Starting with hot reload (air)..."
    air
else
    echo "🏃 Starting normally (install 'air' for hot reload)..."
    go run cmd/server/main.go
fi
EOF

# Make script executable
chmod +x scripts/restart.sh
```

### Create Status Check Script
```bash
cat > scripts/status.sh << 'EOF'
#!/bin/bash

echo "📊 Distributed Rate Limiter Status"
echo "================================="

# Check Go service
echo "🎯 Rate Limiter Service:"
if lsof -i :8080 >/dev/null 2>&1; then
    echo "   ✅ Running on port 8080"
    curl -s http://localhost:8080/health | jq . 2>/dev/null || echo "   ⚠️ Service running but not responding properly"
else
    echo "   ❌ Not running"
fi

# Check Docker services
echo ""
echo "🐳 Docker Services:"
docker-compose ps

# Check Redis
echo ""
echo "🔴 Redis:"
if redis-cli ping >/dev/null 2>&1; then
    echo "   ✅ Connected and responding"
    redis-cli info memory | grep used_memory_human || true
else
    echo "   ❌ Not responding"
fi

# Check PostgreSQL
echo ""
echo "🐘 PostgreSQL:"
if docker-compose exec -T postgres pg_isready -U ratelimiter >/dev/null 2>&1; then
    echo "   ✅ Connected and ready"
else
    echo "   ❌ Not ready"
fi

# Check system resources
echo ""
echo "💻 System Resources:"
echo "   Memory: $(free -h 2>/dev/null | grep '^Mem:' | awk '{print $3 "/" $2}' || echo 'N/A')"
echo "   CPU: $(top -l 1 -n 0 | grep "CPU usage" 2>/dev/null || echo 'N/A')"
echo "   Disk: $(df -h . | tail -1 | awk '{print $3 "/" $2 " (" $5 " used)"}')"

# Check ports
echo ""
echo "🔌 Port Usage:"
echo "   8080 (API): $(lsof -i :8080 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"
echo "   6379 (Redis): $(lsof -i :6379 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"
echo "   5432 (PostgreSQL): $(lsof -i :5432 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"

echo ""
echo "================================="
EOF

# Make script executable
chmod +x scripts/status.sh
```

## 🔄 Quick Reference Commands

### Daily Usage
```bash
# Start everything
./scripts/restart.sh

# Check status
./scripts/status.sh

# Stop everything
./scripts/shutdown.sh
```

### Development Workflow
```bash
# Start services only (without Go server)
docker-compose up -d

# Start Go server with hot reload
air
# OR
go run cmd/server/main.go

# Test the API
curl http://localhost:8080/health
```

### Troubleshooting Commands
```bash
# Check what's using port 8080
sudo lsof -i :8080

# Kill process on port 8080
kill -9 $(lsof -t -i:8080)

# Restart Docker services
docker-compose restart

# View service logs
docker-compose logs -f redis
docker-compose logs -f postgres

# Clean Docker system
docker system prune -f
```

## 🆘 Emergency Recovery

### If Services Won't Stop
```bash
# Nuclear option - stop everything Docker related
docker stop $(docker ps -aq)
docker rm $(docker ps -aq)

# Kill all Go processes
pkill -9 go

# Reset network (macOS)
sudo dscacheutil -flushcache

# Restart Docker Desktop
# macOS: Quit Docker Desktop and restart
# Linux: sudo systemctl restart docker
```

### If Port 8080 is Stuck
```bash
# Find and kill the process
sudo lsof -i :8080
sudo kill -9 <PID>

# Or use different port
export PORT=8081
go run cmd/server/main.go
```

### If Redis Data is Corrupted
```bash
# Stop everything
docker-compose down

# Remove Redis data volume
docker-compose down -v

# Restart (will create fresh Redis)
docker-compose up -d
```

## 📝 Session Management

### Save Session State
```bash
# Create session snapshot
git add .
git commit -m "session: $(date '+%Y-%m-%d %H:%M') - current progress"
git tag "session-$(date '+%Y%m%d-%H%M')"

# Export environment state
env | grep -E "(REDIS|POSTGRES|PORT)" > .env.backup
```

### Restore Session State
```bash
# List available session tags
git tag | grep session

# Restore to specific session
git checkout session-20240727-1430

# Restore environment
source .env.backup
```

---

**Keep this guide handy for smooth development sessions! The automated scripts make starting and stopping your distributed rate limiter effortless.** 🚀

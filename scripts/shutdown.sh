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

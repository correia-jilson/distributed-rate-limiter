#!/bin/bash

echo "📊 Distributed Rate Limiter Status"
echo "================================="

# Check Go service
echo "🎯 Rate Limiter Service:"
if lsof -i :8080 >/dev/null 2>&1; then
    echo "   ✅ Running on port 8080"
    curl -s http://localhost:8080/health 2>/dev/null | head -1 || echo "   ⚠️ Service running but not responding properly"
else
    echo "   ❌ Not running"
fi

# Check Docker services
echo ""
echo "🐳 Docker Services:"
if command -v docker-compose &> /dev/null; then
    docker-compose ps
else
    echo "   docker-compose not available"
fi

# Check Redis
echo ""
echo "🔴 Redis:"
if redis-cli ping >/dev/null 2>&1; then
    echo "   ✅ Connected and responding"
else
    echo "   ❌ Not responding"
fi

# Check ports
echo ""
echo "🔌 Port Usage:"
echo "   8080 (API): $(lsof -i :8080 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"
echo "   6379 (Redis): $(lsof -i :6379 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"
echo "   5432 (PostgreSQL): $(lsof -i :5432 >/dev/null 2>&1 && echo 'OCCUPIED' || echo 'FREE')"

echo ""
echo "================================="

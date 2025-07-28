// cmd/server/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

type RateLimitRequest struct {
	Key       string `json:"key" binding:"required"`    // User/API key identifier
	Limit     int    `json:"limit" binding:"required"`  // Requests per window or bucket capacity
	Window    int    `json:"window" binding:"required"` // Window in seconds or refill rate
	Algorithm string `json:"algorithm,omitempty"`       // token_bucket, fixed_window, sliding_window
	Tokens    int    `json:"tokens,omitempty"`          // Tokens requested (for token bucket)
}

type RateLimitResponse struct {
	Allowed    bool    `json:"allowed"`
	Remaining  int     `json:"remaining"`
	ResetTime  int64   `json:"reset_time"`
	RetryAfter int     `json:"retry_after,omitempty"` // Seconds to wait before retry
	Algorithm  string  `json:"algorithm"`
	Tokens     float64 `json:"tokens,omitempty"` // Current tokens (for token bucket)
}

// TokenBucket implementation
type TokenBucket struct {
	redis     *redis.Client
	luaScript string
}

func NewTokenBucket(rdb *redis.Client) *TokenBucket {
	luaScript := `
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_rate = tonumber(ARGV[2])
		local requested_tokens = tonumber(ARGV[3])
		local now = tonumber(ARGV[4])

		-- Get current bucket state
		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1])
		local last_refill = tonumber(bucket[2])

		-- Initialize bucket if it doesn't exist
		if tokens == nil then
			tokens = capacity
			last_refill = now
		end

		-- Calculate tokens to add based on time elapsed
		local time_elapsed = math.max(0, now - last_refill)
		local tokens_to_add = time_elapsed * refill_rate
		tokens = math.min(capacity, tokens + tokens_to_add)

		-- Check if we have enough tokens
		local allowed = 0
		local retry_after = 0
		
		if tokens >= requested_tokens then
			tokens = tokens - requested_tokens
			allowed = 1
		else
			-- Calculate when next token will be available
			local tokens_needed = requested_tokens - tokens
			retry_after = math.ceil(tokens_needed / refill_rate)
		end

		-- Update bucket state
		redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
		redis.call('EXPIRE', key, 3600) -- Expire after 1 hour of inactivity

		return {allowed, tokens, retry_after}
	`

	return &TokenBucket{
		redis:     rdb,
		luaScript: luaScript,
	}
}

type Server struct {
	router      *gin.Engine
	redis       *redis.Client
	logger      *logrus.Logger
	tokenBucket *TokenBucket
}

func NewServer() *Server {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Initialize Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Test Redis connection
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		logger.Fatal("Failed to connect to Redis: ", err)
	}
	logger.Info("Connected to Redis successfully")

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Initialize token bucket
	tokenBucket := NewTokenBucket(rdb)

	server := &Server{
		router:      router,
		redis:       rdb,
		logger:      logger,
		tokenBucket: tokenBucket,
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// Rate limiting endpoint
	s.router.POST("/api/v1/ratelimit", s.checkRateLimit)

	// Debug endpoints
	s.router.GET("/api/v1/bucket/:key", s.getBucketState)
	s.router.GET("/api/v1/sliding/:key", s.getSlidingWindowState)

	// Metrics endpoint (placeholder for now)
	s.router.GET("/metrics", s.getMetrics)
}

func (s *Server) healthCheck(c *gin.Context) {
	// Check Redis connection
	ctx := context.Background()
	_, err := s.redis.Ping(ctx).Result()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"redis":  "disconnected",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"redis":  "connected",
		"time":   time.Now().Unix(),
	})
}

func (s *Server) checkRateLimit(c *gin.Context) {
	var req RateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if req.Algorithm == "" {
		req.Algorithm = "token_bucket"
	}
	if req.Tokens == 0 {
		req.Tokens = 1
	}

	var response RateLimitResponse

	switch req.Algorithm {
	case "token_bucket":
		response = s.tokenBucketRateLimit(req)
	case "fixed_window":
		allowed, remaining, resetTime := s.fixedWindowRateLimit(req)
		response = RateLimitResponse{
			Allowed:   allowed,
			Remaining: remaining,
			ResetTime: resetTime,
			Algorithm: req.Algorithm,
		}
	case "sliding_window":
		response = s.slidingWindowRateLimit(req)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported algorithm: " + req.Algorithm})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"key":       req.Key,
		"algorithm": req.Algorithm,
		"allowed":   response.Allowed,
		"remaining": response.Remaining,
	}).Info("Rate limit check")

	c.JSON(http.StatusOK, response)
}

func (s *Server) tokenBucketRateLimit(req RateLimitRequest) RateLimitResponse {
	ctx := context.Background()
	now := float64(time.Now().UnixNano()) / 1e9

	// Calculate refill rate (tokens per second)
	refillRate := float64(req.Limit) / float64(req.Window)

	key := fmt.Sprintf("token_bucket:%s", req.Key)

	// Execute Lua script using Eval
	result, err := s.redis.Eval(ctx, s.tokenBucket.luaScript, []string{key},
		req.Limit,  // capacity
		refillRate, // refill rate
		req.Tokens, // requested tokens
		fmt.Sprintf("%.6f", now)).Result()

	if err != nil {
		s.logger.Error("Token bucket error: ", err)
		// Fail open
		return RateLimitResponse{
			Allowed:   true,
			Remaining: req.Limit - 1,
			ResetTime: time.Now().Add(time.Duration(req.Window) * time.Second).Unix(),
			Algorithm: req.Algorithm,
		}
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1

	// Handle tokens value (could be string or number)
	var tokensFloat float64
	switch v := values[1].(type) {
	case string:
		tokensFloat, _ = strconv.ParseFloat(v, 64)
	case int64:
		tokensFloat = float64(v)
	case float64:
		tokensFloat = v
	default:
		tokensFloat = 0
	}

	retryAfter := int(values[2].(int64))

	return RateLimitResponse{
		Allowed:    allowed,
		Remaining:  int(tokensFloat),
		ResetTime:  time.Now().Add(time.Duration(req.Window) * time.Second).Unix(),
		RetryAfter: retryAfter,
		Algorithm:  req.Algorithm,
		Tokens:     tokensFloat,
	}
}

func (s *Server) fixedWindowRateLimit(req RateLimitRequest) (bool, int, int64) {
	ctx := context.Background()
	now := time.Now()
	window := time.Duration(req.Window) * time.Second

	windowStart := now.Truncate(window)
	key := fmt.Sprintf("fixed_window:%s:%d", req.Key, windowStart.Unix())

	count, err := s.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		count = 0
	} else if err != nil {
		s.logger.Error("Redis error: ", err)
		return true, req.Limit - 1, windowStart.Add(window).Unix()
	}

	if count >= req.Limit {
		return false, 0, windowStart.Add(window).Unix()
	}

	pipe := s.redis.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err = pipe.Exec(ctx)

	if err != nil {
		s.logger.Error("Redis pipeline error: ", err)
		return true, req.Limit - 1, windowStart.Add(window).Unix()
	}

	remaining := req.Limit - count - 1
	if remaining < 0 {
		remaining = 0
	}

	return true, remaining, windowStart.Add(window).Unix()
}

func (s *Server) slidingWindowRateLimit(req RateLimitRequest) RateLimitResponse {
	ctx := context.Background()
	now := time.Now()
	windowStart := now.Add(-time.Duration(req.Window) * time.Second)

	key := fmt.Sprintf("sliding_window:%s", req.Key)

	// Lua script for sliding window implementation
	luaScript := `
		local key = KEYS[1]
		local window_start = tonumber(ARGV[1])
		local now = tonumber(ARGV[2])
		local limit = tonumber(ARGV[3])
		local request_id = ARGV[4]

		-- Remove expired entries (outside the sliding window)
		redis.call('ZREMRANGEBYSCORE', key, '-inf', window_start)

		-- Count current requests in the window
		local current_count = redis.call('ZCARD', key)

		-- Check if request should be allowed
		if current_count < limit then
			-- Add current request to the sorted set
			redis.call('ZADD', key, now, request_id)
			redis.call('EXPIRE', key, 3600) -- Expire after 1 hour of inactivity
			return {1, limit - current_count - 1, 0}
		else
			-- Get the oldest request timestamp to calculate retry time
			local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
			local retry_after = 1
			if #oldest > 0 then
				local oldest_time = tonumber(oldest[2])
				local window_duration = tonumber(ARGV[5] or 60)
				retry_after = math.ceil((oldest_time + window_duration) - now)
				if retry_after < 1 then retry_after = 1 end
			end
			return {0, 0, retry_after}
		end
	`

	// Generate unique request ID
	requestID := fmt.Sprintf("%d_%d", now.UnixNano(), time.Now().Nanosecond())

	// Execute Lua script
	result, err := s.redis.Eval(ctx, luaScript, []string{key},
		windowStart.Unix(),  // window_start
		now.Unix(),          // now
		req.Limit,           // limit
		requestID,           // request_id
		req.Window).Result() // window duration for retry calculation

	if err != nil {
		s.logger.Error("Sliding window error: ", err)
		// Fail open
		return RateLimitResponse{
			Allowed:   true,
			Remaining: req.Limit - 1,
			ResetTime: now.Add(time.Duration(req.Window) * time.Second).Unix(),
			Algorithm: req.Algorithm,
		}
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	remaining := int(values[1].(int64))
	retryAfter := int(values[2].(int64))

	return RateLimitResponse{
		Allowed:    allowed,
		Remaining:  remaining,
		ResetTime:  now.Add(time.Duration(req.Window) * time.Second).Unix(),
		RetryAfter: retryAfter,
		Algorithm:  req.Algorithm,
	}
}

func (s *Server) getBucketState(c *gin.Context) {
	key := c.Param("key")
	bucketKey := fmt.Sprintf("token_bucket:%s", key)

	result, err := s.redis.HGetAll(c.Request.Context(), bucketKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"state": result,
	})
}

func (s *Server) getSlidingWindowState(c *gin.Context) {
	key := c.Param("key")
	windowKey := fmt.Sprintf("sliding_window:%s", key)

	// Get all entries with scores (timestamps)
	result, err := s.redis.ZRangeWithScores(c.Request.Context(), windowKey, 0, -1).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Format for easy reading
	entries := make([]map[string]interface{}, len(result))
	for i, entry := range result {
		timestamp := time.Unix(int64(entry.Score), 0)
		entries[i] = map[string]interface{}{
			"request_id": entry.Member,
			"timestamp":  timestamp.Format(time.RFC3339),
			"score":      entry.Score,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"key":     key,
		"count":   len(entries),
		"entries": entries,
	})
}

func (s *Server) getMetrics(c *gin.Context) {
	c.String(http.StatusOK, "# Metrics endpoint - Coming soon!")
}

func (s *Server) Start(port string) error {
	s.logger.Info("Starting rate limiter server on port ", port)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: s.router,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		s.logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			s.logger.Fatal("Server forced to shutdown: ", err)
		}
	}()

	return srv.ListenAndServe()
}

func main() {
	server := NewServer()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := server.Start(port); err != nil && err != http.ErrServerClosed {
		log.Fatal("Failed to start server: ", err)
	}
}

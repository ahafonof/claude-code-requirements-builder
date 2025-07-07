package main

import (
	"context"
	"net/http"
	"sync"
	"time"
	
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	StateClosed    CircuitState = iota // Normal operation (using Redis)
	StateOpen                           // Circuit open (using fallback)
	StateHalfOpen                       // Testing if Redis is available again
)

// Config for distributed rate limiter
type Config struct {
	RedisURL         string
	Limit            int
	Window           time.Duration
	FailureThreshold int
	RecoveryInterval time.Duration
}

// Metrics tracks rate limiter performance
type Metrics struct {
	mu               sync.RWMutex
	TotalRequests    int64
	AllowedRequests  int64
	RejectedRequests int64
	RedisLatency     time.Duration
	RedisFailures    int64
	FallbackMode     string
	FallbackCount    int64
	LastUpdated      time.Time
}

// CircuitBreaker manages Redis connection health
type CircuitBreaker struct {
	mu               sync.RWMutex
	state            CircuitState
	failures         int
	failureThreshold int
	lastFailureTime  time.Time
	recoveryInterval time.Duration
}

// DistributedRateLimiter extends basic rate limiter with Redis support
type DistributedRateLimiter struct {
	redisClient     *redis.Client
	fallbackLimiter *RateLimiter
	circuitBreaker  *CircuitBreaker
	metrics         *Metrics
	config          *Config
	luaScript       *redis.Script
	ctx             context.Context
	eventEmitter    *EventEmitter
}

// NewDistributedRateLimiter creates a new distributed rate limiter
func NewDistributedRateLimiter(cfg *Config, eventEmitter *EventEmitter) (*DistributedRateLimiter, error) {
	// Initialize Redis client
	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	
	redisClient := redis.NewClient(opt)
	ctx := context.Background()
	
	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		// Redis not available, but we'll continue with fallback
		// Log the error for debugging purposes
		_ = err
	}
	
	// Create fallback limiter
	fallbackLimiter := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    cfg.Limit,
		window:   cfg.Window,
	}
	
	// Create circuit breaker
	circuitBreaker := &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: cfg.FailureThreshold,
		recoveryInterval: cfg.RecoveryInterval,
	}
	
	// Initialize metrics
	metrics := &Metrics{
		FallbackMode: "distributed",
		LastUpdated:  time.Now(),
	}
	
	// Lua script for atomic rate limiting
	luaScript := redis.NewScript(`
		local key = KEYS[1]
		local now = ARGV[1]
		local windowStart = ARGV[2]
		local limit = tonumber(ARGV[3])
		local requestId = ARGV[4]
		
		-- Remove old entries
		redis.call('ZREMRANGEBYSCORE', key, 0, windowStart)
		
		-- Count current requests
		local count = redis.call('ZCARD', key)
		
		-- Check limit
		if count >= limit then
			return 0
		else
			redis.call('ZADD', key, now, requestId)
			redis.call('EXPIRE', key, 120)
			return 1
		end
	`)
	
	drl := &DistributedRateLimiter{
		redisClient:     redisClient,
		fallbackLimiter: fallbackLimiter,
		circuitBreaker:  circuitBreaker,
		metrics:         metrics,
		config:          cfg,
		luaScript:       luaScript,
		ctx:             ctx,
		eventEmitter:    eventEmitter,
	}
	
	// Start recovery goroutine
	go drl.startRecoveryMonitor()
	
	return drl, nil
}

// Allow checks if request from IP should be allowed
func (drl *DistributedRateLimiter) Allow(ip string) bool {
	start := time.Now()
	drl.metrics.mu.Lock()
	drl.metrics.TotalRequests++
	drl.metrics.mu.Unlock()
	
	// Check circuit breaker state
	if drl.circuitBreaker.IsOpen() {
		return drl.fallbackAllow(ip)
	}
	
	// Try Redis operation
	allowed, err := drl.redisAllow(ip)
	if err != nil {
		drl.circuitBreaker.RecordFailure(drl.eventEmitter)
		// Emit Redis failure event
		if drl.eventEmitter != nil {
			drl.eventEmitter.EmitRedisFailure("rate_limit_check", err)
		}
		return drl.fallbackAllow(ip)
	}
	
	// Record success
	drl.circuitBreaker.RecordSuccess()
	drl.recordMetrics(allowed, time.Since(start), false)
	
	return allowed
}

// AllowWithRequest checks if request should be allowed and emits events
func (drl *DistributedRateLimiter) AllowWithRequest(ip string, r *http.Request) bool {
	allowed := drl.Allow(ip)
	
	// Emit rate limit rejection event if applicable
	if !allowed && drl.eventEmitter != nil {
		drl.eventEmitter.EmitRateLimitRejection(r)
	}
	
	return allowed
}

// redisAllow performs rate limiting using Redis
func (drl *DistributedRateLimiter) redisAllow(ip string) (bool, error) {
	key := "rate_limit:" + ip
	now := time.Now().UnixMilli()
	windowStart := now - int64(drl.config.Window.Milliseconds())
	requestID := uuid.New().String()
	
	result, err := drl.luaScript.Run(
		drl.ctx,
		drl.redisClient,
		[]string{key},
		now,
		windowStart,
		drl.config.Limit,
		requestID,
	).Result()
	
	if err != nil {
		return false, err
	}
	
	allowed := result.(int64) == 1
	return allowed, nil
}

// fallbackAllow uses local rate limiter when Redis is unavailable
func (drl *DistributedRateLimiter) fallbackAllow(ip string) bool {
	drl.metrics.mu.Lock()
	drl.metrics.FallbackCount++
	drl.metrics.FallbackMode = "fallback"
	drl.metrics.mu.Unlock()
	
	allowed := drl.fallbackLimiter.allow(ip)
	drl.recordMetrics(allowed, 0, true)
	
	return allowed
}

// recordMetrics updates performance metrics
func (drl *DistributedRateLimiter) recordMetrics(allowed bool, latency time.Duration, isFallback bool) {
	drl.metrics.mu.Lock()
	defer drl.metrics.mu.Unlock()
	
	if allowed {
		drl.metrics.AllowedRequests++
	} else {
		drl.metrics.RejectedRequests++
	}
	
	if !isFallback {
		drl.metrics.RedisLatency = latency
	}
	
	drl.metrics.LastUpdated = time.Now()
}

// GetMetrics returns current metrics
func (drl *DistributedRateLimiter) GetMetrics() Metrics {
	drl.metrics.mu.RLock()
	defer drl.metrics.mu.RUnlock()
	
	// Create a copy without the mutex
	return Metrics{
		TotalRequests:    drl.metrics.TotalRequests,
		AllowedRequests:  drl.metrics.AllowedRequests,
		RejectedRequests: drl.metrics.RejectedRequests,
		RedisLatency:     drl.metrics.RedisLatency,
		RedisFailures:    drl.metrics.RedisFailures,
		FallbackMode:     drl.metrics.FallbackMode,
		FallbackCount:    drl.metrics.FallbackCount,
		LastUpdated:      drl.metrics.LastUpdated,
	}
}

// startRecoveryMonitor periodically checks if Redis is available
func (drl *DistributedRateLimiter) startRecoveryMonitor() {
	ticker := time.NewTicker(drl.config.RecoveryInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		if drl.circuitBreaker.IsOpen() {
			// Try to ping Redis
			if err := drl.redisClient.Ping(drl.ctx).Err(); err == nil {
				drl.circuitBreaker.Reset()
				drl.metrics.mu.Lock()
				drl.metrics.FallbackMode = "distributed"
				drl.metrics.mu.Unlock()
			}
		}
	}
}

// Close closes Redis connection
func (drl *DistributedRateLimiter) Close() error {
	return drl.redisClient.Close()
}

// CircuitBreaker methods

// IsOpen checks if circuit is open
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	return cb.state == StateOpen
}

// RecordFailure records a Redis failure
func (cb *CircuitBreaker) RecordFailure(eventEmitter *EventEmitter) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	oldState := cb.state
	cb.failures++
	cb.lastFailureTime = time.Now()
	
	if cb.failures >= cb.failureThreshold {
		cb.state = StateOpen
		// Emit circuit breaker state change event
		if eventEmitter != nil && oldState != StateOpen {
			stateNames := map[CircuitState]string{
				StateClosed:   "closed",
				StateOpen:     "open",
				StateHalfOpen: "half-open",
			}
			eventEmitter.EmitCircuitBreakerStateChange(stateNames[oldState], stateNames[StateOpen], cb.failures)
		}
	}
}

// RecordSuccess records a successful Redis operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		cb.failures = 0
	}
}

// Reset attempts to reset the circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	if cb.state == StateOpen {
		cb.state = StateHalfOpen
	}
}
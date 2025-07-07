package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"
	
	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

// Helper function to create test config
func testConfig() *Config {
	return &Config{
		RedisURL:         "redis://localhost:6379/0",
		Limit:            10,
		Window:           time.Minute,
		FailureThreshold: 5,
		RecoveryInterval: 10 * time.Second,
	}
}

// Helper function to check Redis availability
func skipIfRedisUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = client.Close() }()
	
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available, skipping integration tests")
	}
}

// Test basic Redis connection
func TestRedisConnection(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = client.Close() }()
	
	ctx := context.Background()
	
	// Test ping
	err := client.Ping(ctx).Err()
	if err != nil {
		t.Errorf("Redis ping failed: %v", err)
	}
	
	// Test basic operations
	key := "test_key"
	err = client.Set(ctx, key, "test_value", time.Second).Err()
	if err != nil {
		t.Errorf("Redis SET failed: %v", err)
	}
	
	val, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Errorf("Redis GET failed: %v", err)
	}
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got %s", val)
	}
}

// Test Lua script functionality
func TestLuaScript(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = client.Close() }()
	
	ctx := context.Background()
	
	// Define Lua script (same as in DistributedRateLimiter)
	script := redis.NewScript(`
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
	
	// Test script execution
	now := time.Now().UnixMilli()
	windowStart := now - 60000 // 1 minute window
	
	testCases := []struct {
		name     string
		limit    int
		requests int
		expected int64
	}{
		{"under_limit", 5, 3, 1},
		{"at_limit", 5, 5, 0},
		{"over_limit", 5, 7, 0},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			key := "test_rate_limit_" + tc.name
			client.Del(ctx, key)
			
			// Make requests up to limit
			for i := 0; i < tc.requests-1; i++ {
				requestId := uuid.New().String()
				script.Run(ctx, client, []string{key}, now, windowStart, tc.limit, requestId)
			}
			
			// Final request should match expected result
			requestId := uuid.New().String()
			result, err := script.Run(ctx, client, []string{key}, now, windowStart, tc.limit, requestId).Result()
			
			if err != nil {
				t.Errorf("Script execution failed: %v", err)
			}
			
			if result != tc.expected {
				t.Errorf("Expected %d, got %v", tc.expected, result)
			}
		})
	}
}

// Test distributed scenario with multiple clients
func TestDistributedScenario(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	client1 := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	client2 := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer func() { _ = client1.Close() }()
	defer func() { _ = client2.Close() }()
	
	ctx := context.Background()
	
	// Lua script
	script := redis.NewScript(`
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
	
	key := "test_distributed"
	client1.Del(ctx, key)
	
	limit := 10
	now := time.Now().UnixMilli()
	windowStart := now - 60000
	
	// Client 1 makes 5 requests
	for i := 0; i < 5; i++ {
		requestId := uuid.New().String()
		result, err := script.Run(ctx, client1, []string{key}, now, windowStart, limit, requestId).Result()
		if err != nil {
			t.Errorf("Client1 request %d failed: %v", i, err)
		}
		if result != int64(1) {
			t.Errorf("Client1 request %d rejected unexpectedly", i)
		}
	}
	
	// Client 2 makes 5 requests (should reach limit)
	for i := 0; i < 5; i++ {
		requestId := uuid.New().String()
		result, err := script.Run(ctx, client2, []string{key}, now, windowStart, limit, requestId).Result()
		if err != nil {
			t.Errorf("Client2 request %d failed: %v", i, err)
		}
		if result != int64(1) {
			t.Errorf("Client2 request %d rejected unexpectedly", i)
		}
	}
	
	// Next request should be rejected
	requestId := uuid.New().String()
	result, err := script.Run(ctx, client1, []string{key}, now, windowStart, limit, requestId).Result()
	if err != nil {
		t.Errorf("Final request failed: %v", err)
	}
	if result != int64(0) {
		t.Errorf("Expected rejection, got %v", result)
	}
}

// Test cleanup and TTL
func TestCleanupAndTTL(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	defer func() { _ = client.Close() }()
	
	ctx := context.Background()
	
	// Test that old entries are cleaned up
	key := "test_cleanup"
	client.Del(ctx, key)
	
	// Add old entry
	oldTimestamp := float64(time.Now().Add(-2 * time.Minute).UnixMilli())
	client.ZAdd(ctx, key, redis.Z{Score: oldTimestamp, Member: "old_request"})
	
	// Add current entry
	now := float64(time.Now().UnixMilli())
	client.ZAdd(ctx, key, redis.Z{Score: now, Member: "current_request"})
	
	// Check initial count
	count, err := client.ZCard(ctx, key).Result()
	if err != nil {
		t.Errorf("ZCard failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 entries, got %d", count)
	}
	
	// Remove old entries
	windowStart := now - 60000
	removed, err := client.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%f", windowStart)).Result()
	if err != nil {
		t.Errorf("ZRemRangeByScore failed: %v", err)
	}
	if removed != 1 {
		t.Errorf("Expected to remove 1 entry, removed %d", removed)
	}
	
	// Check count after cleanup
	count, err = client.ZCard(ctx, key).Result()
	if err != nil {
		t.Errorf("ZCard after cleanup failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 entry after cleanup, got %d", count)
	}
}

// Test DistributedRateLimiter basic functionality
func TestDistributedRateLimiter_Allow(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	cfg := testConfig()
	cfg.Limit = 5
	cfg.Window = time.Second
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	ip := "192.168.1.1"
	
	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		if !drl.Allow(ip) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}
	
	// 6th request should be rejected
	if drl.Allow(ip) {
		t.Error("6th request should be rejected")
	}
	
	// Check metrics
	metrics := drl.GetMetrics()
	if metrics.TotalRequests != 6 {
		t.Errorf("Expected 6 total requests, got %d", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 5 {
		t.Errorf("Expected 5 allowed requests, got %d", metrics.AllowedRequests)
	}
	if metrics.RejectedRequests != 1 {
		t.Errorf("Expected 1 rejected request, got %d", metrics.RejectedRequests)
	}
}

// Test circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	cb := &CircuitBreaker{
		failureThreshold: 3,
		recoveryInterval: time.Second,
		state:           StateClosed,
	}
	
	// Initially closed
	if cb.IsOpen() {
		t.Error("Circuit should be closed initially")
	}
	
	// Record failures
	cb.RecordFailure(nil)
	cb.RecordFailure(nil)
	if cb.IsOpen() {
		t.Error("Circuit should still be closed after 2 failures")
	}
	
	// Third failure should open circuit
	cb.RecordFailure(nil)
	if !cb.IsOpen() {
		t.Error("Circuit should be open after 3 failures")
	}
	
	// Reset to half-open
	cb.Reset()
	if cb.state != StateHalfOpen {
		t.Error("Circuit should be half-open after reset")
	}
	
	// Success should close circuit
	cb.RecordSuccess()
	if cb.IsOpen() {
		t.Error("Circuit should be closed after successful operation")
	}
}

// Test fallback scenario
func TestFallbackScenario(t *testing.T) {
	// Use invalid Redis URL to simulate unavailable Redis
	cfg := testConfig()
	cfg.RedisURL = "redis://invalid:6379/0"
	cfg.Limit = 3
	cfg.Window = time.Second
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	
	// Should still work with fallback
	if drl != nil {
		defer func() { _ = drl.Close() }()
		
		ip := "192.168.1.1"
		
		// Should use fallback limiter
		for i := 0; i < 3; i++ {
			if !drl.Allow(ip) {
				t.Errorf("Fallback request %d should be allowed", i+1)
			}
		}
		
		// Check that we're in fallback mode
		metrics := drl.GetMetrics()
		if metrics.FallbackMode != "fallback" {
			t.Errorf("Expected fallback mode, got %s", metrics.FallbackMode)
		}
		if metrics.FallbackCount < 3 {
			t.Errorf("Expected at least 3 fallback requests, got %d", metrics.FallbackCount)
		}
	}
}

// Test AllowWithRequest with event emission
func TestDistributedRateLimiter_AllowWithRequest(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	// Create activity feed and event emitter
	feed := NewActivityFeed(100)
	broadcaster := NewSSEBroadcaster()
	eventEmitter := &EventEmitter{
		feed:        feed,
		broadcaster: broadcaster,
	}
	
	cfg := testConfig()
	cfg.Limit = 2
	cfg.Window = time.Second
	
	drl, err := NewDistributedRateLimiter(cfg, eventEmitter)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	
	// First two requests should be allowed
	if !drl.AllowWithRequest("192.168.1.1", req) {
		t.Error("First request should be allowed")
	}
	if !drl.AllowWithRequest("192.168.1.1", req) {
		t.Error("Second request should be allowed")
	}
	
	// Third request should be rejected and emit event
	if drl.AllowWithRequest("192.168.1.1", req) {
		t.Error("Third request should be rejected")
	}
	
	// Check that rate limit rejection event was emitted
	events := feed.GetRecentEvents(10)
	found := false
	for _, event := range events {
		if event.Type == EventTypeRateLimitRejected {
			found = true
			if event.IP != "192.168.1.1" {
				t.Errorf("Expected IP 192.168.1.1, got %s", event.IP)
			}
			if event.Path != "/test" {
				t.Errorf("Expected path /test, got %s", event.Path)
			}
			break
		}
	}
	
	if !found {
		t.Error("Rate limit rejection event not found")
	}
}

// Test AllowWithRequest with nil eventEmitter
func TestDistributedRateLimiter_AllowWithRequest_NilEmitter(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	cfg := testConfig()
	cfg.Limit = 1
	cfg.Window = time.Second
	
	// Create rate limiter without event emitter
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	
	// First request should be allowed
	if !drl.AllowWithRequest("192.168.1.1", req) {
		t.Error("First request should be allowed")
	}
	
	// Second request should be rejected (no panic with nil emitter)
	if drl.AllowWithRequest("192.168.1.1", req) {
		t.Error("Second request should be rejected")
	}
}

// Test recovery monitor functionality
func TestDistributedRateLimiter_RecoveryMonitor(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	// Create event emitter for tracking state changes
	feed := NewActivityFeed(100)
	broadcaster := NewSSEBroadcaster()
	eventEmitter := &EventEmitter{
		feed:        feed,
		broadcaster: broadcaster,
	}
	
	cfg := testConfig()
	cfg.FailureThreshold = 2
	cfg.RecoveryInterval = 100 * time.Millisecond // Short interval for testing
	
	drl, err := NewDistributedRateLimiter(cfg, eventEmitter)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Force circuit breaker to open by recording failures
	drl.circuitBreaker.RecordFailure(eventEmitter)
	drl.circuitBreaker.RecordFailure(eventEmitter)
	
	if !drl.circuitBreaker.IsOpen() {
		t.Error("Circuit breaker should be open")
	}
	
	// Verify fallback mode
	metrics := drl.GetMetrics()
	initialMode := metrics.FallbackMode
	
	// Wait for recovery monitor to attempt recovery
	time.Sleep(200 * time.Millisecond)
	
	// Circuit should attempt to go to half-open state
	drl.circuitBreaker.mu.RLock()
	state := drl.circuitBreaker.state
	drl.circuitBreaker.mu.RUnlock()
	
	if state != StateHalfOpen && state != StateClosed {
		t.Errorf("Expected circuit to be in half-open or closed state, got %v", state)
	}
	
	// Check that recovery was attempted
	newMetrics := drl.GetMetrics()
	if initialMode == "fallback" && newMetrics.FallbackMode == "distributed" {
		// Recovery successful
		t.Log("Recovery monitor successfully restored distributed mode")
	}
}

// Test Allow with circuit breaker open
func TestDistributedRateLimiter_Allow_CircuitOpen(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	cfg := testConfig()
	cfg.Limit = 3
	cfg.Window = time.Second
	cfg.FailureThreshold = 1 // Open circuit after 1 failure
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Force circuit to open
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateOpen
	drl.circuitBreaker.failures = 1
	drl.circuitBreaker.mu.Unlock()
	
	// Request should use fallback
	ip := "192.168.1.1"
	
	// First 3 requests should be allowed (fallback limit)
	for i := 0; i < 3; i++ {
		if !drl.Allow(ip) {
			t.Errorf("Fallback request %d should be allowed", i+1)
		}
	}
	
	// 4th request should be rejected
	if drl.Allow(ip) {
		t.Error("4th fallback request should be rejected")
	}
	
	// Verify we're in fallback mode
	metrics := drl.GetMetrics()
	if metrics.FallbackMode != "fallback" {
		t.Errorf("Expected fallback mode, got %s", metrics.FallbackMode)
	}
	if metrics.FallbackCount < 4 {
		t.Errorf("Expected at least 4 fallback requests, got %d", metrics.FallbackCount)
	}
}

// Test Allow with Redis failures
func TestDistributedRateLimiter_Allow_RedisFailure(t *testing.T) {
	// Create event emitter to track Redis failures
	feed := NewActivityFeed(100)
	broadcaster := NewSSEBroadcaster()
	eventEmitter := &EventEmitter{
		feed:        feed,
		broadcaster: broadcaster,
	}
	
	// Use invalid Redis URL to force failures
	cfg := testConfig()
	cfg.RedisURL = "redis://invalid-host:6379/0"
	cfg.Limit = 2
	cfg.Window = time.Second
	cfg.FailureThreshold = 3
	
	drl, err := NewDistributedRateLimiter(cfg, eventEmitter)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Force circuit to be closed initially
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateClosed
	drl.circuitBreaker.failures = 0
	drl.circuitBreaker.mu.Unlock()
	
	ip := "192.168.1.1"
	
	// First request should fail and use fallback
	allowed := drl.Allow(ip)
	if !allowed {
		t.Error("First request should be allowed via fallback")
	}
	
	// Check that Redis failure event was emitted
	events := feed.GetRecentEvents(10)
	foundRedisFailure := false
	for _, event := range events {
		if event.Type == EventTypeRedisFailure {
			foundRedisFailure = true
			if event.Details["operation"] != "rate_limit_check" {
				t.Errorf("Expected operation 'rate_limit_check', got %v", event.Details["operation"])
			}
			break
		}
	}
	
	if !foundRedisFailure {
		t.Error("Redis failure event not found")
	}
	
	// Verify circuit breaker recorded failure
	if drl.circuitBreaker.failures < 1 {
		t.Error("Circuit breaker should have recorded at least one failure")
	}
}

// Test concurrent access to Allow
func TestDistributedRateLimiter_Allow_Concurrent(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	cfg := testConfig()
	cfg.Limit = 100
	cfg.Window = time.Second
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Run concurrent requests
	concurrency := 50
	var wg sync.WaitGroup
	wg.Add(concurrency)
	
	results := make([]bool, concurrency)
	
	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			ip := fmt.Sprintf("192.168.1.%d", idx%10)
			results[idx] = drl.Allow(ip)
		}(i)
	}
	
	wg.Wait()
	
	// Count allowed requests
	allowed := 0
	for _, result := range results {
		if result {
			allowed++
		}
	}
	
	// All requests should be allowed (under limit)
	if allowed != concurrency {
		t.Errorf("Expected %d allowed requests, got %d", concurrency, allowed)
	}
	
	// Verify metrics consistency
	metrics := drl.GetMetrics()
	if metrics.TotalRequests != int64(concurrency) {
		t.Errorf("Expected %d total requests, got %d", concurrency, metrics.TotalRequests)
	}
}

// Test recovery monitor edge cases
func TestDistributedRateLimiter_RecoveryMonitor_EdgeCases(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	cfg := testConfig()
	cfg.RecoveryInterval = 50 * time.Millisecond
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Test 1: Recovery monitor with closed circuit (should do nothing)
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateClosed
	drl.circuitBreaker.mu.Unlock()
	
	time.Sleep(100 * time.Millisecond)
	
	// Circuit should remain closed
	if drl.circuitBreaker.IsOpen() {
		t.Error("Circuit should remain closed")
	}
	
	// Test 2: Recovery monitor with half-open circuit
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateHalfOpen
	drl.circuitBreaker.mu.Unlock()
	
	time.Sleep(100 * time.Millisecond)
	
	// Circuit should not be open
	if drl.circuitBreaker.IsOpen() {
		t.Error("Circuit should not be open when in half-open state")
	}
}

// Test table-driven tests for various scenarios
func TestDistributedRateLimiter_TableDriven(t *testing.T) {
	skipIfRedisUnavailable(t)
	
	testCases := []struct {
		name            string
		limit           int
		window          time.Duration
		requests        int
		expectedAllowed int
		expectedMetrics func(*Metrics) error
	}{
		{
			name:            "all_allowed",
			limit:           10,
			window:          time.Second,
			requests:        5,
			expectedAllowed: 5,
			expectedMetrics: func(m *Metrics) error {
				if m.AllowedRequests != 5 {
					return fmt.Errorf("expected 5 allowed, got %d", m.AllowedRequests)
				}
				if m.RejectedRequests != 0 {
					return fmt.Errorf("expected 0 rejected, got %d", m.RejectedRequests)
				}
				return nil
			},
		},
		{
			name:            "some_rejected",
			limit:           3,
			window:          time.Second,
			requests:        5,
			expectedAllowed: 3,
			expectedMetrics: func(m *Metrics) error {
				if m.AllowedRequests != 3 {
					return fmt.Errorf("expected 3 allowed, got %d", m.AllowedRequests)
				}
				if m.RejectedRequests != 2 {
					return fmt.Errorf("expected 2 rejected, got %d", m.RejectedRequests)
				}
				return nil
			},
		},
		{
			name:            "all_rejected_after_limit",
			limit:           1,
			window:          time.Second,
			requests:        3,
			expectedAllowed: 1,
			expectedMetrics: func(m *Metrics) error {
				if m.AllowedRequests != 1 {
					return fmt.Errorf("expected 1 allowed, got %d", m.AllowedRequests)
				}
				if m.RejectedRequests != 2 {
					return fmt.Errorf("expected 2 rejected, got %d", m.RejectedRequests)
				}
				return nil
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.Limit = tc.limit
			cfg.Window = tc.window
			
			drl, err := NewDistributedRateLimiter(cfg, nil)
			if err != nil {
				t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
			}
			defer func() { _ = drl.Close() }()
			
			ip := "192.168.1.1"
			allowed := 0
			
			for i := 0; i < tc.requests; i++ {
				if drl.Allow(ip) {
					allowed++
				}
			}
			
			if allowed != tc.expectedAllowed {
				t.Errorf("Expected %d allowed requests, got %d", tc.expectedAllowed, allowed)
			}
			
			metrics := drl.GetMetrics()
			if err := tc.expectedMetrics(&metrics); err != nil {
				t.Error(err)
			}
		})
	}
}

// Test AllowWithRequest without Redis (using mocked state)
func TestDistributedRateLimiter_AllowWithRequest_WithoutRedis(t *testing.T) {
	// Create event emitter
	feed := NewActivityFeed(100)
	broadcaster := NewSSEBroadcaster()
	eventEmitter := &EventEmitter{
		feed:        feed,
		broadcaster: broadcaster,
	}
	
	// Use invalid Redis to ensure we're in fallback mode
	cfg := testConfig()
	cfg.RedisURL = "redis://invalid:6379/0"
	cfg.Limit = 2
	cfg.Window = time.Second
	
	drl, err := NewDistributedRateLimiter(cfg, eventEmitter)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer func() { _ = drl.Close() }()
	
	// Force circuit to open so we use fallback
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateOpen
	drl.circuitBreaker.mu.Unlock()
	
	// Create test requests
	req1, _ := http.NewRequest("GET", "/api/test", nil)
	req1.Header.Set("X-Real-IP", "10.0.0.1")
	
	req2, _ := http.NewRequest("POST", "/api/data", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	
	req3, _ := http.NewRequest("PUT", "/api/update", nil)
	req3.RemoteAddr = "10.0.0.1:12345"
	
	// First two requests should be allowed
	if !drl.AllowWithRequest("10.0.0.1", req1) {
		t.Error("First request should be allowed")
	}
	if !drl.AllowWithRequest("10.0.0.1", req2) {
		t.Error("Second request should be allowed")
	}
	
	// Third request should be rejected and emit event
	if drl.AllowWithRequest("10.0.0.1", req3) {
		t.Error("Third request should be rejected")
	}
	
	// Check that rate limit rejection event was emitted
	events := feed.GetRecentEvents(10)
	found := false
	for _, event := range events {
		if event.Type == EventTypeRateLimitRejected {
			found = true
			if event.IP != "10.0.0.1" {
				t.Errorf("Expected IP 10.0.0.1, got %s", event.IP)
			}
			if event.Path != "/api/update" {
				t.Errorf("Expected path /api/update, got %s", event.Path)
			}
			if event.Details["method"] != "PUT" {
				t.Errorf("Expected method PUT, got %v", event.Details["method"])
			}
			break
		}
	}
	
	if !found {
		t.Error("Rate limit rejection event not found")
	}
	
	// Verify metrics
	metrics := drl.GetMetrics()
	if metrics.FallbackMode != "fallback" {
		t.Errorf("Expected fallback mode, got %s", metrics.FallbackMode)
	}
	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}
	if metrics.AllowedRequests != 2 {
		t.Errorf("Expected 2 allowed requests, got %d", metrics.AllowedRequests)
	}
	if metrics.RejectedRequests != 1 {
		t.Errorf("Expected 1 rejected request, got %d", metrics.RejectedRequests)
	}
}

// Test recovery monitor with context cancellation
func TestDistributedRateLimiter_RecoveryMonitor_ContextCancel(t *testing.T) {
	// Use invalid Redis URL
	cfg := testConfig()
	cfg.RedisURL = "redis://invalid:6379/0"
	cfg.RecoveryInterval = 50 * time.Millisecond
	
	drl, err := NewDistributedRateLimiter(cfg, nil)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	
	// Force circuit to open
	drl.circuitBreaker.mu.Lock()
	drl.circuitBreaker.state = StateOpen
	drl.circuitBreaker.mu.Unlock()
	
	// Close the limiter which should stop the recovery monitor
	err = drl.Close()
	if err != nil {
		t.Errorf("Failed to close limiter: %v", err)
	}
	
	// Give some time for goroutine to exit
	time.Sleep(100 * time.Millisecond)
	
	// Circuit should still be open (recovery monitor stopped)
	if !drl.circuitBreaker.IsOpen() {
		t.Error("Circuit should remain open after closing limiter")
	}
}
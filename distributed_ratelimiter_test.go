package main

import (
	"context"
	"fmt"
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
	defer client.Close()
	
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
	defer client.Close()
	
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
	defer client.Close()
	
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
	defer client1.Close()
	defer client2.Close()
	
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
	defer client.Close()
	
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
	
	drl, err := NewDistributedRateLimiter(cfg)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	defer drl.Close()
	
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
	cb.RecordFailure()
	cb.RecordFailure()
	if cb.IsOpen() {
		t.Error("Circuit should still be closed after 2 failures")
	}
	
	// Third failure should open circuit
	cb.RecordFailure()
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
	
	drl, err := NewDistributedRateLimiter(cfg)
	if err != nil {
		t.Fatalf("Failed to create DistributedRateLimiter: %v", err)
	}
	
	// Should still work with fallback
	if drl != nil {
		defer drl.Close()
		
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
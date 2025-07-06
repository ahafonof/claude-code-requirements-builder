# Redis Integration Tests

## Test File: distributed_ratelimiter_test.go

### Test Structure
```go
package main

import (
    "context"
    "testing"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/google/uuid"
)

// Test Redis connection and basic operations
func TestRedisConnection(t *testing.T) {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()
    
    ctx := context.Background()
    
    // Test ping
    err := client.Ping(ctx).Err()
    if err != nil {
        t.Skip("Redis not available, skipping integration tests")
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
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()
    
    ctx := context.Background()
    
    if client.Ping(ctx).Err() != nil {
        t.Skip("Redis not available")
    }
    
    // Define Lua script
    script := `
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
    `
    
    // Test script execution
    now := time.Now().Unix()
    windowStart := now - 60
    
    testCases := []struct {
        name     string
        limit    int
        requests int
        expected int
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
                client.Eval(ctx, script, []string{key}, now, windowStart, tc.limit, requestId)
            }
            
            // Final request should match expected result
            requestId := uuid.New().String()
            result, err := client.Eval(ctx, script, []string{key}, now, windowStart, tc.limit, requestId).Result()
            
            if err != nil {
                t.Errorf("Script execution failed: %v", err)
            }
            
            if result != int64(tc.expected) {
                t.Errorf("Expected %d, got %v", tc.expected, result)
            }
        })
    }
}

// Test distributed scenario with multiple clients
func TestDistributedScenario(t *testing.T) {
    client1 := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    client2 := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    defer client1.Close()
    defer client2.Close()
    
    ctx := context.Background()
    
    if client1.Ping(ctx).Err() != nil {
        t.Skip("Redis not available")
    }
    
    // Same script as above
    script := `...` // Same Lua script
    
    key := "test_distributed"
    client1.Del(ctx, key)
    
    limit := 10
    now := time.Now().Unix()
    windowStart := now - 60
    
    // Client 1 makes 5 requests
    for i := 0; i < 5; i++ {
        requestId := uuid.New().String()
        result, err := client1.Eval(ctx, script, []string{key}, now, windowStart, limit, requestId).Result()
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
        result, err := client2.Eval(ctx, script, []string{key}, now, windowStart, limit, requestId).Result()
        if err != nil {
            t.Errorf("Client2 request %d failed: %v", i, err)
        }
        if result != int64(1) {
            t.Errorf("Client2 request %d rejected unexpectedly", i)
        }
    }
    
    // Next request should be rejected
    requestId := uuid.New().String()
    result, err := client1.Eval(ctx, script, []string{key}, now, windowStart, limit, requestId).Result()
    if err != nil {
        t.Errorf("Final request failed: %v", err)
    }
    if result != int64(0) {
        t.Errorf("Expected rejection, got %v", result)
    }
}

// Test cleanup and TTL
func TestCleanupAndTTL(t *testing.T) {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer client.Close()
    
    ctx := context.Background()
    
    if client.Ping(ctx).Err() != nil {
        t.Skip("Redis not available")
    }
    
    // Test that old entries are cleaned up
    key := "test_cleanup"
    client.Del(ctx, key)
    
    // Add old entry
    oldTimestamp := time.Now().Add(-2 * time.Minute).Unix()
    client.ZAdd(ctx, key, redis.Z{Score: float64(oldTimestamp), Member: "old_request"})
    
    // Add current entry
    now := time.Now().Unix()
    client.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: "current_request"})
    
    // Check initial count
    count, err := client.ZCard(ctx, key).Result()
    if err != nil {
        t.Errorf("ZCard failed: %v", err)
    }
    if count != 2 {
        t.Errorf("Expected 2 entries, got %d", count)
    }
    
    // Remove old entries
    windowStart := now - 60
    client.ZRemRangeByScore(ctx, key, "0", string(windowStart))
    
    // Check count after cleanup
    count, err = client.ZCard(ctx, key).Result()
    if err != nil {
        t.Errorf("ZCard after cleanup failed: %v", err)
    }
    if count != 1 {
        t.Errorf("Expected 1 entry after cleanup, got %d", count)
    }
}
```

### Test Setup Requirements
1. **Redis server**: Must be running on localhost:6379
2. **Dependencies**: go-redis/v9, google/uuid
3. **Test data**: Cleanup test keys after each test
4. **Timeouts**: Handle Redis connection timeouts gracefully

### Test Categories
- **Unit tests**: Lua script logic, Redis operations
- **Integration tests**: Real Redis instance, distributed scenarios  
- **Performance tests**: Latency measurement, throughput
- **Failure tests**: Redis connection failures, recovery

### Test Execution
```bash
# Run with Redis requirement
go test -v -tags=integration

# Skip Redis tests if not available
go test -v -short
```
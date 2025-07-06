# Distributed Rate Limiting Design Document

## Overview
Extend existing in-memory rate limiter to work in distributed environment using Redis for synchronization between servers.

## Requirements
1. **Redis synchronization**: Share rate limiting state across servers
2. **Graceful degradation**: Fallback to local if Redis unavailable
3. **Metrics collection**: Requests, rejections, Redis latency
4. **Monitoring endpoint**: `/metrics` endpoint for operational visibility

## Architecture

### Core Components

#### 1. DistributedRateLimiter
Main orchestration struct that manages Redis and fallback operations.

```go
type DistributedRateLimiter struct {
    redisClient  *redis.Client
    fallbackLimiter *RateLimiter  // Existing implementation
    circuitBreaker *CircuitBreaker
    metrics      *Metrics
    config       *Config
}
```

#### 2. Redis Integration
- **Data Structure**: Sorted Sets for each IP
- **Key Pattern**: `rate_limit:{ip}`
- **Score**: Unix timestamp in milliseconds
- **Value**: Unique request ID (UUID)
- **TTL**: 120 seconds for automatic cleanup

#### 3. Lua Script for Atomicity
```lua
-- Atomic rate limiting operation
-- Remove old entries, count requests, check limit, add new request
local count = redis.call('ZCARD', KEYS[1])
if count >= tonumber(ARGV[3]) then
    return 0  -- Reject
else
    redis.call('ZADD', KEYS[1], ARGV[1], ARGV[4])
    redis.call('EXPIRE', KEYS[1], 120)
    return 1  -- Allow
end
```

#### 4. Circuit Breaker
- **States**: DISTRIBUTED, FALLBACK, CIRCUIT_OPEN
- **Failure Threshold**: 5 failures within 30 seconds
- **Recovery**: Periodic health checks every 10 seconds
- **Metrics**: Track mode switches and failure counts

#### 5. Metrics System
```go
type Metrics struct {
    TotalRequests    int64
    AllowedRequests  int64
    RejectedRequests int64
    RedisLatency     time.Duration
    RedisFailures    int64
    FallbackMode     string
    FallbackCount    int64
    LastUpdated      time.Time
}
```

## Implementation Flow

### Request Processing
1. **Request arrives** → `DistributedRateLimiter.Allow(ip)`
2. **Check circuit state**:
   - If CIRCUIT_OPEN → Use fallback limiter
   - If DISTRIBUTED → Try Redis operation
3. **Redis operation**:
   - Execute Lua script with IP and timestamp
   - Measure latency for metrics
   - Handle failures and update circuit breaker
4. **Update metrics** → Increment counters
5. **Return decision** → Allow/Reject

### Fallback Logic
- Redis failure increments failure counter
- Circuit opens after threshold exceeded
- Periodic health checks attempt Redis reconnection
- Circuit closes when Redis is healthy again

## Configuration
```go
type Config struct {
    RedisURL        string
    Limit           int           // Default: 100
    Window          time.Duration // Default: 1 minute
    FailureThreshold int          // Default: 5
    RecoveryInterval time.Duration // Default: 10 seconds
}
```

## Testing Strategy

### Unit Tests
- `TestDistributedRateLimiter_Allow` - Basic rate limiting
- `TestCircuitBreaker_States` - Circuit breaker transitions
- `TestMetrics_Collection` - Metrics accuracy
- `TestLuaScript_Atomicity` - Redis script behavior

### Integration Tests
- `TestRedisIntegration` - Real Redis instance
- `TestDistributedScenario` - Multiple servers
- `TestFallbackScenario` - Redis failure handling
- `TestRecoveryScenario` - Redis recovery

### Performance Tests
- Redis latency measurement
- Throughput comparison (local vs distributed)
- Memory usage analysis

## Dependencies
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/google/uuid` - Unique request IDs
- Existing codebase (RateLimiter, test framework)

## Migration Strategy
1. **Phase 1**: Add distributed components alongside existing
2. **Phase 2**: Update middleware to use DistributedRateLimiter
3. **Phase 3**: Add /metrics endpoint
4. **Phase 4**: Configuration and deployment

## Operational Considerations
- Monitor Redis connection health
- Alert on circuit breaker state changes
- Track fallback mode usage
- Redis memory usage monitoring
- Performance impact measurement

## Success Criteria
- ✅ Rate limiting works across multiple server instances
- ✅ Service continues during Redis outages
- ✅ Metrics provide operational visibility
- ✅ No performance degradation compared to local rate limiting
- ✅ Comprehensive test coverage (>90%)
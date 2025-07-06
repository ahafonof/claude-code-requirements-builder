# Distributed Rate Limiting - Architectural Decisions

## Redis Client Choice: go-redis/redis v9
**Decision**: Use go-redis/redis over redigo
**Reasoning**: 
- Active development and maintenance
- Built-in connection pooling
- Better context support for timeouts
- Superior error handling
- Easier fallback implementation

## Redis Data Structure: Sorted Sets
**Decision**: Use Redis Sorted Sets (ZADD/ZREMRANGEBYSCORE)
**Reasoning**:
- Efficient time window queries with atomic operations
- Automatic sorting by timestamp
- Built-in cleanup with ZREMRANGEBYSCORE
- True sliding window implementation
- Good performance characteristics

**Alternative considered**: String with TTL - rejected due to fixed window limitation

## Fallback Strategy: Circuit Breaker Pattern
**Decision**: Implement circuit breaker with three states
**States**:
- DISTRIBUTED: Using Redis normally
- FALLBACK: Using in-memory after Redis failure
- CIRCUIT_OPEN: Too many failures, using fallback

**Benefits**:
- Automatic recovery when Redis is restored
- Service continues during Redis outages
- Clear operational visibility

## Atomicity: Redis Lua Script
**Decision**: Use Lua script for atomic rate limiting operations
**Script operations**:
1. Remove old entries (ZREMRANGEBYSCORE)
2. Count current requests (ZCARD)
3. Check limit and add new request (ZADD)
4. Set TTL for cleanup (EXPIRE)

**Benefits**:
- Prevents race conditions between servers
- Reduces Redis round trips
- Consistent behavior across all operations

## Updated
2025-01-05
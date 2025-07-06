# Context Findings

## Existing Architecture Analysis

### Current Rate Limiting System
- **Main Components**:
  - `DistributedRateLimiter` - manages distributed rate limiting with Redis
  - `RateLimiter` - local in-memory rate limiter (fallback)
  - `CircuitBreaker` - handles Redis failures and switches to fallback mode
  - `Metrics` struct - collects system statistics

### Current Metrics Collection
The system already tracks:
- Total requests count
- Allowed/rejected requests
- Redis latency
- Redis failures count
- Fallback mode status
- Circuit breaker state
- Last update timestamp

### Integration Points Identified

1. **RateLimitMiddleware** (ratelimiter.go:68-88)
   - Central point where all requests are processed
   - Already determines allow/reject decisions
   - Can emit events for activity feed

2. **DistributedRateLimiter.Allow()** (distributed_ratelimiter.go:141)
   - Core method that processes rate limit checks
   - Updates metrics
   - Can trigger events on state changes

3. **Circuit Breaker State Changes**
   - `RecordFailure()` - when Redis fails
   - `Reset()` - when returning to distributed mode
   - Important state transitions for monitoring

4. **Existing /metrics endpoint** (main.go:42)
   - Already provides JSON metrics
   - Separate from activity feed (confirmed by user)

### Technology Choice: Server-Sent Events (SSE)

Based on research and requirements:
- **Unidirectional streaming** (server → client) fits our needs
- **Simple implementation** in Go using http.Flusher
- **Auto-reconnection** built into browsers
- **Text-based format** suitable for JSON events
- **Native browser support** for web interface

### Key Events to Track

Based on user requirements (only important events):
1. **Request Rejected** - when rate limit exceeded
2. **Circuit Open** - Redis failures trigger fallback
3. **Circuit Closed** - Redis recovered, returning to distributed
4. **Fallback Mode Activated** - switching to local rate limiting
5. **High Traffic Alert** - optional, when rejection rate spikes

### Files That Need Modification

1. **New file: activity_feed.go**
   - ActivityEvent struct
   - ActivityFeed manager
   - SSE handler implementation

2. **main.go**
   - Add `/activity-feed` SSE endpoint
   - Add `/activity-feed.html` web interface endpoint

3. **distributed_ratelimiter.go**
   - Emit events in Allow() method
   - Emit events on circuit state changes

4. **ratelimiter.go**
   - Emit events in middleware on rejections

### Implementation Patterns to Follow

1. **Concurrent-safe event broadcasting**
   - Use channels for event distribution
   - Mutex protection for client management

2. **Non-blocking event emission**
   - Use buffered channels
   - Drop events if buffer full (don't block main flow)

3. **Clean client disconnection handling**
   - Use request context for detection
   - Proper cleanup of resources

### Technical Constraints

1. **Memory-only storage** (no persistence after restart)
2. **Limited event buffer** (e.g., last 1000 events)
3. **Must not impact rate limiter performance**
4. **Support multiple concurrent SSE clients**
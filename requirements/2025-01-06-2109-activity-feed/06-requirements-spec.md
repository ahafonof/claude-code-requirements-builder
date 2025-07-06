# Requirements Specification: Real-Time Activity Feed

## Problem Statement

The distributed rate limiter system currently collects metrics but lacks a real-time view of system events. Operators need immediate visibility into important system activities such as rate limit violations, circuit breaker state changes, and fallback mode activations to monitor system health and detect issues quickly.

## Solution Overview

Implement a real-time activity feed that streams important system events to web browsers using Server-Sent Events (SSE). The feed will show only significant events (not all requests) with filtering capabilities, providing operators with a live view of system behavior.

## Functional Requirements

### 1. Event Collection
- **FR-1.1**: Capture rate limit rejection events with IP, endpoint, method, and timestamp
- **FR-1.2**: Capture circuit breaker state changes (open/closed)
- **FR-1.3**: Capture fallback mode activations and deactivations
- **FR-1.4**: Generate unique event IDs in format: `{timestamp_nano}-{sequence}`
- **FR-1.5**: Store last 1000 events in memory (circular buffer)

### 2. Real-Time Streaming
- **FR-2.1**: Provide SSE endpoint at `/activity-feed` for real-time event streaming
- **FR-2.2**: Support multiple concurrent SSE clients (max 100)
- **FR-2.3**: Send keepalive messages every 30 seconds to maintain connection
- **FR-2.4**: Automatically send recent events to newly connected clients

### 3. Web Interface
- **FR-3.1**: Serve web interface at `/activity-feed.html`
- **FR-3.2**: Display events in real-time as they arrive
- **FR-3.3**: Show event timestamp, type, and relevant details
- **FR-3.4**: Provide client-side filtering by event type and IP address
- **FR-3.5**: Auto-reconnect on connection loss

### 4. Event Types
- **FR-4.1**: `request_rejected` - Rate limit exceeded
- **FR-4.2**: `circuit_open` - Circuit breaker opened (Redis failures)
- **FR-4.3**: `circuit_closed` - Circuit breaker closed (Redis recovered)
- **FR-4.4**: `fallback_mode` - Switched to local rate limiting
- **FR-4.5**: `redis_failure` - Redis operation failed

## Technical Requirements

### 1. Performance
- **TR-1.1**: Event emission must be non-blocking (use buffered channels)
- **TR-1.2**: Drop events if broadcast buffer full (don't block rate limiter)
- **TR-1.3**: Use goroutines for SSE client management
- **TR-1.4**: Emit events immediately when they occur

### 2. Implementation Details
- **TR-2.1**: Create `activity_feed.go` with ActivityEvent and ActivityFeed types
- **TR-2.2**: Embed HTML interface as string constant in `main.go`
- **TR-2.3**: Use `http.Flusher` interface for SSE streaming
- **TR-2.4**: Protect shared data with appropriate mutexes

### 3. Integration Points
- **TR-3.1**: Emit events in `RateLimitMiddleware` before sending 429 response
- **TR-3.2**: Emit events in `DistributedRateLimiter.Allow()` for state changes
- **TR-3.3**: Emit events in circuit breaker state transitions
- **TR-3.4**: Create global `activityFeed` instance accessible from all components

### 4. Data Structure
```go
type ActivityEvent struct {
    ID        string            `json:"id"`
    Timestamp time.Time         `json:"timestamp"`
    Type      ActivityEventType `json:"type"`
    IP        string            `json:"ip,omitempty"`
    Endpoint  string            `json:"endpoint,omitempty"`
    Method    string            `json:"method,omitempty"`
    Status    int               `json:"status,omitempty"`
    Message   string            `json:"message,omitempty"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
}
```

## Implementation Hints

### 1. SSE Implementation Pattern
```go
w.Header().Set("Content-Type", "text/event-stream")
w.Header().Set("Cache-Control", "no-cache")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("Access-Control-Allow-Origin", "*")

// Use http.Flusher for real-time streaming
flusher, ok := w.(http.Flusher)
```

### 2. Event Broadcasting Pattern
- Use dedicated goroutine for managing clients
- Channel-based communication: register, unregister, broadcast
- Cleanup disconnected clients using request context

### 3. Files to Modify
1. Create `activity_feed.go` - Core activity feed implementation
2. Update `main.go` - Add endpoints and HTML interface
3. Update `distributed_ratelimiter.go` - Emit circuit breaker events
4. Update `ratelimiter.go` - Emit rejection events in middleware

## Acceptance Criteria

1. ✅ Web interface at `/activity-feed.html` shows real-time events
2. ✅ Only important events appear (rejections, state changes, not all requests)
3. ✅ Events can be filtered by type and IP address
4. ✅ Multiple browsers can view the feed simultaneously
5. ✅ System continues to work if no clients are connected
6. ✅ Events appear immediately when they occur
7. ✅ Connection automatically reconnects after network issues
8. ✅ No performance impact on rate limiting operations

## Assumptions

1. No event persistence - feed resets on server restart
2. Maximum 100 concurrent SSE clients is sufficient
3. 1000 event buffer provides adequate history
4. All events have equal importance (no severity levels)
5. Filtering happens client-side for simplicity
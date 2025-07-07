package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event types for activity feed
const (
	EventTypeRateLimitRejected        = "rate_limit_rejected"
	EventTypeCircuitBreakerStateChange = "circuit_breaker_state_change"
	EventTypeRedisFailure             = "redis_failure"
)

// ActivityEvent represents a system event for the activity feed
type ActivityEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	IP        string                 `json:"ip,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ActivityFeed manages a circular buffer of events
type ActivityFeed struct {
	events   []*ActivityEvent
	size     int
	position int
	mu       sync.RWMutex
}

// NewActivityFeed creates a new activity feed with specified buffer size
func NewActivityFeed(size int) *ActivityFeed {
	return &ActivityFeed{
		events: make([]*ActivityEvent, size),
		size:   size,
	}
}

// AddEvent adds an event to the circular buffer
func (af *ActivityFeed) AddEvent(event *ActivityEvent) {
	af.mu.Lock()
	defer af.mu.Unlock()

	af.events[af.position] = event
	af.position = (af.position + 1) % af.size
}

// GetRecentEvents returns the most recent events up to the specified limit
func (af *ActivityFeed) GetRecentEvents(limit int) []*ActivityEvent {
	af.mu.RLock()
	defer af.mu.RUnlock()

	result := make([]*ActivityEvent, 0, limit)
	
	// Calculate how many events we have
	count := 0
	for _, event := range af.events {
		if event != nil {
			count++
		}
	}

	// If we have fewer events than the buffer size, return all
	if count < af.size {
		for i := 0; i < count && len(result) < limit; i++ {
			if af.events[i] != nil {
				result = append(result, af.events[i])
			}
		}
		return result
	}

	// Otherwise, return events in order from oldest to newest
	// Start from position (oldest) and go around
	for i := 0; i < af.size && len(result) < limit; i++ {
		idx := (af.position + i) % af.size
		if af.events[idx] != nil {
			result = append(result, af.events[idx])
		}
	}

	return result
}

// SSEClient represents a Server-Sent Events client
type SSEClient struct {
	Events         chan *ActivityEvent
	Done           chan bool
	ResponseWriter http.ResponseWriter
}

// SSEBroadcaster manages SSE client connections and broadcasts events
type SSEBroadcaster struct {
	clients map[*SSEClient]bool
	mu      sync.RWMutex
}

// NewSSEBroadcaster creates a new SSE broadcaster
func NewSSEBroadcaster() *SSEBroadcaster {
	return &SSEBroadcaster{
		clients: make(map[*SSEClient]bool),
	}
}

// Subscribe adds a new client to the broadcaster
func (b *SSEBroadcaster) Subscribe(w http.ResponseWriter) *SSEClient {
	client := &SSEClient{
		Events:         make(chan *ActivityEvent, 10), // Buffered channel
		Done:           make(chan bool),
		ResponseWriter: w,
	}

	b.mu.Lock()
	b.clients[client] = true
	b.mu.Unlock()

	return client
}

// Unsubscribe removes a client from the broadcaster
func (b *SSEBroadcaster) Unsubscribe(client *SSEClient) {
	b.mu.Lock()
	delete(b.clients, client)
	b.mu.Unlock()
	
	close(client.Events)
	close(client.Done)
}

// GetClientCount returns the number of connected clients
func (b *SSEBroadcaster) GetClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Broadcast sends an event to all connected clients (non-blocking)
func (b *SSEBroadcaster) Broadcast(event *ActivityEvent) {
	b.mu.RLock()
	clients := make([]*SSEClient, 0, len(b.clients))
	for client := range b.clients {
		clients = append(clients, client)
	}
	b.mu.RUnlock()

	// Send to clients without blocking
	for _, client := range clients {
		select {
		case client.Events <- event:
			// Event sent successfully
		default:
			// Client buffer full, drop event (non-blocking behavior)
		}
	}
}

// EventEmitter integrates ActivityFeed and SSEBroadcaster
type EventEmitter struct {
	feed        *ActivityFeed
	broadcaster *SSEBroadcaster
}

// Emit adds an event to the feed and broadcasts it
func (e *EventEmitter) Emit(event *ActivityEvent) {
	e.feed.AddEvent(event)
	e.broadcaster.Broadcast(event)
}

// EmitRateLimitRejection emits a rate limit rejection event
func (e *EventEmitter) EmitRateLimitRejection(r *http.Request) {
	event := &ActivityEvent{
		ID:        fmt.Sprintf("rl-%d", time.Now().UnixNano()),
		Type:      EventTypeRateLimitRejected,
		Timestamp: time.Now(),
		IP:        getClientIP(r),
		Path:      r.URL.Path,
		Details: map[string]interface{}{
			"method": r.Method,
		},
	}
	e.Emit(event)
}

// EmitCircuitBreakerStateChange emits a circuit breaker state change event
func (e *EventEmitter) EmitCircuitBreakerStateChange(oldState, newState string, failures int) {
	event := &ActivityEvent{
		ID:        fmt.Sprintf("cb-%d", time.Now().UnixNano()),
		Type:      EventTypeCircuitBreakerStateChange,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"old_state": oldState,
			"new_state": newState,
			"failures":  failures,
		},
	}
	e.Emit(event)
}

// EmitRedisFailure emits a Redis failure event
func (e *EventEmitter) EmitRedisFailure(operation string, err error) {
	event := &ActivityEvent{
		ID:        fmt.Sprintf("rf-%d", time.Now().UnixNano()),
		Type:      EventTypeRedisFailure,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"operation": operation,
			"error":     err.Error(),
		},
	}
	e.Emit(event)
}
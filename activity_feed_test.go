package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestActivityEvent_Structure verifies the structure of ActivityEvent
func TestActivityEvent_Structure(t *testing.T) {
	// Test event creation and JSON marshaling
	event := &ActivityEvent{
		ID:        "test-123",
		Type:      "rate_limit_rejected",
		Timestamp: time.Now(),
		IP:        "192.168.1.1",
		Path:      "/api/test",
		Details: map[string]interface{}{
			"limit":        100,
			"window":       "1m",
			"current_rate": 105,
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	expectedFields := []string{"id", "type", "timestamp", "ip", "path", "details"}
	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON missing field: %s", field)
		}
	}

	// Test unmarshaling
	var decoded ActivityEvent
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	// Verify values
	if decoded.ID != event.ID {
		t.Errorf("ID mismatch: got %s, want %s", decoded.ID, event.ID)
	}
	if decoded.Type != event.Type {
		t.Errorf("Type mismatch: got %s, want %s", decoded.Type, event.Type)
	}
	if decoded.IP != event.IP {
		t.Errorf("IP mismatch: got %s, want %s", decoded.IP, event.IP)
	}
}

// TestActivityFeed_CircularBuffer tests circular buffer functionality
func TestActivityFeed_CircularBuffer(t *testing.T) {
	// Create feed with small buffer for testing
	feed := NewActivityFeed(5)

	// Add events more than buffer size
	for i := 0; i < 10; i++ {
		event := &ActivityEvent{
			ID:        fmt.Sprintf("event-%d", i),
			Type:      "test_event",
			Timestamp: time.Now(),
		}
		feed.AddEvent(event)
	}

	// Get recent events
	events := feed.GetRecentEvents(10)

	// Should only have last 5 events
	if len(events) != 5 {
		t.Errorf("Expected 5 events, got %d", len(events))
	}

	// Verify we have the latest events (5-9)
	for i, event := range events {
		expectedID := fmt.Sprintf("event-%d", i+5)
		if event.ID != expectedID {
			t.Errorf("Event %d has wrong ID: got %s, want %s", i, event.ID, expectedID)
		}
	}

	// Test GetRecentEvents with limit
	limitedEvents := feed.GetRecentEvents(3)
	if len(limitedEvents) != 3 {
		t.Errorf("Expected 3 events with limit, got %d", len(limitedEvents))
	}
}

// TestActivityFeed_Concurrency tests thread safety
func TestActivityFeed_Concurrency(t *testing.T) {
	feed := NewActivityFeed(1000)
	const numGoroutines = 10
	const eventsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Start multiple goroutines adding events
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				event := &ActivityEvent{
					ID:        fmt.Sprintf("g%d-e%d", goroutineID, j),
					Type:      "concurrent_test",
					Timestamp: time.Now(),
				}
				feed.AddEvent(event)
			}
		}(i)
	}

	// Also read concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				_ = feed.GetRecentEvents(100)
				time.Sleep(time.Microsecond)
			}
		}()
	}

	wg.Wait()

	// Verify all events were added
	events := feed.GetRecentEvents(1000)
	if len(events) != numGoroutines*eventsPerGoroutine {
		t.Errorf("Expected %d events, got %d", numGoroutines*eventsPerGoroutine, len(events))
	}

	// Verify no data corruption by checking unique IDs
	seen := make(map[string]bool)
	for _, event := range events {
		if seen[event.ID] {
			t.Errorf("Duplicate event ID found: %s", event.ID)
		}
		seen[event.ID] = true
	}
}

// TestSSEBroadcaster_ClientManagement tests client connection management
func TestSSEBroadcaster_ClientManagement(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// Test adding clients
	const numClients = 5
	clients := make([]*SSEClient, numClients)

	for i := 0; i < numClients; i++ {
		w := httptest.NewRecorder()
		client := broadcaster.Subscribe(w)
		clients[i] = client
	}

	// Verify client count
	if broadcaster.GetClientCount() != numClients {
		t.Errorf("Expected %d clients, got %d", numClients, broadcaster.GetClientCount())
	}

	// Test removing clients
	for i := 0; i < 3; i++ {
		broadcaster.Unsubscribe(clients[i])
	}

	// Verify updated client count
	if broadcaster.GetClientCount() != numClients-3 {
		t.Errorf("Expected %d clients after unsubscribe, got %d", numClients-3, broadcaster.GetClientCount())
	}

	// Test broadcast reaches remaining clients
	testEvent := &ActivityEvent{
		ID:   "broadcast-test",
		Type: "test_broadcast",
	}

	// Set up goroutines to read from remaining clients
	var wg sync.WaitGroup
	wg.Add(2) // Only 2 clients remain active

	for i := 3; i < numClients; i++ {
		go func(client *SSEClient) {
			defer wg.Done()
			select {
			case event := <-client.Events:
				if event.ID != testEvent.ID {
					t.Errorf("Received wrong event ID: %s", event.ID)
				}
			case <-time.After(time.Second):
				t.Error("Timeout waiting for broadcast event")
			}
		}(clients[i])
	}

	broadcaster.Broadcast(testEvent)
	wg.Wait()
}

// TestSSEBroadcaster_NonBlocking tests non-blocking event emission
func TestSSEBroadcaster_NonBlocking(t *testing.T) {
	broadcaster := NewSSEBroadcaster()

	// Create client with small buffer
	w := httptest.NewRecorder()
	client := &SSEClient{
		Events:       make(chan *ActivityEvent, 1), // Buffer of 1
		Done:        make(chan bool),
		ResponseWriter: w,
	}
	
	// Manually add client to test buffer overflow
	broadcaster.clients[client] = true

	// Send multiple events quickly
	start := time.Now()
	for i := 0; i < 100; i++ {
		event := &ActivityEvent{
			ID:   fmt.Sprintf("nb-event-%d", i),
			Type: "non_blocking_test",
		}
		broadcaster.Broadcast(event)
	}
	duration := time.Since(start)

	// Should complete quickly (non-blocking)
	if duration > 100*time.Millisecond {
		t.Errorf("Broadcast took too long: %v", duration)
	}

	// Verify at least one event was received
	select {
	case event := <-client.Events:
		if event.Type != "non_blocking_test" {
			t.Error("Received wrong event type")
		}
	default:
		t.Error("No events received")
	}
}

// TestEventEmission_Integration tests integration with middleware
func TestEventEmission_Integration(t *testing.T) {
	// Create activity feed and broadcaster
	feed := NewActivityFeed(100)
	broadcaster := NewSSEBroadcaster()
	
	// Create event emitter that integrates both
	emitter := &EventEmitter{
		feed:        feed,
		broadcaster: broadcaster,
	}

	// Subscribe a test client
	w := httptest.NewRecorder()
	client := broadcaster.Subscribe(w)

	// Test rate limit event emission
	rateLimitEvent := &ActivityEvent{
		ID:        "rl-001",
		Type:      EventTypeRateLimitRejected,
		Timestamp: time.Now(),
		IP:        "192.168.1.100",
		Path:      "/api/data",
		Details: map[string]interface{}{
			"limit": 100,
			"rate":  105,
		},
	}

	// Emit event
	emitter.Emit(rateLimitEvent)

	// Verify event was added to feed
	feedEvents := feed.GetRecentEvents(1)
	if len(feedEvents) != 1 {
		t.Fatal("Event not added to feed")
	}
	if feedEvents[0].ID != rateLimitEvent.ID {
		t.Error("Wrong event in feed")
	}

	// Verify event was broadcast
	select {
	case event := <-client.Events:
		if event.ID != rateLimitEvent.ID {
			t.Error("Broadcast event ID mismatch")
		}
		if event.Type != EventTypeRateLimitRejected {
			t.Error("Broadcast event type mismatch")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Broadcast event not received")
	}

	// Test circuit breaker event
	cbEvent := &ActivityEvent{
		ID:        "cb-001",
		Type:      EventTypeCircuitBreakerStateChange,
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"old_state": "closed",
			"new_state": "open",
			"failures":  5,
		},
	}

	emitter.Emit(cbEvent)

	// Verify both events in feed
	allEvents := feed.GetRecentEvents(10)
	if len(allEvents) != 2 {
		t.Errorf("Expected 2 events in feed, got %d", len(allEvents))
	}

	// Test HTTP handler integration
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate rate limit rejection
		emitter.EmitRateLimitRejection(r)
		w.WriteHeader(http.StatusTooManyRequests)
	})

	// Make request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "10.0.0.1:12345"
	rr := httptest.NewRecorder()
	
	handler.ServeHTTP(rr, req)

	// Verify rate limit event was emitted
	time.Sleep(10 * time.Millisecond) // Give time for async emission
	latestEvents := feed.GetRecentEvents(5)
	
	// Find the rate limit event from our request
	found := false
	for _, event := range latestEvents {
		if event.Type == EventTypeRateLimitRejected && event.IP == "10.0.0.1" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("Rate limit rejection event not found in feed")
	}
}

// Helper to create test event emitter
func createTestEmitter() *EventEmitter {
	return &EventEmitter{
		feed:        NewActivityFeed(100),
		broadcaster: NewSSEBroadcaster(),
	}
}

// TestEmitCircuitBreakerStateChange tests circuit breaker event emission
func TestEmitCircuitBreakerStateChange(t *testing.T) {
	emitter := createTestEmitter()
	
	// Subscribe a client to receive events
	w := httptest.NewRecorder()
	client := emitter.broadcaster.Subscribe(w)
	
	// Emit circuit breaker state change
	emitter.EmitCircuitBreakerStateChange("closed", "open", 5)
	
	// Verify event was received
	select {
	case event := <-client.Events:
		if event.Type != EventTypeCircuitBreakerStateChange {
			t.Errorf("Wrong event type: got %s, want %s", event.Type, EventTypeCircuitBreakerStateChange)
		}
		
		// Verify details
		if oldState, ok := event.Details["old_state"].(string); !ok || oldState != "closed" {
			t.Error("Old state not correct")
		}
		if newState, ok := event.Details["new_state"].(string); !ok || newState != "open" {
			t.Error("New state not correct")
		}
		if failures, ok := event.Details["failures"].(int); !ok || failures != 5 {
			t.Error("Failures count not correct")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Circuit breaker event not received")
	}
}

// TestEmitRedisFailure tests Redis failure event emission
func TestEmitRedisFailure(t *testing.T) {
	emitter := createTestEmitter()
	
	// Subscribe a client
	w := httptest.NewRecorder()
	client := emitter.broadcaster.Subscribe(w)
	
	// Emit Redis failure
	testErr := fmt.Errorf("connection refused")
	emitter.EmitRedisFailure("SET", testErr)
	
	// Verify event
	select {
	case event := <-client.Events:
		if event.Type != EventTypeRedisFailure {
			t.Errorf("Wrong event type: got %s, want %s", event.Type, EventTypeRedisFailure)
		}
		
		// Verify details
		if operation, ok := event.Details["operation"].(string); !ok || operation != "SET" {
			t.Error("Operation not correct")
		}
		if errMsg, ok := event.Details["error"].(string); !ok || errMsg != "connection refused" {
			t.Error("Error message not correct")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Redis failure event not received")
	}
}

// TestGetClientIP tests client IP extraction from various headers
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		setupReq   func(*http.Request)
		expectedIP string
	}{
		{
			name: "X-Forwarded-For single IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "192.168.1.100")
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Forwarded-For multiple IPs",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "192.168.1.100, 10.0.0.1, 172.16.0.1")
			},
			expectedIP: "192.168.1.100",
		},
		{
			name: "X-Real-IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "192.168.1.200")
			},
			expectedIP: "192.168.1.200",
		},
		{
			name: "RemoteAddr with port",
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.50:12345"
			},
			expectedIP: "192.168.1.50",
		},
		{
			name: "RemoteAddr without port",
			setupReq: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.60"
			},
			expectedIP: "192.168.1.60",
		},
		{
			name: "Priority: X-Forwarded-For over X-Real-IP",
			setupReq: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "192.168.1.100")
				r.Header.Set("X-Real-IP", "192.168.1.200")
				r.RemoteAddr = "192.168.1.300:8080"
			},
			expectedIP: "192.168.1.100",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupReq(req)
			
			ip := getClientIP(req)
			if ip != tt.expectedIP {
				t.Errorf("getClientIP() = %s, want %s", ip, tt.expectedIP)
			}
		})
	}
}


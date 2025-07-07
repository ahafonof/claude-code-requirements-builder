package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   Response
	}{
		{
			name:           "successful_health_check",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "API is healthy",
				Status:  http.StatusOK,
			},
		},
		{
			name:           "post_method_health_check",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "API is healthy",
				Status:  http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/health", nil)
			rr := httptest.NewRecorder()

			healthHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}

			// Check Content-Type header
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Check response body
			var resp Response
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Message != tt.expectedBody.Message || resp.Status != tt.expectedBody.Status {
				t.Errorf("handler returned unexpected body: got %+v want %+v",
					resp, tt.expectedBody)
			}
		})
	}
}

// TestUsersHandler tests the users endpoint
func TestUsersHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   Response
	}{
		{
			name:           "successful_users_request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "Users endpoint",
				Status:  http.StatusOK,
			},
		},
		{
			name:           "users_request_with_post",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "Users endpoint",
				Status:  http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/users", nil)
			rr := httptest.NewRecorder()

			usersHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}

			// Check Content-Type header
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Check response body
			var resp Response
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Message != tt.expectedBody.Message || resp.Status != tt.expectedBody.Status {
				t.Errorf("handler returned unexpected body: got %+v want %+v",
					resp, tt.expectedBody)
			}
		})
	}
}

// TestProductsHandler tests the products endpoint
func TestProductsHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   Response
	}{
		{
			name:           "successful_products_request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "Products endpoint",
				Status:  http.StatusOK,
			},
		},
		{
			name:           "products_request_with_put",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
			expectedBody: Response{
				Message: "Products endpoint",
				Status:  http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/api/products", nil)
			rr := httptest.NewRecorder()

			productsHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}

			// Check Content-Type header
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Check response body
			var resp Response
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if resp.Message != tt.expectedBody.Message || resp.Status != tt.expectedBody.Status {
				t.Errorf("handler returned unexpected body: got %+v want %+v",
					resp, tt.expectedBody)
			}
		})
	}
}

// TestMetricsHandler tests the metrics endpoint
func TestMetricsHandler(t *testing.T) {
	// Save current state
	originalUseDistributed := useDistributed
	originalDistributedLimiter := distributedLimiter
	defer func() {
		useDistributed = originalUseDistributed
		distributedLimiter = originalDistributedLimiter
	}()

	tests := []struct {
		name                string
		method              string
		setupDistributed    bool
		expectedStatus      int
		expectedMode        string
		expectCircuitState  bool
	}{
		{
			name:             "metrics_with_get_method_in_memory",
			method:           http.MethodGet,
			setupDistributed: false,
			expectedStatus:   http.StatusOK,
			expectedMode:     "in-memory",
			expectCircuitState: false,
		},
		{
			name:             "metrics_with_post_method",
			method:           http.MethodPost,
			setupDistributed: false,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedMode:     "",
			expectCircuitState: false,
		},
		{
			name:             "metrics_with_put_method",
			method:           http.MethodPut,
			setupDistributed: false,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedMode:     "",
			expectCircuitState: false,
		},
		{
			name:             "metrics_with_delete_method",
			method:           http.MethodDelete,
			setupDistributed: false,
			expectedStatus:   http.StatusMethodNotAllowed,
			expectedMode:     "",
			expectCircuitState: false,
		},
		{
			name:             "metrics_with_distributed_limiter",
			method:           http.MethodGet,
			setupDistributed: true,
			expectedStatus:   http.StatusOK,
			expectedMode:     "in-memory", // Will be overridden by distributed limiter
			expectCircuitState: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			if tt.setupDistributed {
				useDistributed = true
				// Create a mock distributed limiter for testing
				distributedLimiter = &DistributedRateLimiter{
					fallbackLimiter: &RateLimiter{
						requests: make(map[string][]time.Time),
						limit:    100,
						window:   time.Minute,
					},
					metrics: &Metrics{
						FallbackMode:     "distributed",
						TotalRequests:    1000,
						AllowedRequests:  950,
						RejectedRequests: 50,
						RedisLatency:     100 * time.Millisecond,
						RedisFailures:    5,
						FallbackCount:    10,
						LastUpdated:      time.Now(),
					},
					circuitBreaker: &CircuitBreaker{
						state: StateClosed,
					},
				}
			} else {
				useDistributed = false
				distributedLimiter = nil
			}

			req := httptest.NewRequest(tt.method, "/metrics", nil)
			rr := httptest.NewRecorder()

			metricsHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}

			// For non-GET methods, we expect no body
			if tt.method != http.MethodGet {
				return
			}

			// Check Content-Type header
			expectedContentType := "application/json"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Check response body
			var metricsData map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&metricsData); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Check mode
			if tt.setupDistributed {
				if mode, ok := metricsData["mode"].(string); !ok || mode != "distributed" {
					t.Errorf("Expected mode 'distributed', got %v", metricsData["mode"])
				}
			} else {
				if mode, ok := metricsData["mode"].(string); !ok || mode != "in-memory" {
					t.Errorf("Expected mode 'in-memory', got %v", metricsData["mode"])
				}
			}

			// Check last_updated exists
			if _, ok := metricsData["last_updated"].(string); !ok {
				t.Error("Expected last_updated field in response")
			}

			// Check circuit state if distributed
			if tt.expectCircuitState {
				if _, ok := metricsData["circuit_state"].(string); !ok {
					t.Error("Expected circuit_state field in response for distributed mode")
				}
			}
		})
	}
}

// TestSSEHandler tests the Server-Sent Events handler
func TestSSEHandler(t *testing.T) {
	// Save and restore global event emitter
	originalEventEmitter := globalEventEmitter
	defer func() {
		globalEventEmitter = originalEventEmitter
	}()

	tests := []struct {
		name                string
		setupEventEmitter   bool
		expectError         bool
		expectedStatus      int
		cancelContext       bool
	}{
		{
			name:              "successful_sse_connection",
			setupEventEmitter: true,
			expectError:       false,
			expectedStatus:    http.StatusOK,
			cancelContext:     true,
		},
		{
			name:              "sse_without_event_emitter",
			setupEventEmitter: false,
			expectError:       true,
			expectedStatus:    http.StatusInternalServerError,
			cancelContext:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test conditions
			if tt.setupEventEmitter {
				globalEventEmitter = &EventEmitter{
					feed:        NewActivityFeed(100),
					broadcaster: NewSSEBroadcaster(),
				}
				// Add some test events
				globalEventEmitter.feed.AddEvent(&ActivityEvent{
					ID:        "test-1",
					Type:      EventTypeRateLimitRejected,
					Timestamp: time.Now(),
					IP:        "192.168.1.1",
					Path:      "/test",
				})
			} else {
				globalEventEmitter = nil
			}

			// Create request with context
			ctx := context.Background()
			if tt.cancelContext {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				// Cancel context after a short delay to simulate client disconnect
				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()
			}

			req := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil)
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			// Use a channel to signal when handler completes
			done := make(chan bool)
			go func() {
				sseHandler(rr, req)
				done <- true
			}()

			// Wait for handler to complete or timeout
			select {
			case <-done:
				// Handler completed
			case <-time.After(200 * time.Millisecond):
				// For successful SSE, we expect it to run until context cancels
				if !tt.expectError {
					// This is expected for successful SSE
				}
			}

			// Check headers for SSE
			if !tt.expectError {
				expectedHeaders := map[string]string{
					"Content-Type":                "text/event-stream",
					"Cache-Control":               "no-cache",
					"Connection":                  "keep-alive",
					"Access-Control-Allow-Origin": "*",
				}

				for header, expectedValue := range expectedHeaders {
					if value := rr.Header().Get(header); value != expectedValue {
						t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, value)
					}
				}

				// Check that some data was written (at least the initial event)
				body := rr.Body.String()
				if !strings.Contains(body, "data:") {
					t.Error("Expected SSE data in response body")
				}
			} else {
				// Check error response
				if rr.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
				}
			}
		})
	}
}

// TestActivityFeedHandler tests the activity feed HTML handler
func TestActivityFeedHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		expectedStatus     int
		expectedContent    []string
		unexpectedContent  []string
	}{
		{
			name:           "successful_activity_feed_request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedContent: []string{
				"<!DOCTYPE html>",
				"<title>Rate Limiter Activity Feed</title>",
				"<h1>",
				"Rate Limiter Activity Feed",
				"<div class=\"stats\">",
				"<div class=\"events\"",
				"EventSource('/api/events/stream')",
				"Total Events",
				"Rate Limit Rejections",
				"Circuit Breaker Changes",
				"Redis Failures",
			},
			unexpectedContent: []string{},
		},
		{
			name:           "activity_feed_with_post_method",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedContent: []string{
				"<!DOCTYPE html>",
				"Rate Limiter Activity Feed",
			},
			unexpectedContent: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/activity-feed", nil)
			rr := httptest.NewRecorder()

			activityFeedHandler(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					rr.Code, tt.expectedStatus)
			}

			// Check Content-Type header
			expectedContentType := "text/html"
			if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("handler returned wrong content type: got %v want %v",
					contentType, expectedContentType)
			}

			// Check response body contains expected content
			body := rr.Body.String()
			for _, expected := range tt.expectedContent {
				if !strings.Contains(body, expected) {
					t.Errorf("Expected body to contain %q", expected)
				}
			}

			// Check response body doesn't contain unexpected content
			for _, unexpected := range tt.unexpectedContent {
				if strings.Contains(body, unexpected) {
					t.Errorf("Expected body not to contain %q", unexpected)
				}
			}

			// Validate HTML structure
			if !strings.HasPrefix(strings.TrimSpace(body), "<!DOCTYPE html>") {
				t.Error("Expected HTML to start with DOCTYPE declaration")
			}

			// Check for proper closing tags
			requiredTags := []string{"</html>", "</head>", "</body>", "</script>", "</style>"}
			for _, tag := range requiredTags {
				if !strings.Contains(body, tag) {
					t.Errorf("Expected HTML to contain closing tag %s", tag)
				}
			}
		})
	}
}

// TestGetEventEmitter tests the GetEventEmitter function
func TestGetEventEmitter(t *testing.T) {
	// Save and restore global event emitter
	originalEventEmitter := globalEventEmitter
	defer func() {
		globalEventEmitter = originalEventEmitter
	}()

	tests := []struct {
		name              string
		setupEventEmitter bool
		expectNil         bool
	}{
		{
			name:              "get_existing_event_emitter",
			setupEventEmitter: true,
			expectNil:         false,
		},
		{
			name:              "get_nil_event_emitter",
			setupEventEmitter: false,
			expectNil:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEventEmitter {
				globalEventEmitter = &EventEmitter{
					feed:        NewActivityFeed(100),
					broadcaster: NewSSEBroadcaster(),
				}
			} else {
				globalEventEmitter = nil
			}

			result := GetEventEmitter()

			if tt.expectNil && result != nil {
				t.Error("Expected nil event emitter, got non-nil")
			}

			if !tt.expectNil && result == nil {
				t.Error("Expected non-nil event emitter, got nil")
			}

			if !tt.expectNil && result != globalEventEmitter {
				t.Error("Expected GetEventEmitter to return globalEventEmitter")
			}
		})
	}
}

// Test helper to verify JSON encoding/decoding
func TestResponseJSONMarshaling(t *testing.T) {
	resp := Response{
		Message: "Test message",
		Status:  200,
	}

	// Test encoding
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&resp); err != nil {
		t.Fatalf("Failed to encode Response: %v", err)
	}

	// Test decoding
	var decoded Response
	if err := json.NewDecoder(&buf).Decode(&decoded); err != nil {
		t.Fatalf("Failed to decode Response: %v", err)
	}

	if decoded.Message != resp.Message || decoded.Status != resp.Status {
		t.Errorf("Decoded response doesn't match original: got %+v, want %+v", decoded, resp)
	}
}
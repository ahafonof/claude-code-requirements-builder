package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with rate limiter (100 requests per minute)
	rateLimited := RateLimitMiddleware(handler)

	t.Run("allows requests under limit", func(t *testing.T) {
		// Should allow 100 requests
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:1234"
			rr := httptest.NewRecorder()

			rateLimited.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Request %d failed: got status %d, want %d", i+1, rr.Code, http.StatusOK)
			}
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		// Reset limiter for new test
		resetRateLimiter()

		// Make 100 requests (should succeed)
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.2:1234"
			rr := httptest.NewRecorder()
			rateLimited.ServeHTTP(rr, req)
		}

		// 101st request should be blocked
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.2:1234"
		rr := httptest.NewRecorder()

		rateLimited.ServeHTTP(rr, req)

		if rr.Code != http.StatusTooManyRequests {
			t.Errorf("Expected rate limit: got status %d, want %d", rr.Code, http.StatusTooManyRequests)
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		resetRateLimiter()

		// IP 1 makes 100 requests
		for i := 0; i < 100; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.3:1234"
			rr := httptest.NewRecorder()
			rateLimited.ServeHTTP(rr, req)
		}

		// IP 2 should still be able to make requests
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.4:1234"
		rr := httptest.NewRecorder()

		rateLimited.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Different IP should not be limited: got status %d, want %d", rr.Code, http.StatusOK)
		}
	})

	t.Run("rate limit resets after time window", func(t *testing.T) {
		t.Skip("Skipping time-based test for now")
		// This would require mocking time or waiting actual time
	})
}

// TestCleanup tests the cleanup function
func TestCleanup(t *testing.T) {
	t.Run("removes expired entries", func(t *testing.T) {
		// Create a custom limiter for testing
		testLimiter := &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		}

		// Add some old and new entries
		now := time.Now()
		oldTime := now.Add(-2 * time.Minute)
		newTime := now.Add(-30 * time.Second)

		testLimiter.requests["192.168.1.1"] = []time.Time{oldTime, oldTime}
		testLimiter.requests["192.168.1.2"] = []time.Time{oldTime, newTime}
		testLimiter.requests["192.168.1.3"] = []time.Time{newTime, newTime}

		// Run cleanup
		testLimiter.cleanup()

		// Check results
		if _, exists := testLimiter.requests["192.168.1.1"]; exists {
			t.Error("IP with only old entries should be removed")
		}

		if len(testLimiter.requests["192.168.1.2"]) != 1 {
			t.Errorf("IP with mixed entries should keep only new ones: got %d, want 1", 
				len(testLimiter.requests["192.168.1.2"]))
		}

		if len(testLimiter.requests["192.168.1.3"]) != 2 {
			t.Errorf("IP with only new entries should keep all: got %d, want 2", 
				len(testLimiter.requests["192.168.1.3"]))
		}
	})

	t.Run("handles empty requests map", func(t *testing.T) {
		testLimiter := &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		}

		// Should not panic
		testLimiter.cleanup()

		if len(testLimiter.requests) != 0 {
			t.Error("Empty map should remain empty after cleanup")
		}
	})

	t.Run("concurrent cleanup and access", func(t *testing.T) {
		testLimiter := &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		}

		// Add initial data
		now := time.Now()
		for i := 0; i < 10; i++ {
			ip := fmt.Sprintf("192.168.1.%d", i)
			testLimiter.requests[ip] = []time.Time{now}
		}

		// Run cleanup and allow concurrently
		done := make(chan bool)
		errors := make(chan error, 100)

		// Start cleanup goroutines
		for i := 0; i < 5; i++ {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("cleanup panic: %v", r)
					}
				}()
				for j := 0; j < 10; j++ {
					testLimiter.cleanup()
					time.Sleep(time.Millisecond)
				}
				done <- true
			}()
		}

		// Start allow goroutines
		for i := 0; i < 5; i++ {
			go func(id int) {
				defer func() {
					if r := recover(); r != nil {
						errors <- fmt.Errorf("allow panic: %v", r)
					}
				}()
				for j := 0; j < 10; j++ {
					ip := fmt.Sprintf("192.168.2.%d", id)
					testLimiter.allow(ip)
					time.Sleep(time.Millisecond)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		close(errors)
		
		// Check for errors
		for err := range errors {
			t.Error(err)
		}
	})
}

// TestInit tests the initialization function
func TestInit(t *testing.T) {
	t.Run("initializes without Redis URL", func(t *testing.T) {
		// Save original env
		originalRedisURL := os.Getenv("REDIS_URL")
		os.Unsetenv("REDIS_URL")
		defer func() {
			if originalRedisURL != "" {
				os.Setenv("REDIS_URL", originalRedisURL)
			}
		}()

		// Reset global variables
		limiter = nil
		distributedLimiter = nil
		useDistributed = false
		globalEventEmitter = nil

		// Call initializeRateLimiter
		initializeRateLimiter()

		// Check results
		if limiter == nil {
			t.Fatal("limiter should be initialized")
		}
		if limiter.limit != 100 {
			t.Errorf("limiter.limit = %d, want 100", limiter.limit)
		}
		if limiter.window != time.Minute {
			t.Errorf("limiter.window = %v, want %v", limiter.window, time.Minute)
		}
		if useDistributed {
			t.Error("useDistributed should be false without Redis URL")
		}
		if distributedLimiter != nil {
			t.Error("distributedLimiter should be nil without Redis URL")
		}
		if globalEventEmitter == nil {
			t.Fatal("globalEventEmitter should be initialized")
		}
		if globalEventEmitter.feed == nil {
			t.Error("globalEventEmitter.feed should be initialized")
		}
		if globalEventEmitter.broadcaster == nil {
			t.Error("globalEventEmitter.broadcaster should be initialized")
		}
	})

	t.Run("attempts Redis initialization with Redis URL", func(t *testing.T) {
		// Set invalid Redis URL to test
		// Using an invalid URL format to trigger ParseURL error
		os.Setenv("REDIS_URL", "invalid://redis:url")
		defer os.Unsetenv("REDIS_URL")

		// Reset global variables
		limiter = nil
		distributedLimiter = nil
		useDistributed = false
		globalEventEmitter = nil

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Call initializeRateLimiter
		initializeRateLimiter()

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output := make([]byte, 1024)
		n, _ := r.Read(output)
		outputStr := string(output[:n])

		// Check results
		if limiter == nil {
			t.Fatal("limiter should be initialized as fallback")
		}
		if useDistributed {
			t.Error("useDistributed should be false with invalid Redis URL")
		}
		if distributedLimiter != nil {
			t.Error("distributedLimiter should be nil with invalid Redis URL")
		}
		if !strings.Contains(outputStr, "Failed to initialize distributed rate limiter") {
			t.Error("Should print error message about failed initialization")
		}
		if !strings.Contains(outputStr, "Falling back to in-memory rate limiter") {
			t.Error("Should print message about falling back")
		}
	})

	t.Run("initializes with valid Redis URL format", func(t *testing.T) {
		// Set a valid Redis URL format (connection will fail but parsing succeeds)
		os.Setenv("REDIS_URL", "redis://localhost:6379")
		defer os.Unsetenv("REDIS_URL")

		// Reset global variables
		limiter = nil
		distributedLimiter = nil
		useDistributed = false
		globalEventEmitter = nil

		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		// Call initializeRateLimiter
		initializeRateLimiter()

		// Restore stdout
		w.Close()
		os.Stdout = oldStdout

		// Read captured output
		output := make([]byte, 1024)
		n, _ := r.Read(output)
		outputStr := string(output[:n])

		// Check results
		if limiter == nil {
			t.Fatal("limiter should be initialized")
		}
		// With valid URL format, distributed limiter will be created
		// even if Redis is not available (it has fallback)
		if !useDistributed {
			t.Error("useDistributed should be true with valid Redis URL format")
		}
		if distributedLimiter == nil {
			t.Error("distributedLimiter should be initialized with valid URL format")
		}
		if !strings.Contains(outputStr, "Using distributed rate limiter with Redis") {
			t.Error("Should print message about using distributed limiter")
		}
	})

	t.Run("cleanup goroutine starts", func(t *testing.T) {
		// This test verifies that the cleanup goroutine is started
		// We can't directly test the goroutine, but we can verify
		// that the system doesn't panic and continues to work

		// Reset
		limiter = nil
		os.Unsetenv("REDIS_URL")

		// Get initial goroutine count
		initialGoroutines := runtime.NumGoroutine()

		// Call initializeRateLimiter
		initializeRateLimiter()

		// Give time for goroutine to start
		time.Sleep(10 * time.Millisecond)

		// Check that we have at least one more goroutine
		newGoroutines := runtime.NumGoroutine()
		if newGoroutines <= initialGoroutines {
			t.Error("Cleanup goroutine should be started")
		}

		// Verify limiter still works
		if !limiter.allow("test-ip") {
			t.Error("Limiter should allow first request")
		}
	})
}

// TestInitWithMockRedis tests init with a mock Redis that succeeds
func TestInitWithMockRedis(t *testing.T) {
	t.Run("successful Redis connection", func(t *testing.T) {
		// This test would require a mock Redis server or test container
		// For now, we skip it but document the test case
		t.Skip("Requires mock Redis server")
		
		// Test would:
		// 1. Start mock Redis server
		// 2. Set REDIS_URL to mock server
		// 3. Call init()
		// 4. Verify useDistributed = true
		// 5. Verify distributedLimiter != nil
		// 6. Verify output contains "Using distributed rate limiter with Redis"
	})
}

// TestCleanupGoroutineLeak tests that cleanup doesn't leak goroutines
func TestCleanupGoroutineLeak(t *testing.T) {
	t.Run("no goroutine leak on repeated cleanup", func(t *testing.T) {
		testLimiter := &RateLimiter{
			requests: make(map[string][]time.Time),
			limit:    100,
			window:   time.Minute,
		}

		// Add many entries
		now := time.Now()
		for i := 0; i < 1000; i++ {
			ip := fmt.Sprintf("192.168.%d.%d", i/256, i%256)
			testLimiter.requests[ip] = []time.Time{now}
		}

		// Get initial goroutine count
		runtime.GC()
		initialGoroutines := runtime.NumGoroutine()

		// Run cleanup many times
		for i := 0; i < 100; i++ {
			testLimiter.cleanup()
		}

		// Check goroutine count hasn't increased
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
		finalGoroutines := runtime.NumGoroutine()

		if finalGoroutines > initialGoroutines {
			t.Errorf("Possible goroutine leak: initial=%d, final=%d", 
				initialGoroutines, finalGoroutines)
		}
	})
}

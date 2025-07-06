package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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

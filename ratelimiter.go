package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// RateLimiter tracks requests per IP
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

var (
	limiter            *RateLimiter
	distributedLimiter *DistributedRateLimiter
	useDistributed     bool
	globalEventEmitter *EventEmitter
)

func init() {
	initializeRateLimiter()
}

// initializeRateLimiter initializes the rate limiting system
func initializeRateLimiter() {
	// Initialize global event emitter
	globalEventEmitter = &EventEmitter{
		feed:        NewActivityFeed(1000),
		broadcaster: NewSSEBroadcaster(),
	}
	
	// Check if Redis URL is provided
	redisURL := os.Getenv("REDIS_URL")
	if redisURL != "" {
		// Try to initialize distributed rate limiter
		cfg := &Config{
			RedisURL:         redisURL,
			Limit:            100,
			Window:           time.Minute,
			FailureThreshold: 5,
			RecoveryInterval: 10 * time.Second,
		}
		
		drl, err := NewDistributedRateLimiter(cfg, globalEventEmitter)
		if err == nil {
			distributedLimiter = drl
			useDistributed = true
			fmt.Println("Using distributed rate limiter with Redis")
		} else {
			fmt.Printf("Failed to initialize distributed rate limiter: %v\n", err)
			fmt.Println("Falling back to in-memory rate limiter")
		}
	}
	
	// Always initialize the fallback limiter
	limiter = &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    100,
		window:   time.Minute,
	}

	// Cleanup old entries periodically
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()
}

// RateLimitMiddleware creates middleware that limits requests per IP
func RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)

		var allowed bool
		if useDistributed && distributedLimiter != nil {
			allowed = distributedLimiter.AllowWithRequest(ip, r)
		} else {
			allowed = limiter.allow(ip)
			// Emit event for local rate limiter too
			if !allowed && globalEventEmitter != nil {
				globalEventEmitter.EmitRateLimitRejection(r)
			}
		}

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"Rate limit exceeded. Maximum 100 requests per minute allowed."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks if request from IP is allowed
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get or create request list for IP
	requests, exists := rl.requests[ip]
	if !exists {
		rl.requests[ip] = []time.Time{now}
		return true
	}

	// Remove old requests outside window
	validRequests := []time.Time{}
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if under limit
	if len(validRequests) >= rl.limit {
		rl.requests[ip] = validRequests
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests
	return true
}

// cleanup removes old entries
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for ip, requests := range rl.requests {
		validRequests := []time.Time{}
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validRequests
		}
	}
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, RemoteAddr might not have a port
		return r.RemoteAddr
	}
	return ip
}

// resetRateLimiter clears all rate limit data (for testing)
func resetRateLimiter() {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	limiter.requests = make(map[string][]time.Time)
}

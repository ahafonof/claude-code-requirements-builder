# Code Patterns

## Rate Limiting Pattern

### Overview
Sliding window rate limiting implementation для Go HTTP API. Обмежує запити per IP адресу.

### Implementation
```go
// RateLimiter structure
type RateLimiter struct {
    mu       sync.RWMutex
    requests map[string][]time.Time  // IP -> timestamps
    limit    int                     // max requests
    window   time.Duration           // time window
}

// Middleware pattern
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := getClientIP(r)
        if !limiter.allow(ip) {
            w.WriteHeader(http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Key Features
- Sliding window algorithm - точніший за fixed window
- Per-IP tracking через map з mutex для thread safety
- Automatic cleanup старих записів через goroutine
- Підтримка proxy headers (X-Forwarded-For, X-Real-IP)

### Testing Approach
- Unit tests для різних сценаріїв
- Test helper `resetRateLimiter()` для очищення стану між тестами
- Окремі тести для різних IP адрес

### Integration
```go
// Apply to all routes
rateLimitedMux := RateLimitMiddleware(mux)
http.ListenAndServe(":8080", rateLimitedMux)
```
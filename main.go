package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "API is healthy",
		Status:  http.StatusOK,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "Users endpoint",
		Status:  http.StatusOK,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "Products endpoint",
		Status:  http.StatusOK,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// metricsHandler returns rate limiter metrics
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	
	// Create metrics response
	metricsData := struct {
		Mode             string    `json:"mode"`
		TotalRequests    int64     `json:"total_requests"`
		AllowedRequests  int64     `json:"allowed_requests"`
		RejectedRequests int64     `json:"rejected_requests"`
		RedisLatency     string    `json:"redis_latency,omitempty"`
		RedisFailures    int64     `json:"redis_failures,omitempty"`
		FallbackCount    int64     `json:"fallback_count,omitempty"`
		LastUpdated      string    `json:"last_updated"`
		CircuitState     string    `json:"circuit_state,omitempty"`
	}{
		Mode: "in-memory",
		LastUpdated: time.Now().Format(time.RFC3339),
	}
	
	// If using distributed limiter, get its metrics
	if useDistributed && distributedLimiter != nil {
		metrics := distributedLimiter.GetMetrics()
		metricsData.Mode = metrics.FallbackMode
		metricsData.TotalRequests = metrics.TotalRequests
		metricsData.AllowedRequests = metrics.AllowedRequests
		metricsData.RejectedRequests = metrics.RejectedRequests
		metricsData.RedisLatency = metrics.RedisLatency.String()
		metricsData.RedisFailures = metrics.RedisFailures
		metricsData.FallbackCount = metrics.FallbackCount
		metricsData.LastUpdated = metrics.LastUpdated.Format(time.RFC3339)
		
		// Add circuit breaker state
		if distributedLimiter.circuitBreaker.IsOpen() {
			metricsData.CircuitState = "open"
		} else {
			metricsData.CircuitState = "closed"
		}
	}
	
	json.NewEncoder(w).Encode(metricsData)
}

func main() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/users", usersHandler)
	mux.HandleFunc("/api/products", productsHandler)
	mux.HandleFunc("/metrics", metricsHandler)

	// Apply rate limiting middleware to all requests
	rateLimitedMux := RateLimitMiddleware(mux)

	log.Println("Starting server on :8080 with rate limiting (100 req/min per IP)")
	log.Fatal(http.ListenAndServe(":8080", rateLimitedMux))
}

package main

import (
	"encoding/json"
	"fmt"
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
	_ = json.NewEncoder(w).Encode(resp)
}

func usersHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "Users endpoint",
		Status:  http.StatusOK,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	resp := Response{
		Message: "Products endpoint",
		Status:  http.StatusOK,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
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
	
	_ = json.NewEncoder(w).Encode(metricsData)
}

// GetEventEmitter returns the global event emitter
func GetEventEmitter() *EventEmitter {
	return globalEventEmitter
}

// sseHandler handles Server-Sent Events connections
func sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Get event emitter
	emitter := GetEventEmitter()
	if emitter == nil {
		http.Error(w, "Event system not initialized", http.StatusInternalServerError)
		return
	}

	// Subscribe client
	client := emitter.broadcaster.Subscribe(w)
	defer emitter.broadcaster.Unsubscribe(client)

	// Send initial events from feed
	recentEvents := emitter.feed.GetRecentEvents(50)
	for _, event := range recentEvents {
		data, err := json.Marshal(event)
		if err != nil {
			continue
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
	}

	// Flush to send initial events
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Keep connection open and send events
	for {
		select {
		case event := <-client.Events:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-client.Done:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// activityFeedHandler serves the HTML interface for activity feed
func activityFeedHandler(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>Rate Limiter Activity Feed</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
        }
        h1 {
            color: #333;
        }
        .stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .stat-label {
            font-size: 14px;
            color: #666;
            margin-bottom: 5px;
        }
        .stat-value {
            font-size: 24px;
            font-weight: bold;
            color: #333;
        }
        .events {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            max-height: 600px;
            overflow-y: auto;
        }
        .event {
            padding: 10px;
            border-left: 3px solid #ddd;
            margin-bottom: 10px;
            background: #fafafa;
        }
        .event.rate_limit_rejected {
            border-left-color: #e74c3c;
            background: #fff5f5;
        }
        .event.circuit_breaker_state_change {
            border-left-color: #f39c12;
            background: #fffaf0;
        }
        .event.redis_failure {
            border-left-color: #e67e22;
            background: #fff8f0;
        }
        .event-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 5px;
        }
        .event-type {
            font-weight: bold;
            font-size: 14px;
        }
        .event-time {
            font-size: 12px;
            color: #666;
        }
        .event-details {
            font-size: 13px;
            color: #444;
        }
        .status {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
            margin-left: 10px;
        }
        .status.connected {
            background: #27ae60;
            color: white;
        }
        .status.disconnected {
            background: #e74c3c;
            color: white;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>
            Rate Limiter Activity Feed
            <span id="connection-status" class="status disconnected">Disconnected</span>
        </h1>
        
        <div class="stats">
            <div class="stat-card">
                <div class="stat-label">Total Events</div>
                <div class="stat-value" id="total-events">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Rate Limit Rejections</div>
                <div class="stat-value" id="rate-limit-rejections">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Circuit Breaker Changes</div>
                <div class="stat-value" id="circuit-breaker-changes">0</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Redis Failures</div>
                <div class="stat-value" id="redis-failures">0</div>
            </div>
        </div>
        
        <h2>Recent Events</h2>
        <div class="events" id="events-container">
            <p>Waiting for events...</p>
        </div>
    </div>

    <script>
        const eventsContainer = document.getElementById('events-container');
        const connectionStatus = document.getElementById('connection-status');
        const stats = {
            total: 0,
            rate_limit_rejected: 0,
            circuit_breaker_state_change: 0,
            redis_failure: 0
        };

        function updateStats() {
            document.getElementById('total-events').textContent = stats.total;
            document.getElementById('rate-limit-rejections').textContent = stats.rate_limit_rejected;
            document.getElementById('circuit-breaker-changes').textContent = stats.circuit_breaker_state_change;
            document.getElementById('redis-failures').textContent = stats.redis_failure;
        }

        function addEvent(event) {
            stats.total++;
            if (stats[event.type] !== undefined) {
                stats[event.type]++;
            }
            updateStats();

            const eventEl = document.createElement('div');
            eventEl.className = 'event ' + event.type;
            
            const time = new Date(event.timestamp).toLocaleTimeString();
            let detailsHtml = '';
            
            if (event.type === 'rate_limit_rejected') {
                detailsHtml = 'IP: ' + event.ip + ', Path: ' + event.path;
            } else if (event.type === 'circuit_breaker_state_change') {
                detailsHtml = 'State: ' + event.details.old_state + ' → ' + event.details.new_state;
                if (event.details.failures) {
                    detailsHtml += ', Failures: ' + event.details.failures;
                }
            } else if (event.type === 'redis_failure') {
                detailsHtml = 'Operation: ' + event.details.operation + ', Error: ' + event.details.error;
            }
            
            eventEl.innerHTML = ` + "`" + `
                <div class="event-header">
                    <span class="event-type">${event.type.replace(/_/g, ' ').toUpperCase()}</span>
                    <span class="event-time">${time}</span>
                </div>
                <div class="event-details">${detailsHtml}</div>
            ` + "`" + `;
            
            // Remove "waiting" message if present
            const waitingMsg = eventsContainer.querySelector('p');
            if (waitingMsg) {
                waitingMsg.remove();
            }
            
            eventsContainer.insertBefore(eventEl, eventsContainer.firstChild);
            
            // Keep only last 100 events in DOM
            while (eventsContainer.children.length > 100) {
                eventsContainer.removeChild(eventsContainer.lastChild);
            }
        }

        function connect() {
            const evtSource = new EventSource('/api/events/stream');
            
            evtSource.onopen = function() {
                connectionStatus.textContent = 'Connected';
                connectionStatus.className = 'status connected';
            };
            
            evtSource.onmessage = function(e) {
                try {
                    const event = JSON.parse(e.data);
                    addEvent(event);
                } catch (err) {
                    console.error('Failed to parse event:', err);
                }
            };
            
            evtSource.onerror = function() {
                connectionStatus.textContent = 'Disconnected';
                connectionStatus.className = 'status disconnected';
                evtSource.close();
                
                // Reconnect after 5 seconds
                setTimeout(connect, 5000);
            };
        }

        // Start connection
        connect();
    </script>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	_, _ = w.Write([]byte(html))
}

func main() {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", healthHandler)
	mux.HandleFunc("/api/users", usersHandler)
	mux.HandleFunc("/api/products", productsHandler)
	mux.HandleFunc("/metrics", metricsHandler)
	
	// Activity feed endpoints
	mux.HandleFunc("/api/events/stream", sseHandler)
	mux.HandleFunc("/activity-feed", activityFeedHandler)

	// Apply rate limiting middleware to all requests
	rateLimitedMux := RateLimitMiddleware(mux)

	log.Println("Starting server on :8080 with rate limiting (100 req/min per IP)")
	log.Fatal(http.ListenAndServe(":8080", rateLimitedMux))
}
package engine

import (
	"net/http"
	"sync"
	"time"
)

// CachedResponse stores the intercepted HTTP response data for future identical requests.
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	CreatedAt  time.Time
}

// RateLimiter tracks client request timestamps in a thread-safe map to enforce sliding-window rate limits.
type IdempotencyEngine struct {
	Resp map[string]CachedResponse
	Mu   sync.RWMutex
}

// NewIdempotencyEngine initializes and returns
// a pointer to a new IdempotencyEngine.
func NewIdempotencyEngine() *IdempotencyEngine {
	return &IdempotencyEngine{
		Resp: make(map[string]CachedResponse),
	}
}

// CleanupWorker
func (IE *IdempotencyEngine) CleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)

	for range ticker.C {
		threshold := time.Now().Add(-24 * time.Hour)
		IE.Mu.Lock()

		for key, cached := range IE.Resp {
			if cached.CreatedAt.Before(threshold) {
				delete(IE.Resp, key)
			}
		}
		IE.Mu.Unlock()
	}
}

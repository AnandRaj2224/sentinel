package middleware

import (
	"net/http"
	"time"

	"github.com/AnandRaj2224/sentinel/internal/engine"
)

// responseRecorder wraps htt.responseWriter
// to capture status code and body of downstream HTTP response.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

// WriteHeader method captures the status Code and passes
// to the underlying ResponseWriter.
func (rr *responseRecorder) WriteHeader(statusCode int) {
	rr.statusCode = statusCode
	rr.ResponseWriter.WriteHeader(statusCode)
}

// WriteHeader method captures the body and passes
// to the underlying ResponseWriter.
func (rr *responseRecorder) Write(b []byte) (int, error) {
	rr.body = append(rr.body, b...)
	return rr.ResponseWriter.Write(b)
}

// IdempotencyMiddleware intercepts requests to serve cached responses if a matching
// Idempotency-Key is found.
func IdempotencyMiddleware(IE *engine.IdempotencyEngine, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")

		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		IE.Mu.RLock()
		cached, exists := IE.Resp[key]
		IE.Mu.RUnlock()

		if exists {
			for key, val := range cached.Headers {
				for _, v := range val {
					w.Header().Set(key, v)
				}
			}
			w.WriteHeader(cached.StatusCode)
			w.Write(cached.Body)
			return
		}
		rr := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rr, r)
		IE.Mu.Lock()
		defer IE.Mu.Unlock()

		newResp := engine.CachedResponse{
			StatusCode: rr.statusCode,
			Body:       rr.body,
			Headers:    w.Header(),
			CreatedAt:  time.Now(),
		}
		IE.Resp[r.Header.Get("Idempotency-Key")] = newResp
	})
}

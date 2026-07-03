package main

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)
// RateLimitMiddleware parses the IP address and checks it against the rate limit,if rate limit exceded
// return a http.StatusTooManyRequests.
func RateLimitMiddleware(limiter *RateLimiter, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			logger.Error("failed while parsing IP", err)
		}
		if limiter.Allow(ip) {
			next.ServeHTTP(w, r)
			return
		}
		logger.Warn("rate limit was exceeded, IP:", ip)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	})
}

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

// CachedResponse stores the intercepted HTTP response data for future identical requests.
type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// RateLimiter tracks client request timestamps in a thread-safe map to enforce sliding-window rate limits.
type IdempotencyEngine struct {
	resp map[string]CachedResponse
	mu   sync.RWMutex
}

// NewIdempotencyEngine initializes and returns
// a pointer to a new IdempotencyEngine.
func NewIdempotencyEngine() *IdempotencyEngine {
	return &IdempotencyEngine{
		resp: make(map[string]CachedResponse),
	}
}

// IdempotencyMiddleware intercepts requests to serve cached responses if a matching
// Idempotency-Key is found.
func IdempotencyMiddleware(IE *IdempotencyEngine, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Idempotency-Key")

		if key == "" {
			next.ServeHTTP(w, r)
			return
		}

		IE.mu.RLock()
		cached, exists := IE.resp[key]
		IE.mu.RUnlock()

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
		IE.mu.Lock()
		defer IE.mu.Unlock()

		newResp := CachedResponse{
			StatusCode: rr.statusCode,
			Body:       rr.body,
			Headers:    w.Header(),
		}
		IE.resp[r.Header.Get("Idempotency-Key")] = newResp
	})
}

// RateLimiter struct contains visitors
// that maps ip address to requests and
// the frequency / no of time in a time period.
type RateLimiter struct {
	visitors map[string][]time.Time
	mu       sync.RWMutex
}

// NewRateLimiter initalises a new RateLimiter struct
// and return it address.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		visitors: make(map[string][]time.Time),
	}
}

// Allow implements a sliding window algo for
// blocking the requests that exceeds the limits.
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	var filtered []time.Time
	threshold := time.Now().Add(-10 * time.Second)

	for _, t := range rl.visitors[ip] {
		if t.After(threshold) {
			filtered = append(filtered, t)
		}
	}
	rl.visitors[ip] = filtered
	if len(rl.visitors[ip]) < 5 {
		rl.visitors[ip] = append(rl.visitors[ip], time.Now())
		return true
	}
	return false
}

// LoggingMiddleware intercepts inbound requests to measure latency and output structured JSON telemetry.
func LoggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start).String()

		logger.Info("request completed",
			slog.String("HTTP method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("duration", duration),
		)
	})
}

func main() {

	// load the env so they are avalible to use.
	err := godotenv.Load()
	if err != nil {
		fmt.Println("error loading env!")
	}

	// port for the running server -> current proxy.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// target URL for the protected server.
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:9000"
	}

	// parsing the main severs URL.
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal("failed to parse the url!", err)
	}

	// creating a new slog object
	// review
	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)

	// review
	logger := slog.New(jsonHandler)

	// initialising the Rate Limiter.
	limiter := NewRateLimiter()

	// creating a new reverse Proxy with with main servers parsed URL.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	// logging contain the function + closure object returned from loggingMiddleware.
	logging := LoggingMiddleware(logger, proxy)

	logger.Info("Starting Sentinel", "port", port, "target_url", targetURL)
	log.Fatal(http.ListenAndServe(":"+port, logging))

}

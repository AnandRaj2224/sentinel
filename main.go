package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

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

// logging function that logs HTTP method,path,and duration to complete the request
// in json format using slog.
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

package middleware

import (
	"log/slog"
	"net"
	"net/http"

	"github.com/AnandRaj2224/sentinel/internal/engine"
)

// RateLimitMiddleware parses the IP address and checks it against the rate limit,if rate limit exceded
// return a http.StatusTooManyRequests.
func RateLimitMiddleware(limiter *engine.RateLimiter, logger *slog.Logger, next http.Handler) http.Handler {
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
	})
}

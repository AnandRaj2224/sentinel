package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

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

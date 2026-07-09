package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/AnandRaj2224/sentinel/internal/engine"
	"github.com/AnandRaj2224/sentinel/internal/middleware"
)

// DynamicRouter is a function that takes the route of an incomming
// request matches against predefined map of routes if passes creates a
// new proxy server for that route.
func DynamicRouter(routes map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		target, exists := routes[path]
		if exists == false {
			http.Error(w, "Route not found", http.StatusNotFound)
			return
		}
		parsedURL, err := url.Parse(target)
		if err != nil {
			http.Error(w, "failed Parsing the route", http.StatusInternalServerError)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(parsedURL)
		proxy.ServeHTTP(w, r)
	})
}

func main() {

	port := "8000"
	routes := map[string]string{
		"/api/users":    "http://localhost:9000",
		"/api/payments": "http://localhost:9001",
	}
	router := DynamicRouter(routes)

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)

	limiter := engine.NewRateLimiter()
	idempEngine := engine.NewIdempotencyEngine()

	idempHandler := middleware.IdempotencyMiddleware(idempEngine, router)
	rlHandler := middleware.RateLimitMiddleware(limiter, logger, idempHandler)
	finalHandler := middleware.LoggingMiddleware(logger, rlHandler)

	logger.Info("Starting Sentinel", "port", port)
	go limiter.CleanupWorker()
	go idempEngine.CleanupWorker()
	log.Fatal(http.ListenAndServe(":"+port, finalHandler))

}

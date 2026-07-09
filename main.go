package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/AnandRaj2224/sentinel/internal/engine"
	"github.com/AnandRaj2224/sentinel/internal/middleware"
	"github.com/joho/godotenv"
)

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

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)

	limiter := engine.NewRateLimiter()
	idempEngine := engine.NewIdempotencyEngine()

	// creating a new reverse Proxy with with main servers parsed URL.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)

	idempHandler := middleware.IdempotencyMiddleware(idempEngine, proxy)
	rlHandler := middleware.RateLimitMiddleware(limiter, logger, idempHandler)
	finalHandler := middleware.LoggingMiddleware(logger, rlHandler)

	logger.Info("Starting Sentinel", "port", port, "target_url", targetURL)
	go limiter.CleanupWorker()
	go idempEngine.CleanupWorker()
	log.Fatal(http.ListenAndServe(":"+port, finalHandler))

}

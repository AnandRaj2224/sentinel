package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/joho/godotenv"
)

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

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)

	logger := slog.New(jsonHandler)

	// creating a new reverse Proxy with with main servers parsed URL.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	// logging contain the function + closure object returned from loggingMiddleware.
	logging := LoggingMiddleware(logger, proxy)

	logger.Info("Starting Sentinel", "port", port, "target_url", targetURL)
	log.Fatal(http.ListenAndServe(":"+port, logging))

}

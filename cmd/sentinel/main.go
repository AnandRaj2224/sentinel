package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"

	"github.com/AnandRaj2224/sentinel/internal/engine"
	"github.com/AnandRaj2224/sentinel/internal/middleware"
)

// RouteState a struct to hold the state and lock for a single route.
type RouteState struct {
	Count int
	Mu    sync.Mutex
}

// DynamicRouter is a function that takes the route of an incoming
// request matches against predefined map of routes if passes creates a
// new proxy server for that route.
func DynamicRouter(routes map[string]engine.RouteConfig) http.Handler {
	routeStates := make(map[string]*RouteState)

	for path := range routes {
		routeStates[path] = &RouteState{
			Count: 0,
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeFile(w, r, "index.html")
			return
		}

		target, exists := routes[path]

		if !exists {
			http.Error(w, "Route not found", http.StatusNotFound)
			return
		}
		if len(target.TargetURLs) == 0 {
			http.Error(w, "No backend servers available for this route", http.StatusBadGateway)
			return
		}

		state := routeStates[path]

		state.Mu.Lock()
		currCount := state.Count
		state.Count = (currCount + 1) % len(target.TargetURLs)
		state.Mu.Unlock()

		finalURL := target.TargetURLs[currCount]
		parsedURL, err := url.Parse(finalURL)
		if err != nil {
			http.Error(w, "failed Parsing the route", http.StatusInternalServerError)
			return
		}

		proxy := httputil.NewSingleHostReverseProxy(parsedURL)
		proxy.ServeHTTP(w, r)
	})
}

// loadRoutes function takes the routes file
// reads it and create a custom routes map.
func loadRoutes(filename string) map[string]engine.RouteConfig {
	file, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("Failed to Read File:", err)
	}
	routes := make(map[string]engine.RouteConfig)
	err = json.Unmarshal(file, &routes)
	if err != nil {
		log.Fatal("Failed to Read File Content:", file, err)
	}
	return routes
}

func main() {

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	var routes map[string]engine.RouteConfig

	routesEnv := os.Getenv("ROUTES_JSON")

	if routesEnv != "" {
		err := json.Unmarshal([]byte(routesEnv), &routes)
		if err != nil {
			log.Fatal("Failed to parse ROUTES_JSON from environment: ", err)
		}
	} else {
		routes = loadRoutes("routes.json")
	}

	router := DynamicRouter(routes)

	jsonHandler := slog.NewJSONHandler(os.Stdout, nil)
	logger := slog.New(jsonHandler)

	limiter := engine.NewRateLimiter()
	idempEngine := engine.NewIdempotencyEngine()

	idempHandler := middleware.IdempotencyMiddleware(idempEngine, router)
	rlHandler := middleware.RateLimitMiddleware(limiter, routes, logger, idempHandler)
	finalHandler := middleware.LoggingMiddleware(logger, rlHandler)

	go limiter.CleanupWorker()
	go idempEngine.CleanupWorker()

	logger.Info("Starting Sentinel", "port", port)
	log.Fatal(http.ListenAndServe(":"+port, finalHandler))

}

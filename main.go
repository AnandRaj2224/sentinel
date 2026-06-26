package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/joho/godotenv"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		log.Printf("Incoming request: %v\n %v", r.Method, r.URL.Path)

		next.ServeHTTP(w, r)

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
	// creating a new reverse Proxy with with main servers parsed URL.
	proxy := httputil.NewSingleHostReverseProxy(parsedURL)
	// logging contain the function + closure object returned from loggingMiddleware.
	logging := LoggingMiddleware(proxy)

	fmt.Printf("Starting Sentinel on port %v, forwarding to %v\n", port, targetURL)
	log.Fatal(http.ListenAndServe(":"+port, logging))

}

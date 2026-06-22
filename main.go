package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

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

	fmt.Printf("Starting Sentinel on port %v, forwarding to %v",port,targetURL)
}

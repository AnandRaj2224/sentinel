package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9001" // Fallback safety
	}

	log.Printf("Starting Protected Backend Instance on port %s", port)

	err := http.ListenAndServe(":"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		msg := fmt.Sprintf(`{"message": "Hello from Target Server Instance running on port %s"}`, port)
		w.Write([]byte(msg))
	}))

	if err != nil {
		log.Fatal(err)
	}
}

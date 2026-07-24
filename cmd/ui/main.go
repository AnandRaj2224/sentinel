package main

import (
	"log"
	"net/http"
)

func main() {

	err := http.ListenAndServe("localhost:8081", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	}))
	if err != nil {
		log.Fatal(err)
	}
}

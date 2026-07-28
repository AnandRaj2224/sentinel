package main

import (
	"embed"
	"log"
	"net/http"
)

//go:embed index.html
var files embed.FS

func main() {
	page, err := files.ReadFile("index.html")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
)

func main() {

	ports := []string{"9001", "9002"}
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		go func(port string) {
			defer wg.Done()
			err := http.ListenAndServe(":"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				msg := fmt.Sprintf(`{"message": "Hello from Target Server %s"}`, port)
				w.Write([]byte(msg))
			}))
			if err != nil {
				log.Fatal(err)
			}

		}(port)

	}
	wg.Wait()
}

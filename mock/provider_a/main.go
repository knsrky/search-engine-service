package main

import (
	_ "embed"
	"log"
	"net/http"
	"time"
)

//go:embed data.json
var jsonData []byte

func main() {
	http.HandleFunc("/api/contents", func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency (50-200ms)
		time.Sleep(time.Duration(50+time.Now().UnixNano()%150) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Provider", "provider-a")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(jsonData); err != nil {
			log.Printf("[Provider A] Write error: %v", err)
		}

		log.Printf("[Provider A] %s %s - 200 OK", r.Method, r.URL.Path)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
			log.Printf("[Provider A] Health write error: %v", err)
		}
	})

	log.Println("Mock Provider A running on :8081")
	server := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}

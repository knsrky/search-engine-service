package main

import (
	_ "embed"
	"log"
	"net/http"
	"time"
)

//go:embed data.xml
var xmlData []byte

func main() {
	http.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		// Simulate network latency (100-300ms)
		time.Sleep(time.Duration(100+time.Now().UnixNano()%200) * time.Millisecond)

		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("X-Provider", "provider-b")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(xmlData); err != nil {
			log.Printf("[Provider B] Write error: %v", err)
		}

		log.Printf("[Provider B] %s %s - 200 OK", r.Method, r.URL.Path)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`<health><status>healthy</status></health>`)); err != nil {
			log.Printf("[Provider B] Health write error: %v", err)
		}
	})

	log.Println("Mock Provider B running on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}

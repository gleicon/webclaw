package main

import (
	"log"
	"net/http"
	"strings"
)

func main() {
	mux := http.NewServeMux()

	// Test endpoint for jsFetch smoke test
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("WebClaw jsFetch test response - OK"))
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".wasm.br") {
			w.Header().Set("Content-Type", "application/wasm")
			w.Header().Set("Content-Encoding", "br")
			w.Header().Set("Vary", "Accept-Encoding")
		}
		http.FileServer(http.Dir(".")).ServeHTTP(w, r)
	})
	log.Println("Serving on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

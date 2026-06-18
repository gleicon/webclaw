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

	// Serve just-bash browser.js from node_modules in dev (vite-plugin-static-copy only runs at build time)
	mux.HandleFunc("/vendor/browser.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "node_modules/just-bash/dist/bundle/browser.js")
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
	log.Println("Static file server for WebClaw WASM")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

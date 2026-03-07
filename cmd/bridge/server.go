package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// StartServer initializes and starts the HTTP server
func StartServer(port string, initialOTP string) error {
	// Store initial OTP
	storeOTP(initialOTP)

	// Create a new ServeMux to avoid global state
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/auth/otp", handleOTPAuth)
	mux.HandleFunc("/health", handleHealth)

	// Protected routes
	mux.HandleFunc("/file/read", authMiddleware(handleFileRead))
	mux.HandleFunc("/file/write", authMiddleware(handleFileWrite))
	mux.HandleFunc("/file/list", authMiddleware(handleFileList))
	mux.HandleFunc("/exec", authMiddleware(handleExec))
	mux.HandleFunc("/git/clone", authMiddleware(handleGitClone))
	mux.HandleFunc("/git/commit", authMiddleware(handleGitCommit))
	mux.HandleFunc("/git/push", authMiddleware(handleGitPush))

	// CORS headers for browser access
	handler := corsMiddleware(mux)

	// Bind ONLY to localhost (127.0.0.1)
	addr := fmt.Sprintf("127.0.0.1:%s", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	server := &http.Server{
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Bridge running on http://%s", addr)
	return server.Serve(listener)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from localhost and file://
		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string) bool {
	allowed := []string{
		"http://localhost",
		"http://127.0.0.1",
		"https://localhost",
		"https://127.0.0.1",
		"file://",
		"null", // For file:// origins
	}
	for _, prefix := range allowed {
		if strings.HasPrefix(origin, prefix) {
			return true
		}
	}
	return false
}

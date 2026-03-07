package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	otpStore   = make(map[string]OTPInfo)
	tokenStore = make(map[string]TokenInfo)
	storeMutex sync.RWMutex
)

// OTPInfo stores OTP metadata
type OTPInfo struct {
	Expires time.Time
	Used    bool
}

// TokenInfo stores bearer token metadata
type TokenInfo struct {
	Created time.Time
	Expires time.Time
}

// generateOTP creates a 6-digit numeric code using crypto/rand
func generateOTP() string {
	b := make([]byte, 4)
	rand.Read(b) //nolint:errcheck
	n := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if n < 0 {
		n = -n
	}
	return fmt.Sprintf("%06d", n%1000000)
}

// storeOTP saves an OTP with expiration
func storeOTP(otp string) {
	storeMutex.Lock()
	defer storeMutex.Unlock()
	otpStore[otp] = OTPInfo{
		Expires: time.Now().Add(5 * time.Minute),
		Used:    false,
	}
}

// validateOTP checks if OTP is valid and not expired/used
func validateOTP(otp string) bool {
	storeMutex.Lock()
	defer storeMutex.Unlock()

	info, exists := otpStore[otp]
	if !exists || info.Used || time.Now().After(info.Expires) {
		return false
	}

	// Mark as used
	info.Used = true
	otpStore[otp] = info
	return true
}

// generateToken creates a bearer token using crypto/rand
func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	return hex.EncodeToString(b)
}

// storeToken saves a bearer token with expiration
func storeToken(token string) {
	storeMutex.Lock()
	defer storeMutex.Unlock()
	tokenStore[token] = TokenInfo{
		Created: time.Now(),
		Expires: time.Now().Add(24 * time.Hour),
	}
}

// validateToken checks if bearer token is valid
func validateToken(token string) bool {
	storeMutex.RLock()
	defer storeMutex.RUnlock()

	info, exists := tokenStore[token]
	if !exists || time.Now().After(info.Expires) {
		return false
	}
	return true
}

// handleOTPAuth validates OTP and returns bearer token
func handleOTPAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OTP string `json:"otp"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if !validateOTP(req.OTP) {
		http.Error(w, "Invalid or expired OTP", http.StatusUnauthorized)
		return
	}

	token := generateToken()
	storeToken(token)

	resp := struct {
		Token   string `json:"token"`
		Expires string `json:"expires"`
	}{
		Token:   token,
		Expires: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// authMiddleware validates bearer token on protected routes
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization", http.StatusUnauthorized)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		if !validateToken(token) {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

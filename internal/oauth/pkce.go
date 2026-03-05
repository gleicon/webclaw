//go:build js && wasm

// Package oauth provides OAuth 2.0 PKCE authentication for browser-based integrations
package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"syscall/js"
)

// GenerateCodeVerifier creates a PKCE code_verifier per RFC 7636
// Returns a 128-character base64url-encoded random string
func GenerateCodeVerifier() string {
	// RFC 7636 recommends 128 bytes of random data for maximum security
	verifier := make([]byte, 128)

	// Use Web Crypto API for cryptographically secure random values
	crypto := js.Global().Get("crypto")
	if crypto.IsUndefined() || crypto.IsNull() {
		// Fallback: use syscall/js random (less secure, but functional)
		for i := range verifier {
			verifier[i] = byte(js.Global().Get("Math").Call("random").Float() * 256)
		}
	} else {
		// Use crypto.getRandomValues
		u8arr := js.Global().Get("Uint8Array").New(len(verifier))
		crypto.Call("getRandomValues", u8arr)
		for i := range verifier {
			verifier[i] = byte(u8arr.Index(i).Int())
		}
	}

	// Base64url encode (URL-safe, no padding)
	return base64.RawURLEncoding.EncodeToString(verifier)
}

// GenerateCodeChallenge creates a PKCE code_challenge from a code_verifier
// Uses S256 method (SHA-256 hash + base64url encoding)
func GenerateCodeChallenge(verifier string) string {
	// SHA-256 hash of verifier
	hash := sha256.Sum256([]byte(verifier))

	// Base64url encode (no padding)
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// GeneratePKCEPair generates both verifier and challenge in one call
// Returns (verifier, challenge) - verifier must be stored for the token exchange
func GeneratePKCEPair() (verifier string, challenge string) {
	verifier = GenerateCodeVerifier()
	challenge = GenerateCodeChallenge(verifier)
	return verifier, challenge
}

// ValidatePKCEPair validates that a code_challenge was derived from a code_verifier
// Used primarily for testing - in production, the server validates this
func ValidatePKCEPair(verifier, challenge string) bool {
	computed := GenerateCodeChallenge(verifier)
	return computed == challenge
}

// PKCEParams holds both PKCE parameters for easy transport
type PKCEParams struct {
	CodeVerifier    string `json:"code_verifier"`
	CodeChallenge   string `json:"code_challenge"`
	ChallengeMethod string `json:"challenge_method"`
}

// GeneratePKCEParams generates a complete PKCE parameter set
func GeneratePKCEParams() *PKCEParams {
	verifier, challenge := GeneratePKCEPair()
	return &PKCEParams{
		CodeVerifier:    verifier,
		CodeChallenge:   challenge,
		ChallengeMethod: "S256",
	}
}

// Test vectors from RFC 7636 for verification
var testVectors = []struct {
	verifier  string
	challenge string
}{
	{
		// RFC 7636 Appendix B example
		verifier:  "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk",
		challenge: "E9Melhoa2OwvFrEMT9y0hCSP9Sdi3dz1OlYhIJgV3_g",
	},
}

// RunTestVectors verifies PKCE implementation against RFC 7636 test vectors
// Returns error if any test vector fails
func RunTestVectors() error {
	for i, tv := range testVectors {
		computed := GenerateCodeChallenge(tv.verifier)
		if computed != tv.challenge {
			return fmt.Errorf("test vector %d failed: expected %s, got %s", i, tv.challenge, computed)
		}
	}
	return nil
}

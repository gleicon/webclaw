//go:build js && wasm

package oauth

import (
	"strings"
	"testing"
)

// TestGenerateCodeVerifier verifies code verifier generation
func TestGenerateCodeVerifier(t *testing.T) {
	verifier := GenerateCodeVerifier()

	// RFC 7636: verifier must be 43-128 characters
	if len(verifier) < 43 {
		t.Errorf("verifier too short: got %d chars, minimum 43", len(verifier))
	}
	if len(verifier) > 128 {
		t.Errorf("verifier too long: got %d chars, maximum 128", len(verifier))
	}

	// Must be base64url encoded (only A-Z, a-z, 0-9, -, _)
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for _, c := range verifier {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("verifier contains invalid character: %c", c)
		}
	}

	// Must not contain padding (=)
	if strings.Contains(verifier, "=") {
		t.Error("verifier must not contain padding (=)")
	}

	// Must be URL-safe (no + or /)
	if strings.Contains(verifier, "+") || strings.Contains(verifier, "/") {
		t.Error("verifier contains non-URL-safe characters (+ or /)")
	}
}

// TestGenerateCodeChallenge verifies code challenge generation
func TestGenerateCodeChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expectedChallenge := "E9Melhoa2OwvFrEMT9y0hCSP9Sdi3dz1OlYhIJgV3_g"

	challenge := GenerateCodeChallenge(verifier)

	if challenge != expectedChallenge {
		t.Errorf("challenge mismatch: expected %s, got %s", expectedChallenge, challenge)
	}

	// Challenge must be base64url encoded
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	for _, c := range challenge {
		if !strings.ContainsRune(validChars, c) {
			t.Errorf("challenge contains invalid character: %c", c)
		}
	}

	// SHA-256 produces 32 bytes, base64url encoded is 43 characters
	if len(challenge) != 43 {
		t.Errorf("challenge length wrong: expected 43, got %d", len(challenge))
	}
}

// TestGeneratePKCEPair verifies pair generation
func TestGeneratePKCEPair(t *testing.T) {
	verifier, challenge := GeneratePKCEPair()

	// Both should be non-empty
	if verifier == "" {
		t.Error("verifier is empty")
	}
	if challenge == "" {
		t.Error("challenge is empty")
	}

	// Challenge should validate against verifier
	if !ValidatePKCEPair(verifier, challenge) {
		t.Error("challenge does not validate against verifier")
	}

	// Different verifier should produce different challenge
	verifier2, challenge2 := GeneratePKCEPair()
	if verifier == verifier2 {
		t.Error("generated same verifier twice (unlikely but possible)")
	}
	if challenge == challenge2 {
		t.Error("generated same challenge twice (unlikely but possible)")
	}
}

// TestGeneratePKCEParams verifies params struct generation
func TestGeneratePKCEParams(t *testing.T) {
	params := GeneratePKCEParams()

	if params.CodeVerifier == "" {
		t.Error("CodeVerifier is empty")
	}
	if params.CodeChallenge == "" {
		t.Error("CodeChallenge is empty")
	}
	if params.ChallengeMethod != "S256" {
		t.Errorf("ChallengeMethod should be S256, got %s", params.ChallengeMethod)
	}

	// Verify the challenge validates against the verifier
	if !ValidatePKCEPair(params.CodeVerifier, params.CodeChallenge) {
		t.Error("CodeChallenge does not validate against CodeVerifier")
	}
}

// TestValidatePKCEPair verifies validation logic
func TestValidatePKCEPair(t *testing.T) {
	// Valid pair
	verifier := "test_verifier_123"
	challenge := GenerateCodeChallenge(verifier)
	if !ValidatePKCEPair(verifier, challenge) {
		t.Error("valid pair failed validation")
	}

	// Invalid pair
	if ValidatePKCEPair(verifier, "wrong_challenge") {
		t.Error("invalid pair passed validation")
	}
}

// TestRunTestVectors verifies RFC 7636 test vectors
func TestRunTestVectors(t *testing.T) {
	if err := RunTestVectors(); err != nil {
		t.Errorf("RFC 7636 test vectors failed: %v", err)
	}
}

// TestRFC7636AppendixB specifically tests the RFC 7636 Appendix B example
func TestRFC7636AppendixB(t *testing.T) {
	// From RFC 7636 Appendix B
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	expectedChallenge := "E9Melhoa2OwvFrEMT9y0hCSP9Sdi3dz1OlYhIJgV3_g"

	challenge := GenerateCodeChallenge(verifier)

	if challenge != expectedChallenge {
		t.Errorf("RFC 7636 Appendix B test vector failed:\nexpected: %s\ngot:      %s", expectedChallenge, challenge)
	}
}

// BenchmarkGenerateCodeVerifier measures verifier generation performance
func BenchmarkGenerateCodeVerifier(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateCodeVerifier()
	}
}

// BenchmarkGenerateCodeChallenge measures challenge generation performance
func BenchmarkGenerateCodeChallenge(b *testing.B) {
	verifier := GenerateCodeVerifier()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateCodeChallenge(verifier)
	}
}

//go:build js && wasm

package crypto

// PBKDF2 configuration constants
const (
	PBKDF2Iterations = 100000 // OWASP 2023 recommendation
	SaltSize         = 16     // 128 bits
)

//go:build js && wasm

package crypto

import (
	"fmt"
	"syscall/js"
)

const (
	IVSize  = 12 // 96 bits for GCM
	TagSize = 16 // 128 bits authentication tag
)

// EncryptedData holds the result of AES-GCM encryption
type EncryptedData struct {
	Ciphertext []byte // Encrypted data (includes auth tag)
	IV         []byte // 12-byte initialization vector
	Salt       []byte // 16-byte salt (for key derivation)
}

// EncryptWithPassphrase encrypts plaintext using a passphrase-derived key
// Returns encrypted data that can be stored and later decrypted
func EncryptWithPassphrase(plaintext []byte, passphrase string) (*EncryptedData, error) {
	// Generate random salt (16 bytes)
	salt := make([]byte, SaltSize)
	GetRandomValues(salt)

	// Derive key from passphrase
	key, err := deriveKeyFromPassphrase(passphrase, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Generate random IV (12 bytes)
	iv := make([]byte, IVSize)
	GetRandomValues(iv)

	// Encrypt
	encrypted, err := encryptWithKey(plaintext, key, iv)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt: %w", err)
	}

	return &EncryptedData{
		Ciphertext: encrypted,
		IV:         iv,
		Salt:       salt,
	}, nil
}

// DecryptWithPassphrase decrypts ciphertext using a passphrase-derived key
func DecryptWithPassphrase(encrypted *EncryptedData, passphrase string) ([]byte, error) {
	// Derive the same key from passphrase + salt
	key, err := deriveKeyFromPassphrase(passphrase, encrypted.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	// Decrypt
	plaintext, err := decryptWithKey(encrypted.Ciphertext, key, encrypted.IV)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt (wrong passphrase?): %w", err)
	}

	return plaintext, nil
}

// deriveKeyFromPassphrase derives an AES-256 key using PBKDF2
func deriveKeyFromPassphrase(passphrase string, salt []byte) (js.Value, error) {
	// Import passphrase as PBKDF2 key material
	baseKeyPromise := ImportKeyPBKDF2([]byte(passphrase))
	baseKey, err := WaitForPromise(baseKeyPromise)
	if err != nil {
		return js.Undefined(), fmt.Errorf("failed to import passphrase: %w", err)
	}

	// Derive AES-GCM key with 100,000 iterations
	keyPromise := DeriveKey(baseKey, salt, PBKDF2Iterations)
	key, err := WaitForPromise(keyPromise)
	if err != nil {
		return js.Undefined(), fmt.Errorf("failed to derive key: %w", err)
	}

	return key, nil
}

// encryptWithKey encrypts plaintext with an existing key
func encryptWithKey(plaintext []byte, key js.Value, iv []byte) ([]byte, error) {
	encryptPromise := EncryptAESGCM(key, iv, plaintext)
	result, err := WaitForPromise(encryptPromise)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Convert ArrayBuffer to bytes (includes auth tag)
	return ArrayBufferToBytes(result), nil
}

// decryptWithKey decrypts ciphertext with an existing key
func decryptWithKey(ciphertext []byte, key js.Value, iv []byte) ([]byte, error) {
	decryptPromise := DecryptAESGCM(key, iv, ciphertext)
	result, err := WaitForPromise(decryptPromise)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return ArrayBufferToBytes(result), nil
}

package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const encPrefix = "enc:"

// Encrypt encrypts plaintext with AES-256-GCM and returns a hex-encoded
// ciphertext prefixed with "enc:". key must be exactly 32 bytes.
func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encPrefix + hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a value produced by Encrypt. If the value does not carry
// the "enc:" prefix it is returned as-is, allowing backward-compatible reads
// of tokens stored before encryption was enabled.
func Decrypt(key []byte, value string) (string, error) {
	if !strings.HasPrefix(value, encPrefix) {
		return value, nil // plaintext stored before encryption was enabled
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(value, encPrefix))
	if err != nil {
		return "", fmt.Errorf("hex decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	plaintext, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", fmt.Errorf("gcm open: %w", err)
	}
	return string(plaintext), nil
}

// ParseKey decodes a 64-character hex string into a 32-byte AES-256 key.
// Returns nil if the input is empty (encryption disabled).
func ParseKey(hexKey string) ([]byte, error) {
	if hexKey == "" {
		return nil, nil
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, fmt.Errorf("PLAID_TOKEN_ENCRYPTION_KEY must be 64 hex chars: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("PLAID_TOKEN_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d", len(key))
	}
	return key, nil
}

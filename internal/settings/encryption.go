package settings

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

var (
	encKey     []byte
	encKeyOnce sync.Once
	encKeyErr  error
)

// loadEncryptionKey loads and validates the 32-byte hex key from the environment.
// Called once on first use via sync.Once.
func loadEncryptionKey() ([]byte, error) {
	encKeyOnce.Do(func() {
		raw := strings.TrimSpace(os.Getenv("SETTINGS_ENCRYPTION_KEY"))
		if raw == "" {
			encKeyErr = errors.New("SETTINGS_ENCRYPTION_KEY env var is not set")
			return
		}
		decoded, err := hex.DecodeString(raw)
		if err != nil {
			encKeyErr = fmt.Errorf("SETTINGS_ENCRYPTION_KEY is not valid hex: %w", err)
			return
		}
		if len(decoded) != 32 {
			encKeyErr = fmt.Errorf("SETTINGS_ENCRYPTION_KEY must be 32 bytes (64 hex chars), got %d bytes", len(decoded))
			return
		}
		encKey = decoded
	})
	return encKey, encKeyErr
}

// Encrypt encrypts plaintext using AES-256-GCM and returns a hex-encoded ciphertext.
// Returns empty string if plaintext is empty (no encryption needed).
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("encrypt: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encrypt: create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encrypt: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return hex.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a hex-encoded AES-256-GCM ciphertext produced by Encrypt.
// Returns empty string if ciphertext is empty.
func Decrypt(cipherHex string) (string, error) {
	if cipherHex == "" {
		return "", nil
	}

	key, err := loadEncryptionKey()
	if err != nil {
		return "", err
	}

	data, err := hex.DecodeString(cipherHex)
	if err != nil {
		return "", fmt.Errorf("decrypt: hex decode: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("decrypt: create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("decrypt: create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("decrypt: ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GenerateEncryptionKey prints a new random 32-byte hex key (utility for .env setup).
func GenerateEncryptionKey() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Package crypto provides application-layer encryption for secrets stored at rest,
// such as per-user AI provider API keys. It uses AES-256-GCM with a random nonce
// per message; ciphertext is returned base64-encoded for storage in a text column.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// ErrEmptyKey is returned by NewCipher when no encryption key is configured.
var ErrEmptyKey = errors.New("crypto: encryption key must not be empty")

// Cipher encrypts and decrypts short secrets with AES-256-GCM.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher derives a 32-byte AES-256 key from the supplied secret (via SHA-256,
// so any non-empty passphrase is accepted) and returns a ready-to-use Cipher.
// An empty secret is rejected so callers can fail closed at startup.
func NewCipher(secret string) (*Cipher, error) {
	if secret == "" {
		return nil, ErrEmptyKey
	}

	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("crypto: new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: new gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

// Encrypt returns base64(nonce || ciphertext || tag) for the given plaintext.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto: read nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. It returns an error if the input is not valid
// base64, is too short to contain a nonce, or fails authentication.
func (c *Cipher) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: decode: %w", err)
	}

	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("crypto: ciphertext too short")
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto: open: %w", err)
	}

	return string(plaintext), nil
}

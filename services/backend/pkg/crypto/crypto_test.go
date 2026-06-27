package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCipher_RejectsEmptyKey(t *testing.T) {
	_, err := NewCipher("")
	assert.ErrorIs(t, err, ErrEmptyKey)
}

func TestCipher_RoundTrip(t *testing.T) {
	c, err := NewCipher("test-encryption-key")
	require.NoError(t, err)

	plaintext := "sk-ant-api03-EXAMPLE-not-a-real-key"

	encrypted, err := c.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, encrypted, "ciphertext must not equal plaintext")

	decrypted, err := c.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCipher_EncryptUsesRandomNonce(t *testing.T) {
	c, err := NewCipher("test-encryption-key")
	require.NoError(t, err)

	first, err := c.Encrypt("same-input")
	require.NoError(t, err)
	second, err := c.Encrypt("same-input")
	require.NoError(t, err)

	assert.NotEqual(t, first, second, "each encryption must use a fresh nonce")
}

func TestCipher_DecryptRejectsTampering(t *testing.T) {
	c, err := NewCipher("test-encryption-key")
	require.NoError(t, err)

	encrypted, err := c.Encrypt("secret")
	require.NoError(t, err)

	// Flip a character in the ciphertext; GCM authentication must reject it.
	tampered := "A" + encrypted[1:]
	_, err = c.Decrypt(tampered)
	assert.Error(t, err)
}

func TestCipher_DecryptRejectsNonCiphertext(t *testing.T) {
	c, err := NewCipher("test-encryption-key")
	require.NoError(t, err)

	// A legacy plaintext value (not produced by Encrypt) must not decrypt cleanly,
	// so callers can detect it and apply a fallback policy.
	_, err = c.Decrypt("plaintext-legacy-api-key")
	assert.Error(t, err)
}

func TestCipher_WrongKeyFails(t *testing.T) {
	a, err := NewCipher("key-a")
	require.NoError(t, err)
	b, err := NewCipher("key-b")
	require.NoError(t, err)

	encrypted, err := a.Encrypt("secret")
	require.NoError(t, err)

	_, err = b.Decrypt(encrypted)
	assert.Error(t, err)
}

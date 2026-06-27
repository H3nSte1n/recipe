package service

import "go.uber.org/zap"

// APIKeyCipher encrypts and decrypts user AI API keys at the repository boundary.
// Implemented by pkg/crypto.Cipher.
type APIKeyCipher interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

// decryptAPIKey decrypts a stored API key. To support lazy migration of rows that
// predate at-rest encryption, a decrypt failure is treated as a legacy plaintext
// value: it is returned unchanged and a warning is logged. Such rows are
// re-encrypted automatically the next time the config is updated.
func decryptAPIKey(cipher APIKeyCipher, logger *zap.Logger, stored string) string {
	if stored == "" {
		return ""
	}
	plaintext, err := cipher.Decrypt(stored)
	if err != nil {
		logger.Warn("AI API key decrypt failed; treating stored value as legacy plaintext", zap.Error(err))
		return stored
	}
	return plaintext
}

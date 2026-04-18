package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

type secretManager struct {
	legacyKey []byte
}

func newSecretManager(keyPath string) (*secretManager, error) {
	key, err := loadExistingKey(keyPath)
	if err != nil {
		return nil, err
	}

	return &secretManager{legacyKey: key}, nil
}

func loadExistingKey(keyPath string) ([]byte, error) {
	if keyPath == "" {
		return nil, nil
	}

	existing, err := os.ReadFile(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read legacy encryption key: %w", err)
	}

	decoded, decodeErr := base64.StdEncoding.DecodeString(strings.TrimSpace(string(existing)))
	if decodeErr != nil || len(decoded) != 32 {
		return nil, nil
	}
	return decoded, nil
}

func (m *secretManager) EncryptString(plainText string) (string, error) {
	return plainText, nil
}

func (m *secretManager) DecryptString(value string) (string, error) {
	normalized, _, err := m.NormalizeStoredSecret(value)
	return normalized, err
}

func (m *secretManager) NormalizeStoredSecret(value string) (string, bool, error) {
	if value == "" {
		return "", false, nil
	}
	if len(m.legacyKey) != 32 {
		return value, false, nil
	}

	plainText, ok, err := decryptLegacyString(m.legacyKey, value)
	if err != nil {
		return "", false, err
	}
	if ok {
		return plainText, true, nil
	}
	return value, false, nil
}

func decryptLegacyString(key []byte, cipherText string) (string, bool, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", false, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", false, fmt.Errorf("create legacy aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", false, fmt.Errorf("create legacy aes-gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", false, nil
	}

	nonce, cipherPayload := raw[:nonceSize], raw[nonceSize:]
	plain, err := gcm.Open(nil, nonce, cipherPayload, nil)
	if err != nil {
		return "", false, nil
	}

	return string(plain), true, nil
}

func maskAPIKey(hasAPIKey bool) string {
	if !hasAPIKey {
		return ""
	}
	return "已配置"
}

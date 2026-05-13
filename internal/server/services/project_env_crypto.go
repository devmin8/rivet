package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

func (s *ProjectEnvService) encrypt(value string) (string, error) {
	if len(s.secretKey) == 0 {
		return "", ErrMissingSecretKey
	}

	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *ProjectEnvService) decrypt(value string) (string, error) {
	if len(s.secretKey) == 0 {
		return "", ErrMissingSecretKey
	}

	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	block, err := aes.NewCipher(s.secretKey)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	if len(data) < gcm.NonceSize() {
		return "", ErrInvalidSecretKey
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidSecretKey
	}

	return string(plaintext), nil
}

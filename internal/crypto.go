package internal

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
)

func Encrypt(plainText []byte, key []byte) ([]byte, error) {
	// Hash the key to ensure it is exactly 32 bytes (AES-256)
	// This allows users to use any length secret (passphrase or hex key)
	key32 := sha256.Sum256(key)

	block, err := aes.NewCipher(key32[:])
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return aesGCM.Seal(nonce, nonce, plainText, nil), nil
}

func Decrypt(cipherText []byte, key []byte) ([]byte, error) {
	// Hash the key to ensure it is exactly 32 bytes
	key32 := sha256.Sum256(key)

	block, err := aes.NewCipher(key32[:])
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherText) < nonceSize {
		return nil, errors.New("cipher too short")
	}

	nonce := cipherText[:nonceSize]
	cipherData := cipherText[nonceSize:]

	return aesGCM.Open(nil, nonce, cipherData, nil)
}

package internal

import (
	"bytes"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	keyStr := "1234567890ABCDEF1234567890ABCDEF" // 32 bytes
	key := []byte(keyStr)
	plainText := []byte("secret message")

	// 1. Test successful encryption and decryption
	cipherText, err := Encrypt(plainText, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(cipherText) == 0 {
		t.Fatal("CipherText is empty")
	}

	decrypted, err := Decrypt(cipherText, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(decrypted, plainText) {
		t.Errorf("Decrypted message does not match original.\nGot: %s\nWant: %s", decrypted, plainText)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte("1234567890ABCDEF1234567890ABCDEF")
	key2 := []byte("TOTAL_DIFFERENT_KEY_1234567890AB") // 32 bytes
	plainText := []byte("secret message")

	cipherText, _ := Encrypt(plainText, key1)

	// Decrypt with wrong key should fail
	_, err := Decrypt(cipherText, key2)
	if err == nil {
		t.Error("Expected error when decrypting with wrong key, got nil")
	}
}

func TestNonceRandomness(t *testing.T) {
	key := []byte("1234567890ABCDEF1234567890ABCDEF")
	plainText := []byte("same message")

	// Encrypt same message twice
	c1, _ := Encrypt(plainText, key)
	c2, _ := Encrypt(plainText, key)

	// Resulting ciphertext should be different due to random nonce
	if bytes.Equal(c1, c2) {
		t.Error("Encryption should produce different output for same input (nonce usage)")
	}
}

func TestCorruptCiphertext(t *testing.T) {
	key := []byte("1234567890ABCDEF1234567890ABCDEF")

	// Too short to contain nonce
	shortData := []byte("foo")
	_, err := Decrypt(shortData, key)
	if err == nil {
		t.Error("Expected error for short ciphertext, got nil")
	} else if err.Error() != "cipher too short" {
		t.Errorf("Expected 'cipher too short' error, got: %v", err)
	}

	// Tampered data
	valid, _ := Encrypt([]byte("message"), key)
	valid[len(valid)-1] ^= 0x01 // Flip last bit

	_, err = Decrypt(valid, key)
	if err == nil {
		t.Error("Expected error for tampered ciphertext, got nil")
	}
}

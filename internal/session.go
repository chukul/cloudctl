package internal

import (
	"time"
)

// SessionInfo represents a stored AWS session
type SessionInfo struct {
	Profile     string    `json:"profile"`
	RoleARN     string    `json:"role_arn"`
	Expiration  time.Time `json:"expiration"`
	Revoked     bool      `json:"revoked"`
	EncryptedAK []byte    `json:"encrypted_ak"`
	EncryptedSK []byte    `json:"encrypted_sk"`
	EncryptedST []byte    `json:"encrypted_st"`
}

// Encrypt/decrypt functions
// func Encrypt(key []byte, value []byte) ([]byte, error) {...}
// func Decrypt(key []byte, encrypted []byte) (string, error) {...}

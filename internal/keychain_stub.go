//go:build !darwin

package internal

import (
	"fmt"
	"os"
)

// GetSecret stub for non-macOS
func GetSecret(explicitSecret string) (string, error) {
	if explicitSecret != "" {
		return explicitSecret, nil
	}
	envSecret := os.Getenv("CLOUDCTL_SECRET")
	if envSecret != "" {
		return envSecret, nil
	}
	return "", fmt.Errorf("no secret found and keychain is only supported on macOS")
}

// SetupKeychain stub for non-macOS
func SetupKeychain() (string, error) {
	return "", fmt.Errorf("keychain integration is only supported on macOS")
}

// StoreKeychainSecret stub for non-macOS
func StoreKeychainSecret(secret string) error {
	return fmt.Errorf("keychain integration is only supported on macOS")
}

func getKeychainSecret() (string, error) {
	return "", fmt.Errorf("keychain integration is only supported on macOS")
}

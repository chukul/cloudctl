package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	"github.com/keybase/go-keychain"
)

const (
	KeychainService = "cloudctl"
	KeychainAccount = "master-key"
)

// GetSecret retrieves a secret from one of three sources (in priority order):
// 1. Explicit flag/argument (passed in)
// 2. Environment variable (CLOUDCTL_SECRET)
// 3. System Keychain (macOS only)
func GetSecret(explicitSecret string) (string, error) {
	// 1. Explicit flag
	if explicitSecret != "" {
		return explicitSecret, nil
	}

	// 2. Environment variable
	envSecret := os.Getenv("CLOUDCTL_SECRET")
	if envSecret != "" {
		return envSecret, nil
	}

	// 3. System Keychain (macOS only)
	if runtime.GOOS == "darwin" {
		secret, err := getKeychainSecret()
		if err == nil && secret != "" {
			return secret, nil
		}
	}

	return "", fmt.Errorf("no secret found")
}

// SetupKeychain attempts to generate and store a new secret in the keychain
func SetupKeychain() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("keychain integration is only supported on macOS")
	}

	// Generate a random 32-byte key
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(key) // 64 chars hex string

	// Store in keychain
	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(KeychainService)
	item.SetAccount(KeychainAccount)
	item.SetLabel("CloudCtl Encryption Key")
	item.SetAccessGroup(KeychainService)
	item.SetData([]byte(secret))
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	// Remove existing if any
	keychain.DeleteItem(item)

	// Add new
	if err := keychain.AddItem(item); err != nil {
		return "", fmt.Errorf("failed to save to keychain: %w", err)
	}

	return secret, nil
}

func getKeychainSecret() (string, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(KeychainService)
	query.SetAccount(KeychainAccount)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return "", err
	} else if len(results) != 1 {
		return "", fmt.Errorf("secret not found in keychain")
	}

	return string(results[0].Data), nil
}

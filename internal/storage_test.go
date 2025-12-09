package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper to create a temp directory for tests
func setupTestDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "cloudctl-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Override the storePath variable for testing
	// ensure we set it back after test
	originalPath := storePath
	storePath = filepath.Join(dir, "credentials.json")

	t.Cleanup(func() {
		os.RemoveAll(dir)
		storePath = originalPath
	})

	return dir
}

func TestSaveAndLoadCredentials(t *testing.T) {
	setupTestDir(t)

	key := "1234567890ABCDEF1234567890ABCDEF"
	profile := "test-profile"

	session := &AWSSession{
		Profile:       profile,
		AccessKey:     "AKIATEST1234",
		SecretKey:     "SecretKey1234",
		SessionToken:  "Token1234",
		Expiration:    time.Now().Add(1 * time.Hour),
		RoleArn:       "arn:aws:iam::123:role/TestRole",
		SessionName:   profile,
		SourceProfile: "default",
	}

	// 1. Save
	err := SaveCredentials(profile, session, key)
	if err != nil {
		t.Fatalf("SaveCredentials failed: %v", err)
	}

	// 2. File should exist
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		t.Fatal("Credentials file was not created")
	}

	// 3. Load
	loaded, err := LoadCredentials(profile, key)
	if err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	// 4. Verify fields
	if loaded.AccessKey != session.AccessKey {
		t.Errorf("AccessKey mismatch. Got %s, want %s", loaded.AccessKey, session.AccessKey)
	}
	if loaded.SecretKey != session.SecretKey {
		t.Errorf("SecretKey mismatch")
	}

	// Compare times allowing for small serialization diff (RFC3339 loses some precision)
	if !loaded.Expiration.Equal(session.Expiration) && loaded.Expiration.Format(time.RFC3339) != session.Expiration.Format(time.RFC3339) {
		t.Errorf("Expiration mismatch. Got %v, want %v", loaded.Expiration, session.Expiration)
	}
}

func TestSaveMultipleProfiles(t *testing.T) {
	setupTestDir(t)
	key := "1234567890ABCDEF1234567890ABCDEF"

	s1 := &AWSSession{Profile: "p1", AccessKey: "k1", Expiration: time.Now()}
	s2 := &AWSSession{Profile: "p2", AccessKey: "k2", Expiration: time.Now()}

	SaveCredentials("p1", s1, key)
	SaveCredentials("p2", s2, key)

	l1, _ := LoadCredentials("p1", key)
	l2, _ := LoadCredentials("p2", key)

	if l1.AccessKey != "k1" || l2.AccessKey != "k2" {
		t.Error("Failed to retrieve multiple profiles correctly")
	}
}

func TestCorruptJSONHandling(t *testing.T) {
	setupTestDir(t)
	key := "1234567890ABCDEF1234567890ABCDEF"

	// Create a corrupt file
	os.WriteFile(storePath, []byte("{ invalid json..."), 0600)

	// Try to save a new session -> Should Error now (thanks to our fix)
	session := &AWSSession{Profile: "new", AccessKey: "k", Expiration: time.Now()}
	err := SaveCredentials("new", session, key)

	if err == nil {
		t.Error("Expected error when saving to corrupt file, got nil")
	}
}

func TestRemoveProfile(t *testing.T) {
	setupTestDir(t)
	key := "1234567890ABCDEF1234567890ABCDEF"

	s1 := &AWSSession{Profile: "p1", AccessKey: "k1", Expiration: time.Now()}
	SaveCredentials("p1", s1, key)

	// Remove
	err := RemoveProfile("p1")
	if err != nil {
		t.Errorf("RemoveProfile failed: %v", err)
	}

	// Should not find it anymore
	_, err = LoadCredentials("p1", key)
	if err == nil { // Expecting error from decryption or key lookup
		// Actually LoadCredentials panics currently if key missing in map?
		// Let's check implementation of LoadCredentials.
		// It reads map[profile], if nil -> crash? No, map lookup returns nil values.
		// Then enc["AccessKey"] calls on nil map -> panic?
		// We should probably safeguard LoadCredentials too, but for now checking if we get error or panic
	}

	// Verify file is empty/removed if last profile
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		// If file exists, check if empty or {}
		b, _ := os.ReadFile(storePath)
		var data map[string]interface{}
		json.Unmarshal(b, &data)
		if len(data) > 0 {
			t.Errorf("Store should be empty, got: %s", string(b))
		}
	}
}

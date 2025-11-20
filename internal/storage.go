package internal

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var storePath = filepath.Join(os.Getenv("HOME"), ".cloudctl", "credentials.json")

// SaveCredentials encrypts and stores AWS session
func SaveCredentials(profile string, creds *AWSSession, key string) error {
	os.MkdirAll(filepath.Dir(storePath), 0700)

	// encrypt each field using []byte(key)
	akEnc, _ := Encrypt([]byte(creds.AccessKey), []byte(key))
	skEnc, _ := Encrypt([]byte(creds.SecretKey), []byte(key))
	stEnc, _ := Encrypt([]byte(creds.SessionToken), []byte(key))
	exEnc, _ := Encrypt([]byte(creds.Expiration.Format(time.RFC3339)), []byte(key))
	rnEnc, _ := Encrypt([]byte(creds.RoleArn), []byte(key))
	snEnc, _ := Encrypt([]byte(creds.SessionName), []byte(key))

	// convert encrypted bytes to base64 strings for JSON
	encrypted := map[string]string{
		"AccessKey":    base64.StdEncoding.EncodeToString(akEnc),
		"SecretKey":    base64.StdEncoding.EncodeToString(skEnc),
		"SessionToken": base64.StdEncoding.EncodeToString(stEnc),
		"Expiration":   base64.StdEncoding.EncodeToString(exEnc),
		"RoleArn":      base64.StdEncoding.EncodeToString(rnEnc),
		"SessionName":  base64.StdEncoding.EncodeToString(snEnc),
	}

	// load existing data
	data := make(map[string]map[string]string)
	if _, err := os.Stat(storePath); err == nil {
		b, _ := os.ReadFile(storePath)
		json.Unmarshal(b, &data)
	}

	data[profile] = encrypted

	// save updated JSON
	b, _ := json.MarshalIndent(data, "", "  ")
	return os.WriteFile(storePath, b, 0600)
}

// LoadCredentials decrypts AWS session for a profile
func LoadCredentials(profile, key string) (*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return nil, err
	}

	var data map[string]map[string]string
	json.Unmarshal(b, &data)
	enc := data[profile]

	decryptField := func(field string) string {
		bytes, _ := base64.StdEncoding.DecodeString(enc[field])
		decrypted, _ := Decrypt(bytes, []byte(key))
		return string(decrypted)
	}

	expStr := decryptField("Expiration")
	exp, _ := time.Parse(time.RFC3339, expStr)

	return &AWSSession{
		Profile:      profile,
		AccessKey:    decryptField("AccessKey"),
		SecretKey:    decryptField("SecretKey"),
		SessionToken: decryptField("SessionToken"),
		Expiration:   exp,
		RoleArn:      decryptField("RoleArn"),
		SessionName:  decryptField("SessionName"),
	}, nil
}

// RemoveProfile deletes a stored profile
func RemoveProfile(profile string) error {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return fmt.Errorf("failed to read store: %w", err)
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(b, &data); err != nil {
		return fmt.Errorf("failed to parse credentials: %w", err)
	}

	if _, ok := data[profile]; !ok {
		return fmt.Errorf("profile '%s' not found", profile)
	}

	delete(data, profile)

	if len(data) == 0 {
		return os.Remove(storePath)
	}

	out, _ := json.MarshalIndent(data, "", "  ")
	return os.WriteFile(storePath, out, 0600)
}

// ClearAllCredentials removes all stored sessions
func ClearAllCredentials() error {
	return os.Remove(storePath)
}

// ListAllSessions returns all stored AWS sessions
func ListAllSessions(secret string) ([]*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return []*AWSSession{}, nil
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	sessions := []*AWSSession{}
	for profile, enc := range data {
		decryptField := func(field string) string {
			bytes, _ := base64.StdEncoding.DecodeString(enc[field])
			decrypted, _ := Decrypt(bytes, []byte(secret))
			return string(decrypted)
		}

		expStr := decryptField("Expiration")
		exp, _ := time.Parse(time.RFC3339, expStr)

		revoked := false
		if val, ok := enc["Revoked"]; ok && val == "true" {
			revoked = true
		}

		sessions = append(sessions, &AWSSession{
			Profile:      profile,
			AccessKey:    decryptField("AccessKey"),
			SecretKey:    decryptField("SecretKey"),
			SessionToken: decryptField("SessionToken"),
			Expiration:   exp,
			RoleArn:      decryptField("RoleArn"),
			SessionName:  decryptField("SessionName"),
			Revoked:      revoked,
		})
	}

	return sessions, nil
}

func ListProfiles() ([]string, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return []string{}, nil
	}
	var data map[string]interface{}
	json.Unmarshal(b, &data)
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys, nil
}

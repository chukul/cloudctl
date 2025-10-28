package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var storePath = filepath.Join(os.Getenv("HOME"), ".cloudctl", "credentials.json")

type storedProfile struct {
	AccessKey    string `json:"AccessKey"`
	SecretKey    string `json:"SecretKey"`
	SessionToken string `json:"SessionToken"`
	Expiration   string `json:"Expiration"`
	RoleArn      string `json:"RoleArn"`
	SessionName  string `json:"SessionName"`
}

func SaveCredentials(profile string, creds *AWSSession, key string) error {
	os.MkdirAll(filepath.Dir(storePath), 0700)

	encrypted := make(map[string]string)
	ak, _ := Encrypt(key, creds.AccessKey)
	sk, _ := Encrypt(key, creds.SecretKey)
	st, _ := Encrypt(key, creds.SessionToken)
	ex, _ := Encrypt(key, creds.Expiration.Format(time.RFC3339))
	rn, _ := Encrypt(key, creds.RoleArn)
	sn, _ := Encrypt(key, creds.SessionName)

	encrypted["AccessKey"] = ak
	encrypted["SecretKey"] = sk
	encrypted["SessionToken"] = st
	encrypted["Expiration"] = ex
	encrypted["RoleArn"] = rn
	encrypted["SessionName"] = sn

	data := make(map[string]map[string]string)
	if _, err := os.Stat(storePath); err == nil {
		b, _ := os.ReadFile(storePath)
		json.Unmarshal(b, &data)
	}

	data[profile] = encrypted
	b, _ := json.MarshalIndent(data, "", "  ")
	return os.WriteFile(storePath, b, 0600)
}

func LoadCredentials(profile, key string) (*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return nil, err
	}
	var data map[string]map[string]string
	json.Unmarshal(b, &data)
	enc := data[profile]

	ak, _ := Decrypt(key, enc["AccessKey"])
	sk, _ := Decrypt(key, enc["SecretKey"])
	st, _ := Decrypt(key, enc["SessionToken"])
	ex, _ := Decrypt(key, enc["Expiration"])
	rn, _ := Decrypt(key, enc["RoleArn"])
	sn, _ := Decrypt(key, enc["SessionName"])

	exp, _ := time.Parse(time.RFC3339, ex)
	return &AWSSession{
		AccessKey:    ak,
		SecretKey:    sk,
		SessionToken: st,
		Expiration:   exp,
		RoleArn:      rn,
		SessionName:  sn,
		Profile:      profile,
	}, nil
}

func ListProfiles() ([]string, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return []string{}, nil
	}
	var data map[string]interface{}
	json.Unmarshal(b, &data)
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	return keys, nil
}

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
		os.Remove(storePath)
		return nil
	}

	out, _ := json.MarshalIndent(data, "", "  ")
	return os.WriteFile(storePath, out, 0600)
}

func ClearAllCredentials() error {
	return os.Remove(storePath)
}

func ListAllSessions(secret string) ([]*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return []*AWSSession{}, nil // empty if file missing
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	sessions := []*AWSSession{}
	for profile, enc := range data {
		ak, _ := Decrypt(secret, enc["AccessKey"])
		sk, _ := Decrypt(secret, enc["SecretKey"])
		st, _ := Decrypt(secret, enc["SessionToken"])
		ex, _ := Decrypt(secret, enc["Expiration"])
		rn, _ := Decrypt(secret, enc["RoleArn"])
		sn, _ := Decrypt(secret, enc["SessionName"])

		exp, _ := time.Parse(time.RFC3339, ex)

		revoked := false
		if val, ok := enc["Revoked"]; ok && val == "true" {
			revoked = true
		}

		sessions = append(sessions, &AWSSession{
			Profile:      profile,
			AccessKey:    ak,
			SecretKey:    sk,
			SessionToken: st,
			Expiration:   exp,
			RoleArn:      rn,
			SessionName:  sn,
			Revoked:      revoked,
		})
	}

	return sessions, nil
}

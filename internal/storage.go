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
var mfaStorePath = filepath.Join(os.Getenv("HOME"), ".cloudctl", "mfa.json")
var roleStorePath = filepath.Join(os.Getenv("HOME"), ".cloudctl", "roles.json")

// SaveCredentials encrypts and stores AWS session
func SaveCredentials(profile string, creds *AWSSession, key string) error {
	os.MkdirAll(filepath.Dir(storePath), 0700)

	// encrypt each field using []byte(key)
	// encrypt each field
	akEnc, err := Encrypt([]byte(creds.AccessKey), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt AccessKey: %w", err)
	}
	skEnc, err := Encrypt([]byte(creds.SecretKey), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt SecretKey: %w", err)
	}
	stEnc, err := Encrypt([]byte(creds.SessionToken), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt SessionToken: %w", err)
	}
	exEnc, err := Encrypt([]byte(creds.Expiration.Format(time.RFC3339)), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt Expiration: %w", err)
	}
	rnEnc, err := Encrypt([]byte(creds.RoleArn), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt RoleArn: %w", err)
	}
	snEnc, err := Encrypt([]byte(creds.SessionName), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt SessionName: %w", err)
	}
	spEnc, err := Encrypt([]byte(creds.SourceProfile), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt SourceProfile: %w", err)
	}
	regEnc, err := Encrypt([]byte(creds.Region), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt Region: %w", err)
	}
	mfaEnc, err := Encrypt([]byte(creds.MfaArn), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt MfaArn: %w", err)
	}
	durEnc, err := Encrypt([]byte(fmt.Sprintf("%d", creds.Duration)), []byte(key))
	if err != nil {
		return fmt.Errorf("failed to encrypt Duration: %w", err)
	}

	// convert encrypted bytes to base64 strings for JSON
	encrypted := map[string]string{
		"AccessKey":     base64.StdEncoding.EncodeToString(akEnc),
		"SecretKey":     base64.StdEncoding.EncodeToString(skEnc),
		"SessionToken":  base64.StdEncoding.EncodeToString(stEnc),
		"Expiration":    base64.StdEncoding.EncodeToString(exEnc),
		"RoleArn":       base64.StdEncoding.EncodeToString(rnEnc),
		"SessionName":   base64.StdEncoding.EncodeToString(snEnc),
		"SourceProfile": base64.StdEncoding.EncodeToString(spEnc),
		"Region":        base64.StdEncoding.EncodeToString(regEnc),
		"MfaArn":        base64.StdEncoding.EncodeToString(mfaEnc),
		"Duration":      base64.StdEncoding.EncodeToString(durEnc),
	}

	// load existing data
	data := make(map[string]map[string]string)
	if _, err := os.Stat(storePath); err == nil {
		b, err := os.ReadFile(storePath)
		if err != nil {
			return fmt.Errorf("failed to read store: %w", err)
		}
		if len(b) > 0 {
			if err := json.Unmarshal(b, &data); err != nil {
				// Don't overwrite if we can't parse!
				return fmt.Errorf("failed to parse existing credentials file (corrupt?): %w", err)
			}
		}
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
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}

	enc, ok := data[profile]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found in store", profile)
	}

	decryptField := func(field string) string {
		bytes, _ := base64.StdEncoding.DecodeString(enc[field])
		decrypted, _ := Decrypt(bytes, []byte(key))
		return string(decrypted)
	}

	expStr := decryptField("Expiration")
	exp, _ := time.Parse(time.RFC3339, expStr)

	durStr := decryptField("Duration")
	var duration int32
	fmt.Sscanf(durStr, "%d", &duration)

	return &AWSSession{
		Profile:       profile,
		AccessKey:     decryptField("AccessKey"),
		SecretKey:     decryptField("SecretKey"),
		SessionToken:  decryptField("SessionToken"),
		Expiration:    exp,
		RoleArn:       decryptField("RoleArn"),
		SessionName:   decryptField("SessionName"),
		SourceProfile: decryptField("SourceProfile"),
		Region:        decryptField("Region"),
		MfaArn:        decryptField("MfaArn"),
		Duration:      duration,
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

		durStr := decryptField("Duration")
		var duration int32
		fmt.Sscanf(durStr, "%d", &duration)

		sessions = append(sessions, &AWSSession{
			Profile:       profile,
			AccessKey:     decryptField("AccessKey"),
			SecretKey:     decryptField("SecretKey"),
			SessionToken:  decryptField("SessionToken"),
			Expiration:    exp,
			RoleArn:       decryptField("RoleArn"),
			SessionName:   decryptField("SessionName"),
			SourceProfile: decryptField("SourceProfile"),
			Region:        decryptField("Region"),
			MfaArn:        decryptField("MfaArn"),
			Duration:      duration,
			Revoked:       revoked,
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

// MFA Device Storage

func SaveMFADevice(name, arn string) error {
	os.MkdirAll(filepath.Dir(mfaStorePath), 0700)

	devices := make(map[string]string)
	if b, err := os.ReadFile(mfaStorePath); err == nil {
		json.Unmarshal(b, &devices)
	}

	devices[name] = arn

	b, _ := json.MarshalIndent(devices, "", "  ")
	return os.WriteFile(mfaStorePath, b, 0600)
}

func ListMFADevices() (map[string]string, error) {
	devices := make(map[string]string)
	b, err := os.ReadFile(mfaStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return devices, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(b, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func RemoveMFADevice(name string) error {
	devices, err := ListMFADevices()
	if err != nil {
		return err
	}

	if _, ok := devices[name]; !ok {
		return fmt.Errorf("device '%s' not found", name)
	}

	delete(devices, name)

	b, _ := json.MarshalIndent(devices, "", "  ")
	return os.WriteFile(mfaStorePath, b, 0600)
}

func GetMFADevice(name string) (string, bool) {
	devices, _ := ListMFADevices()
	arn, ok := devices[name]
	return arn, ok
}

// IAM Role Storage

func SaveRole(name, arn string) error {
	roles, _ := ListRoles()
	roles[name] = arn
	return SaveAllRoles(roles)
}

func SaveAllRoles(roles map[string]string) error {
	os.MkdirAll(filepath.Dir(roleStorePath), 0700)
	b, _ := json.MarshalIndent(roles, "", "  ")
	return os.WriteFile(roleStorePath, b, 0600)
}

func ListRoles() (map[string]string, error) {
	roles := make(map[string]string)
	b, err := os.ReadFile(roleStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return roles, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(b, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func RemoveRole(name string) error {
	roles, err := ListRoles()
	if err != nil {
		return err
	}

	if _, ok := roles[name]; !ok {
		return fmt.Errorf("role '%s' not found", name)
	}

	delete(roles, name)

	if len(roles) == 0 {
		return ClearAllRoles()
	}

	return SaveAllRoles(roles)
}

func ClearAllRoles() error {
	if _, err := os.Stat(roleStorePath); err == nil {
		return os.Remove(roleStorePath)
	}
	return nil
}

func GetRole(name string) (string, bool) {
	roles, _ := ListRoles()
	arn, ok := roles[name]
	return arn, ok
}

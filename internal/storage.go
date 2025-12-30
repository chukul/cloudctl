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

// SaveCredentials encrypts and stores AWS session for a specific profile.
func SaveCredentials(profile string, creds *AWSSession, key string) error {
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	encryptionMap := map[string]string{
		"AccessKey":     creds.AccessKey,
		"SecretKey":     creds.SecretKey,
		"SessionToken":  creds.SessionToken,
		"Expiration":    creds.Expiration.Format(time.RFC3339),
		"RoleArn":       creds.RoleArn,
		"SessionName":   creds.SessionName,
		"SourceProfile": creds.SourceProfile,
		"Region":        creds.Region,
		"MfaArn":        creds.MfaArn,
		"Duration":      fmt.Sprintf("%d", creds.Duration),
	}

	encrypted := make(map[string]string)
	for field, value := range encryptionMap {
		enc, err := Encrypt([]byte(value), []byte(key))
		if err != nil {
			return fmt.Errorf("failed to encrypt %s: %w", field, err)
		}
		encrypted[field] = base64.StdEncoding.EncodeToString(enc)
	}

	// load existing data
	data := make(map[string]map[string]string)
	if b, err := os.ReadFile(storePath); err == nil && len(b) > 0 {
		if err := json.Unmarshal(b, &data); err != nil {
			return fmt.Errorf("failed to parse existing credentials file: %w", err)
		}
	}

	data[profile] = encrypted

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return os.WriteFile(storePath, b, 0600)
}

// LoadCredentials decrypts AWS session for a profile.
func LoadCredentials(profile, key string) (*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	enc, ok := data[profile]
	if !ok {
		return nil, fmt.Errorf("profile '%s' not found in store", profile)
	}

	return decryptSession(profile, enc, key)
}

// decryptSession is a helper to decrypt the fields of a session map.
func decryptSession(profile string, enc map[string]string, key string) (*AWSSession, error) {
	getField := func(field string) (string, error) {
		val, ok := enc[field]
		if !ok {
			return "", nil // Some fields might be missing in older versions
		}
		bytes, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 for %s: %w", field, err)
		}
		decrypted, err := Decrypt(bytes, []byte(key))
		if err != nil {
			return "", fmt.Errorf("failed to decrypt %s: %w", field, err)
		}
		return string(decrypted), nil
	}

	expStr, err := getField("Expiration")
	if err != nil {
		return nil, err
	}
	exp, _ := time.Parse(time.RFC3339, expStr)

	durStr, err := getField("Duration")
	if err != nil {
		return nil, err
	}
	var duration int32
	fmt.Sscanf(durStr, "%d", &duration)

	accessKey, err := getField("AccessKey")
	if err != nil {
		return nil, err
	}
	secretKey, err := getField("SecretKey")
	if err != nil {
		return nil, err
	}
	sessionToken, err := getField("SessionToken")
	if err != nil {
		return nil, err
	}
	roleArn, err := getField("RoleArn")
	if err != nil {
		return nil, err
	}
	sessionName, err := getField("SessionName")
	if err != nil {
		return nil, err
	}
	sourceProfile, err := getField("SourceProfile")
	if err != nil {
		return nil, err
	}
	region, err := getField("Region")
	if err != nil {
		return nil, err
	}
	mfaArn, err := getField("MfaArn")
	if err != nil {
		return nil, err
	}

	revoked := false
	if val, ok := enc["Revoked"]; ok && val == "true" {
		revoked = true
	}

	return &AWSSession{
		Profile:       profile,
		AccessKey:     accessKey,
		SecretKey:     secretKey,
		SessionToken:  sessionToken,
		Expiration:    exp,
		RoleArn:       roleArn,
		SessionName:   sessionName,
		SourceProfile: sourceProfile,
		Region:        region,
		MfaArn:        mfaArn,
		Duration:      duration,
		Revoked:       revoked,
	}, nil
}

// RemoveProfile deletes a stored profile.
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

	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	return os.WriteFile(storePath, out, 0600)
}

// ClearAllCredentials removes all stored sessions.
func ClearAllCredentials() error {
	if err := os.Remove(storePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credentials file: %w", err)
	}
	return nil
}

// ListAllSessions returns all stored AWS sessions.
func ListAllSessions(key string) ([]*AWSSession, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*AWSSession{}, nil
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var data map[string]map[string]string
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	sessions := make([]*AWSSession, 0, len(data))
	for profile, enc := range data {
		s, err := decryptSession(profile, enc, key)
		if err != nil {
			// If one profile fails (e.g. wrong key for some reason), we might want to log it and continue
			// but for now, we'll stop to be safe.
			return nil, fmt.Errorf("failed to decrypt session '%s': %w", profile, err)
		}
		sessions = append(sessions, s)
	}

	return sessions, nil
}

// ListProfiles returns just the names of stored profiles.
func ListProfiles() ([]string, error) {
	b, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read store: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	return keys, nil
}

// SaveMFADevice persists an MFA device ARN with an alias.
func SaveMFADevice(name, arn string) error {
	if err := os.MkdirAll(filepath.Dir(mfaStorePath), 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	devices, err := ListMFADevices()
	if err != nil {
		devices = make(map[string]string)
	}

	devices[name] = arn

	b, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MFA devices: %w", err)
	}
	return os.WriteFile(mfaStorePath, b, 0600)
}

// ListMFADevices returns all stored MFA device aliases.
func ListMFADevices() (map[string]string, error) {
	devices := make(map[string]string)
	b, err := os.ReadFile(mfaStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return devices, nil
		}
		return nil, fmt.Errorf("failed to read MFA store: %w", err)
	}

	if err := json.Unmarshal(b, &devices); err != nil {
		return nil, fmt.Errorf("failed to parse MFA store: %w", err)
	}
	return devices, nil
}

// RemoveMFADevice deletes an MFA device alias.
func RemoveMFADevice(name string) error {
	devices, err := ListMFADevices()
	if err != nil {
		return err
	}

	if _, ok := devices[name]; !ok {
		return fmt.Errorf("device '%s' not found", name)
	}

	delete(devices, name)

	b, err := json.MarshalIndent(devices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MFA devices: %w", err)
	}
	return os.WriteFile(mfaStorePath, b, 0600)
}

// GetMFADevice retrieves an MFA ARN by its alias.
func GetMFADevice(name string) (string, bool) {
	devices, _ := ListMFADevices()
	arn, ok := devices[name]
	return arn, ok
}

// SaveRole persists an IAM Role ARN with an alias.
func SaveRole(name, arn string) error {
	roles, _ := ListRoles()
	roles[name] = arn
	return SaveAllRoles(roles)
}

// SaveAllRoles overwrites the entire role alias store.
func SaveAllRoles(roles map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(roleStorePath), 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}
	b, err := json.MarshalIndent(roles, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal roles: %w", err)
	}
	return os.WriteFile(roleStorePath, b, 0600)
}

// ListRoles returns all stored IAM role aliases.
func ListRoles() (map[string]string, error) {
	roles := make(map[string]string)
	b, err := os.ReadFile(roleStorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return roles, nil
		}
		return nil, fmt.Errorf("failed to read roles store: %w", err)
	}

	if err := json.Unmarshal(b, &roles); err != nil {
		return nil, fmt.Errorf("failed to parse roles store: %w", err)
	}
	return roles, nil
}

// RemoveRole deletes an IAM role alias.
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

// ClearAllRoles removes the entire role alias file.
func ClearAllRoles() error {
	if err := os.Remove(roleStorePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear roles: %w", err)
	}
	return nil
}

// GetRole retrieves an IAM Role ARN by its alias.
func GetRole(name string) (string, bool) {
	roles, _ := ListRoles()
	arn, ok := roles[name]
	return arn, ok
}

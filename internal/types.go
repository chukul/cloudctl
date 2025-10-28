package internal

import "time"

type AWSSession struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Expiration   time.Time

	// New fields for auto-refresh
	Profile     string
	RoleArn     string
	SessionName string
	Revoked     bool // optional flag
}

package internal

import "time"

type AWSSession struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Expiration   time.Time

	// New fields for auto-refresh
	Profile       string
	RoleArn       string
	SessionName   string
	SourceProfile string // Source profile used for assuming the role
	Region        string // AWS Region for this session
	MfaArn        string // MFA Device ARN if used
	Duration      int32  // Requested session duration
	Revoked       bool   // optional flag
}

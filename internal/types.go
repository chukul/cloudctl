package internal

import "time"

// AWSSession represents a stored AWS session with its credentials and metadata.
type AWSSession struct {
	AccessKey    string
	SecretKey    string
	SessionToken string
	Expiration   time.Time

	// Profile is the local alias for this session.
	Profile string
	// RoleArn is the ARN of the IAM role assumed for this session.
	RoleArn string
	// SessionName is the name used during the assume role operation.
	SessionName string
	// SourceProfile is the credentials source used (either another cloudctl profile or AWS CLI profile).
	SourceProfile string
	// Region is the AWS region for this session.
	Region string
	// MfaArn is the MFA device ARN used for authentication.
	MfaArn string
	// Duration is the requested session validity in seconds.
	Duration int32
	// Revoked indicates if the session has been manually invalidated.
	Revoked bool
}

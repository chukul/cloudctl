package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// AssumeRole performs an AWS STS AssumeRole operation and returns a session.
func AssumeRole(profile, roleArn, sessionName, region string) (*AWSSession, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return nil, err
	}

	svc := sts.NewFromConfig(cfg)
	out, err := svc.AssumeRole(context.TODO(), &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &sessionName,
	})
	if err != nil {
		return nil, err
	}

	return &AWSSession{
		AccessKey:    *out.Credentials.AccessKeyId,
		SecretKey:    *out.Credentials.SecretAccessKey,
		SessionToken: *out.Credentials.SessionToken,
		Expiration:   *out.Credentials.Expiration,
		RoleArn:      roleArn,
		SessionName:  sessionName,
	}, nil
}

// PerformRefresh silenty refreshes a single session if possible
func PerformRefresh(s *AWSSession, secret, region string) (*AWSSession, error) {
	if s.RoleArn == "MFA-Session" {
		return nil, fmt.Errorf("MFA sessions cannot be silently refreshed")
	}
	if s.SourceProfile == "" {
		return nil, fmt.Errorf("no source profile stored for this session")
	}

	ctx := context.TODO()
	var cfg aws.Config
	var err error

	// Load source credentials
	sourceSession, sourceErr := LoadCredentials(s.SourceProfile, secret)
	if sourceErr == nil {
		// Source is a cloudctl session - Check if it's still active
		if time.Now().After(sourceSession.Expiration) {
			return nil, fmt.Errorf("source session '%s' has expired", s.SourceProfile)
		}

		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				sourceSession.AccessKey,
				sourceSession.SecretKey,
				sourceSession.SessionToken,
			)),
		)
	} else {
		// Source is standard AWS profile
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithSharedConfigProfile(s.SourceProfile),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load source: %w", err)
	}

	stsClient := sts.NewFromConfig(cfg)
	sessionName := s.Profile
	duration := int32(3600)

	res, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         &s.RoleArn,
		RoleSessionName: &sessionName,
		DurationSeconds: &duration,
	})
	if err != nil {
		return nil, err
	}

	newSession := &AWSSession{
		Profile:       s.Profile,
		AccessKey:     *res.Credentials.AccessKeyId,
		SecretKey:     *res.Credentials.SecretAccessKey,
		SessionToken:  *res.Credentials.SessionToken,
		Expiration:    *res.Credentials.Expiration,
		RoleArn:       s.RoleArn,
		SourceProfile: s.SourceProfile,
		Region:        s.Region,
		MfaArn:        s.MfaArn,
		Duration:      s.Duration,
	}

	if err := SaveCredentials(s.Profile, newSession, secret); err != nil {
		return nil, err
	}

	return newSession, nil
}

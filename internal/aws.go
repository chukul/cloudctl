package internal

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

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
	if s.SourceProfile == "" || s.RoleArn == "MFA-Session" {
		return nil, fmt.Errorf("session cannot be refreshed (no source or MFA session)")
	}

	ctx := context.TODO()
	var cfg aws.Config
	var err error

	// Load source credentials
	sourceSession, sourceErr := LoadCredentials(s.SourceProfile, secret)
	if sourceErr == nil {
		// Source is a cloudctl session
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
	}

	if err := SaveCredentials(s.Profile, newSession, secret); err != nil {
		return nil, err
	}

	return newSession, nil
}

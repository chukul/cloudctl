package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func AssumeRole(profile, roleArn, sessionName string) (*AWSSession, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile))
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

// New function
func RefreshIfExpired(sess *AWSSession, secret string) (*AWSSession, bool, error) {
	now := time.Now()
	if sess.Expiration.After(now.Add(2 * time.Minute)) {
		return sess, false, nil // still valid
	}

	fmt.Println("🔁 Session expired — refreshing credentials...")

	newSess, err := AssumeRole(sess.Profile, sess.RoleArn, sess.SessionName)
	if err != nil {
		return sess, false, err
	}

	err = SaveCredentials(sess.Profile, newSess, secret)
	if err != nil {
		return sess, false, err
	}

	return newSess, true, nil
}

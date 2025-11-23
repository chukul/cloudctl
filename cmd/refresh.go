package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var refreshSecret string
var refreshAll bool
var refreshProfile string

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh AWS session credentials before expiration",
	Long:  `Renew session credentials by re-assuming the role. Can refresh a single profile or all active sessions.`,
	Run: func(cmd *cobra.Command, args []string) {
		if refreshSecret == "" {
			fmt.Println("❌ You must specify --secret to decrypt credentials")
			return
		}

		if refreshAll {
			refreshAllSessions()
		} else if refreshProfile != "" {
			refreshSingleSession(refreshProfile)
		} else {
			fmt.Println("❌ You must specify either --profile or --all")
		}
	},
}

func refreshSingleSession(profile string) {
	fmt.Printf("🔄 Refreshing session for profile '%s'...\n", profile)

	// Load existing session
	session, err := internal.LoadCredentials(profile, refreshSecret)
	if err != nil {
		fmt.Printf("❌ Failed to load session: %v\n", err)
		return
	}

	// Check if already expired
	if time.Until(session.Expiration) <= 0 {
		fmt.Printf("⚠️  Session expired. Please use 'cloudctl login' to create a new session.\n")
		return
	}

	// Use current session credentials to assume role again
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			session.AccessKey,
			session.SecretKey,
			session.SessionToken,
		)),
	)
	if err != nil {
		fmt.Printf("❌ Failed to configure AWS SDK: %v\n", err)
		return
	}

	// Assume role again
	stsClient := sts.NewFromConfig(cfg)
	sessionName := fmt.Sprintf("cloudctl-%d", time.Now().Unix())
	duration := int32(3600)

	roleResult, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
		RoleArn:         &session.RoleArn,
		RoleSessionName: &sessionName,
		DurationSeconds: &duration,
	})
	if err != nil {
		fmt.Printf("❌ Failed to refresh session: %v\n", err)
		return
	}

	// Update session with new credentials
	expiration := *roleResult.Credentials.Expiration
	newSession := &internal.AWSSession{
		Profile:      profile,
		AccessKey:    *roleResult.Credentials.AccessKeyId,
		SecretKey:    *roleResult.Credentials.SecretAccessKey,
		SessionToken: *roleResult.Credentials.SessionToken,
		Expiration:   expiration,
		RoleArn:      session.RoleArn,
	}

	// Save refreshed session
	if err := internal.SaveCredentials(profile, newSession, refreshSecret); err != nil {
		fmt.Printf("❌ Failed to save refreshed session: %v\n", err)
		return
	}

	remaining := time.Until(expiration).Round(time.Minute)
	fmt.Printf("✅ Session refreshed successfully\n")
	fmt.Printf("   Profile: %s\n", profile)
	fmt.Printf("   Role: %s\n", session.RoleArn)
	fmt.Printf("   Expires: %s (%v remaining)\n", expiration.Format("2006-01-02 15:04:05"), remaining)
}

func refreshAllSessions() {
	fmt.Println("🔄 Refreshing all active sessions...")

	sessions, err := internal.ListAllSessions(refreshSecret)
	if err != nil {
		fmt.Printf("❌ Failed to load sessions: %v\n", err)
		return
	}

	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return
	}

	now := time.Now()
	refreshed := 0
	skipped := 0
	failed := 0

	for _, s := range sessions {
		remaining := s.Expiration.Sub(now)
		
		// Skip expired sessions
		if remaining <= 0 {
			fmt.Printf("⏭️  Skipping '%s' (expired)\n", s.Profile)
			skipped++
			continue
		}

		// Refresh the session
		fmt.Printf("\n🔄 Refreshing '%s'...\n", s.Profile)
		
		ctx := context.TODO()
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				s.AccessKey,
				s.SecretKey,
				s.SessionToken,
			)),
		)
		if err != nil {
			fmt.Printf("❌ Failed to configure AWS SDK: %v\n", err)
			failed++
			continue
		}

		stsClient := sts.NewFromConfig(cfg)
		sessionName := fmt.Sprintf("cloudctl-%d", time.Now().Unix())
		duration := int32(3600)

		roleResult, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         &s.RoleArn,
			RoleSessionName: &sessionName,
			DurationSeconds: &duration,
		})
		if err != nil {
			fmt.Printf("❌ Failed to refresh: %v\n", err)
			failed++
			continue
		}

		expiration := *roleResult.Credentials.Expiration
		newSession := &internal.AWSSession{
			Profile:      s.Profile,
			AccessKey:    *roleResult.Credentials.AccessKeyId,
			SecretKey:    *roleResult.Credentials.SecretAccessKey,
			SessionToken: *roleResult.Credentials.SessionToken,
			Expiration:   expiration,
			RoleArn:      s.RoleArn,
		}

		if err := internal.SaveCredentials(s.Profile, newSession, refreshSecret); err != nil {
			fmt.Printf("❌ Failed to save: %v\n", err)
			failed++
			continue
		}

		fmt.Printf("✅ Refreshed successfully (expires: %s)\n", expiration.Format("2006-01-02 15:04:05"))
		refreshed++
	}

	fmt.Printf("\n📊 Summary: %d refreshed, %d skipped, %d failed\n", refreshed, skipped, failed)
}

func init() {
	refreshCmd.Flags().StringVar(&refreshSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	refreshCmd.Flags().BoolVar(&refreshAll, "all", false, "Refresh all active sessions")
	refreshCmd.Flags().StringVar(&refreshProfile, "profile", "", "Profile to refresh")
	rootCmd.AddCommand(refreshCmd)
}

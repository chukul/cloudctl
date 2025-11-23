package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var (
	sourceProfile string // Base AWS CLI profile for assume role
	profile       string // The name for storing the assumed session
	roleArn       string
	mfaArn        string
	secretKey     string
	region        string
	sessionDir    = filepath.Join(os.Getenv("HOME"), ".cloudctl", "sessions")
)

// loginCmd implements `cloudctl login`
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Assume an AWS role and store credentials locally (supports MFA)",
	Run: func(cmd *cobra.Command, args []string) {
		if sourceProfile == "" || profile == "" || roleArn == "" {
			log.Fatal("❌ You must specify --source, --profile, and --role.")
		}

		// Create session directory if not exists
		if err := os.MkdirAll(sessionDir, 0700); err != nil {
			log.Fatalf("Failed to create session directory: %v", err)
		}

		fmt.Printf("🔐 Assuming role %s using source profile %s...\n", roleArn, sourceProfile)

		ctx := context.TODO()

		// Load source profile config
		cfg, err := config.LoadDefaultConfig(ctx, 
			config.WithSharedConfigProfile(sourceProfile),
			config.WithRegion(region))
		if err != nil {
			log.Fatalf("Failed to load source profile %s: %v", sourceProfile, err)
		}

		// Handle MFA if provided
		if mfaArn != "" {
			fmt.Printf("🔒 MFA device detected: %s\n", mfaArn)
			fmt.Print("Enter MFA code: ")
			var mfaCode string
			fmt.Scanln(&mfaCode)

			stsClient := sts.NewFromConfig(cfg)
			input := &sts.GetSessionTokenInput{
				DurationSeconds: aws.Int32(3600),
				SerialNumber:    &mfaArn,
				TokenCode:       &mfaCode,
			}

			result, err := stsClient.GetSessionToken(ctx, input)
			if err != nil {
				log.Fatalf("❌ Failed to get session token with MFA: %v", err)
			}

			cfg.Credentials = aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(
					*result.Credentials.AccessKeyId,
					*result.Credentials.SecretAccessKey,
					*result.Credentials.SessionToken,
				),
			)
			fmt.Println("✅ MFA verification successful.")
		}

		// Assume target IAM role
		stsClient := sts.NewFromConfig(cfg)
		sessionName := fmt.Sprintf("cloudctl-%d", time.Now().Unix())
		duration := int32(3600)

		roleResult, err := stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
			RoleArn:         &roleArn,
			RoleSessionName: &sessionName,
			DurationSeconds: &duration,
		})
		if err != nil {
			log.Fatalf("❌ Failed to assume role: %v", err)
		}

		expiration := *roleResult.Credentials.Expiration

		session := &internal.AWSSession{
			Profile:       profile,
			AccessKey:     *roleResult.Credentials.AccessKeyId,
			SecretKey:     *roleResult.Credentials.SecretAccessKey,
			SessionToken:  *roleResult.Credentials.SessionToken,
			Expiration:    expiration,
			RoleArn:       roleArn,
			SourceProfile: sourceProfile,
		}

		if secretKey != "" {
			if err := internal.SaveCredentials(profile, session, secretKey); err != nil {
				log.Fatalf("❌ Failed to save encrypted session: %v", err)
			}
			fmt.Printf("✅ Encrypted session stored as '%s'\n", profile)
		} else {
			sessionFile := filepath.Join(sessionDir, fmt.Sprintf("%s.json", profile))
			data, _ := json.MarshalIndent(session, "", "  ")
			if err := os.WriteFile(sessionFile, data, 0600); err != nil {
				log.Fatalf("❌ Failed to write session file: %v", err)
			}
			fmt.Printf("✅ Session stored as '%s'\n", profile)
		}

		remaining := time.Until(expiration).Round(time.Minute)
		fmt.Printf("   Role: %s\n", roleArn)
		fmt.Printf("   Source: %s\n", sourceProfile)
		fmt.Printf("   Expires: %s (%v remaining)\n",
			expiration.Local().Format("2006-01-02 15:04:05"), remaining)
	},
}

func init() {
	loginCmd.Flags().StringVar(&sourceProfile, "source", "", "Source AWS CLI profile for base credentials")
	loginCmd.Flags().StringVar(&profile, "profile", "", "Name to store the new session as")
	loginCmd.Flags().StringVar(&roleArn, "role", "", "Target IAM role ARN to assume")
	loginCmd.Flags().StringVar(&mfaArn, "mfa", "", "MFA device ARN (optional)")
	loginCmd.Flags().StringVar(&secretKey, "secret", "", "Optional secret for encryption")
	loginCmd.Flags().StringVar(&region, "region", "ap-southeast-1", "AWS region (default: ap-southeast-1)")
	rootCmd.AddCommand(loginCmd)
}

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	openConsole   bool
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
		var cfg aws.Config
		var err error

		// Check if source profile is a cloudctl session first
		if secretKey != "" {
			session, sessionErr := internal.LoadCredentials(sourceProfile, secretKey)
			if sessionErr == nil {
				// Source is a cloudctl session, use its credentials
				cfg, err = config.LoadDefaultConfig(ctx,
					config.WithRegion(region),
					config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
						session.AccessKey,
						session.SecretKey,
						session.SessionToken,
					)),
				)
				if err != nil {
					log.Fatalf("Failed to configure AWS SDK with session credentials: %v", err)
				}
			} else {
				// Source is an AWS CLI profile
				cfg, err = config.LoadDefaultConfig(ctx,
					config.WithSharedConfigProfile(sourceProfile),
					config.WithRegion(region))
				if err != nil {
					log.Fatalf("Failed to load source profile %s: %v", sourceProfile, err)
				}
			}
		} else {
			// No secret provided, try AWS CLI profile
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithSharedConfigProfile(sourceProfile),
				config.WithRegion(region))
			if err != nil {
				log.Fatalf("Failed to load source profile %s: %v", sourceProfile, err)
			}
		}

		// Handle MFA if provided
		if mfaArn != "" {
			fmt.Printf("🔒 MFA device detected: %s\n", mfaArn)
			mfaCode := readMFACode()

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
		sessionName := profile // Use profile name as session name
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

		// Open console if requested
		if openConsole {
			fmt.Println("\n🌐 Opening AWS Console...")
			if err := openAWSConsole(session, region); err != nil {
				fmt.Printf("⚠️  Failed to open console: %v\n", err)
				fmt.Println("💡 You can open it manually with: cloudctl console --profile", profile, "--open")
			}
		}
	},
}

func openAWSConsole(session *internal.AWSSession, consoleRegion string) error {
	// Create session JSON
	sessionJSON := map[string]string{
		"sessionId":    session.AccessKey,
		"sessionKey":   session.SecretKey,
		"sessionToken": session.SessionToken,
	}

	sessionData, _ := json.Marshal(sessionJSON)

	// Get signin token
	federationURL := "https://signin.aws.amazon.com/federation"
	params := url.Values{}
	params.Add("Action", "getSigninToken")
	params.Add("Session", string(sessionData))

	resp, err := http.Get(fmt.Sprintf("%s?%s", federationURL, params.Encode()))
	if err != nil {
		return fmt.Errorf("failed to get sign-in token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var tokenResp map[string]string
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	signinToken := tokenResp["SigninToken"]
	if signinToken == "" {
		return fmt.Errorf("failed to get sign-in token")
	}

	// Build console URL
	destination := "https://console.aws.amazon.com/"
	if consoleRegion != "" {
		destination = fmt.Sprintf("https://%s.console.aws.amazon.com/console/home?region=%s", consoleRegion, consoleRegion)
	}
	consoleURL := fmt.Sprintf("%s?Action=login&Issuer=cloudctl&Destination=%s&SigninToken=%s",
		federationURL, url.QueryEscape(destination), signinToken)

	// Open in browser
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", consoleURL)
	case "linux":
		cmd = exec.Command("xdg-open", consoleURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", consoleURL)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func init() {
	loginCmd.Flags().StringVar(&sourceProfile, "source", "", "Source AWS CLI profile for base credentials")
	loginCmd.Flags().StringVar(&profile, "profile", "", "Name to store the new session as")
	loginCmd.Flags().StringVar(&roleArn, "role", "", "Target IAM role ARN to assume")
	loginCmd.Flags().StringVar(&mfaArn, "mfa", "", "MFA device ARN (optional)")
	loginCmd.Flags().StringVar(&secretKey, "secret", os.Getenv("CLOUDCTL_SECRET"), "Optional secret for encryption (or set CLOUDCTL_SECRET env var)")
	loginCmd.Flags().StringVar(&region, "region", "ap-southeast-1", "AWS region (default: ap-southeast-1)")
	loginCmd.Flags().BoolVar(&openConsole, "open", false, "Automatically open AWS Console after login")
	rootCmd.AddCommand(loginCmd)
}

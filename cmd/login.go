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
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
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
		// Interactive prompts for missing parameters
		if sourceProfile == "" {
			awsProfiles := listAWSProfiles()
			ctlProfiles, _ := internal.ListProfiles()

			// Merge and deduplicate
			seen := make(map[string]bool)
			var allProfiles []string
			for _, p := range awsProfiles {
				if !seen[p] {
					allProfiles = append(allProfiles, p)
					seen[p] = true
				}
			}
			for _, p := range ctlProfiles {
				if !seen[p] {
					allProfiles = append(allProfiles, p)
					seen[p] = true
				}
			}

			if len(allProfiles) > 0 {
				selected, err := ui.SelectProfile("Select Source Profile", allProfiles)
				if err != nil {
					return
				}
				sourceProfile = selected
			}
		}

		if profile == "" {
			var err error
			profile, err = ui.GetInput("Enter Session Name", "prod-admin", false)
			if err != nil {
				return
			}
		}

		if roleArn == "" {
			var err error
			roleArn, err = ui.GetInput("Enter Role ARN", "arn:aws:iam::123456789012:role/RoleName", false)
			if err != nil {
				return
			}
		}

		if sourceProfile == "" || profile == "" || roleArn == "" {
			fmt.Println("❌ Missing required parameters")
			if sourceProfile == "" {
				fmt.Println("   --source: Source AWS profile or cloudctl session")
			}
			if profile == "" {
				fmt.Println("   --profile: Name for this session")
			}
			if roleArn == "" {
				fmt.Println("   --role: IAM role ARN to assume")
			}
			fmt.Println("\n💡 Example:")
			fmt.Println("   cloudctl login --source default --profile prod-admin --role arn:aws:iam::123456789012:role/AdminRole")
			os.Exit(1)
		}

		// Create session directory if not exists
		if err := os.MkdirAll(sessionDir, 0700); err != nil {
			fmt.Printf("❌ Failed to create session directory: %v\n", err)
			fmt.Printf("💡 Check permissions for: %s\n", sessionDir)
			os.Exit(1)
		}

		// Prepare config (blocking, but usually fast)
		ctx := context.TODO()
		var cfg aws.Config
		var err error

		// Config loading logic...

		// Detect or request secret
		var secret string
		// Only check for secret if user provided one manually, or if they haven't disabled encryption
		// Actually, logic is: try to get secret. If found, use it. If not, ask to create (on macOS) or fail/fallback to plain file ??
		// Original logic: "if secretKey != """.
		// New Logic: Always try to get a secure secret if we are going to store credentials securely.
		// However, we must respect the existing flow.

		useEncryption := false
		secret, err = internal.GetSecret(secretKey)
		if err == nil {
			useEncryption = true
		} else {
			// No secret found. If on macOS, offer to setup keychain.
			if internal.IsMacOS() {
				// Only prompt if we are in interactive mode (profile was not empty means likely non-interactive? No, args check)
				fmt.Println("🔑 No encryption secret found.")
				fmt.Println("   Would you like to generate a secure key and store it in your System Keychain? (y/n)")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) == "y" {
					newSecret, keychainErr := internal.SetupKeychain()
					if keychainErr != nil {
						fmt.Printf("❌ Failed to setup keychain: %v\n", keychainErr)
						// Fallback to unencrypted
					} else {
						secret = newSecret
						useEncryption = true
						fmt.Println("✅ Secure key generated and stored in Keychain.")
					}
				}
			}
		}

		// Config loading logic...
		if useEncryption {
			session, sessionErr := internal.LoadCredentials(sourceProfile, secret)
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
					fmt.Printf("❌ Failed to configure AWS SDK with session credentials: %v\n", err)
					os.Exit(1)
				}
			} else {
				// Source is an AWS CLI profile
				cfg, err = config.LoadDefaultConfig(ctx,
					config.WithSharedConfigProfile(sourceProfile),
					config.WithRegion(region))
				if err != nil {
					fmt.Printf("❌ Profile '%s' not found\n", sourceProfile)

					// Try to list available profiles
					if profiles := listAWSProfiles(); len(profiles) > 0 {
						fmt.Println("\n💡 Available AWS profiles:")
						for _, p := range profiles {
							fmt.Printf("   • %s\n", p)
						}
					}

					// Check for cloudctl sessions
					if sessions, _ := internal.ListProfiles(); len(sessions) > 0 {
						fmt.Println("\n💡 Available cloudctl sessions:")
						for _, s := range sessions {
							fmt.Printf("   • %s\n", s)
						}
					}

					fmt.Println("\n💡 To create a new profile:")
					fmt.Println("   aws configure --profile", sourceProfile)
					os.Exit(1)
				}
			}
		} else {
			// No secret provided, try AWS CLI profile
			cfg, err = config.LoadDefaultConfig(ctx,
				config.WithSharedConfigProfile(sourceProfile),
				config.WithRegion(region))
			if err != nil {
				fmt.Printf("❌ Profile '%s' not found\n", sourceProfile)

				if profiles := listAWSProfiles(); len(profiles) > 0 {
					fmt.Println("\n💡 Available AWS profiles:")
					for _, p := range profiles {
						fmt.Printf("   • %s\n", p)
					}
				}

				fmt.Println("\n💡 To create a new profile:")
				fmt.Println("   aws configure --profile", sourceProfile)
				os.Exit(1)
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
				fmt.Printf("❌ MFA authentication failed: %v\n", err)
				fmt.Println("\n💡 Common issues:")
				fmt.Println("   • Check your MFA code is current (not expired)")
				fmt.Println("   • Verify MFA device ARN is correct")
				fmt.Println("   • Ensure device time is synchronized")
				fmt.Printf("   • MFA ARN format: arn:aws:iam::<account-id>:mfa/<username>\n")
				os.Exit(1)
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

		// Assume target IAM role with spinner
		stsClient := sts.NewFromConfig(cfg)
		sessionName := profile // Use profile name as session name
		duration := int32(3600)

		res, err := ui.Spin(fmt.Sprintf("Assuming role %s...", roleArn), func() (any, error) {
			return stsClient.AssumeRole(ctx, &sts.AssumeRoleInput{
				RoleArn:         &roleArn,
				RoleSessionName: &sessionName,
				DurationSeconds: &duration,
			})
		})

		if err != nil {
			fmt.Printf("❌ Failed to assume role: %v\n", err)
			fmt.Println("\n💡 Common issues:")
			fmt.Println("   • Check the role ARN is correct")
			fmt.Println("   • Verify the role's trust policy allows your source identity")
			fmt.Println("   • Ensure your source credentials have sts:AssumeRole permission")
			fmt.Println("   • Check if the role requires MFA (use --mfa flag)")
			fmt.Printf("\n💡 Role ARN format: arn:aws:iam::<account-id>:role/<role-name>\n")
			os.Exit(1)
		}

		roleResult, ok := res.(*sts.AssumeRoleOutput)
		if !ok || roleResult == nil {
			fmt.Println("❌ Internal error: invalid response from AssumeRole")
			os.Exit(1)
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

		if useEncryption {
			if err := internal.SaveCredentials(profile, session, secret); err != nil {
				fmt.Printf("❌ Failed to save encrypted session: %v\n", err)
				fmt.Printf("💡 Check permissions for: %s\n", filepath.Join(os.Getenv("HOME"), ".cloudctl"))
				os.Exit(1)
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

// listAWSProfiles reads AWS CLI profiles from ~/.aws/credentials and ~/.aws/config
func listAWSProfiles() []string {
	profiles := make(map[string]bool)

	// Check credentials file
	credPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
	if data, err := os.ReadFile(credPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
				profile := strings.Trim(line, "[]")
				profiles[profile] = true
			}
		}
	}

	// Check config file
	configPath := filepath.Join(os.Getenv("HOME"), ".aws", "config")
	if data, err := os.ReadFile(configPath); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[profile ") && strings.HasSuffix(line, "]") {
				profile := strings.TrimPrefix(strings.Trim(line, "[]"), "profile ")
				profiles[profile] = true
			} else if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") && !strings.Contains(line, " ") {
				profile := strings.Trim(line, "[]")
				profiles[profile] = true
			}
		}
	}

	// Convert to sorted slice
	result := make([]string, 0, len(profiles))
	for p := range profiles {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
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

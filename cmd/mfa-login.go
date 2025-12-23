package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	mfaSourceProfile string
	mfaProfile       string
	mfaDeviceArn     string
	mfaSecretKey     string
	mfaDuration      int32
)

var mfaLoginCmd = &cobra.Command{
	Use:   "mfa-login",
	Short: "Get MFA session token to use for multiple role assumptions",
	Long: `Authenticate with MFA once and get a session token valid for up to 12 hours.
Use this session as source profile for subsequent role assumptions without re-entering MFA.`,
	Example: `  # Get MFA session (valid for 12 hours)
  cloudctl mfa-login --source default --profile mfa-session --mfa arn:aws:iam::123:mfa/user
  
  # Use MFA session to assume multiple roles (no MFA needed)
  cloudctl login --source mfa-session --profile role1 --role arn:aws:iam::123:role/Role1
  cloudctl login --source mfa-session --profile role2 --role arn:aws:iam::456:role/Role2`,
	Run: func(cmd *cobra.Command, args []string) {
		// Interactive prompts for missing parameters
		if mfaSourceProfile == "" {
			awsProfiles := listAWSProfiles()
			if len(awsProfiles) > 0 {
				selected, err := ui.SelectProfile("Select Source Profile", awsProfiles)
				if err != nil {
					return
				}
				mfaSourceProfile = selected
			}
		}

		if mfaProfile == "" {
			var err error
			mfaProfile, err = ui.GetInput("Enter MFA Session Name", "mfa-session", false)
			if err != nil {
				return
			}
		}

		if mfaDeviceArn == "" {
			// Check if we have stored devices
			devices, _ := internal.ListMFADevices()
			if len(devices) > 0 {
				// Convert to selection list
				var deviceNames []string
				for name, arn := range devices {
					deviceNames = append(deviceNames, fmt.Sprintf("%s (%s)", name, arn))
				}
				sort.Strings(deviceNames)

				selected, err := ui.SelectProfile("Select MFA Device", deviceNames)
				if err == nil {
					// Parse selected string "name (arn)"
					parts := strings.SplitN(selected, " (", 2)
					mfaDeviceArn = devices[parts[0]]
				}
			}

			// If still empty (no selection or no saved devices), prompt for input
			if mfaDeviceArn == "" {
				var err error
				mfaDeviceArn, err = ui.GetInput("Enter MFA Device ARN", "arn:aws:iam::123:mfa/user", false)
				if err != nil {
					return
				}
			}
		} else {
			// Check if input matches an alias
			if arn, found := internal.GetMFADevice(mfaDeviceArn); found {
				fmt.Printf("📱 Using stored device '%s'\n", mfaDeviceArn)
				mfaDeviceArn = arn
			}
		}

		if mfaSourceProfile == "" || mfaProfile == "" || mfaDeviceArn == "" {
			fmt.Println("❌ Missing required parameters")
			if mfaSourceProfile == "" {
				fmt.Println("   --source: Source AWS profile")
			}
			if mfaProfile == "" {
				fmt.Println("   --profile: Name for this MFA session")
			}
			if mfaDeviceArn == "" {
				fmt.Println("   --mfa: MFA device ARN")
			}
			fmt.Println("\n💡 Example:")
			fmt.Println("   cloudctl mfa-login --source default --profile mfa-session --mfa arn:aws:iam::123456789012:mfa/username")
			os.Exit(1)
		}

		fmt.Printf("🔐 Getting MFA session token from profile %s...\n", mfaSourceProfile)

		ctx := context.TODO()

		// Load source profile config
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithSharedConfigProfile(mfaSourceProfile),
			config.WithRegion(region))
		if err != nil {
			fmt.Printf("❌ Profile '%s' not found\n", mfaSourceProfile)
			fmt.Println("\n💡 To create a new profile:")
			fmt.Println("   aws configure --profile", mfaSourceProfile)
			os.Exit(1)
		}

		// Prompt for MFA code (masked input)
		mfaCode := readMFACode()

		// Get session token with MFA using spinner
		stsClient := sts.NewFromConfig(cfg)
		input := &sts.GetSessionTokenInput{
			DurationSeconds: &mfaDuration,
			SerialNumber:    &mfaDeviceArn,
			TokenCode:       &mfaCode,
		}

		res, err := ui.Spin("Authenticating with MFA...", func() (any, error) {
			return stsClient.GetSessionToken(ctx, input)
		})

		if err != nil {
			fmt.Printf("❌ MFA authentication failed: %v\n", err)
			fmt.Println("\n💡 Common issues:")
			fmt.Println("   • Check your MFA code is current (not expired)")
			fmt.Println("   • Verify MFA device ARN is correct")
			fmt.Println("   • Ensure device time is synchronized")
			fmt.Printf("   • MFA ARN format: arn:aws:iam::<account-id>:mfa/<username>\n")
			os.Exit(1)
		}

		result, ok := res.(*sts.GetSessionTokenOutput)
		if !ok || result == nil {
			fmt.Println("❌ Internal error: invalid response from GetSessionToken")
			os.Exit(1)
		}

		expiration := *result.Credentials.Expiration

		// Store the MFA session
		session := &internal.AWSSession{
			Profile:       mfaProfile,
			AccessKey:     *result.Credentials.AccessKeyId,
			SecretKey:     *result.Credentials.SecretAccessKey,
			SessionToken:  *result.Credentials.SessionToken,
			Expiration:    expiration,
			RoleArn:       "MFA-Session", // Special marker
			SourceProfile: mfaSourceProfile,
		}

		// Get secret from flag, env, or keychain
		secret, err := internal.GetSecret(mfaSecretKey)
		if err != nil {
			// If on macOS and no secret found, offer to create one in keychain
			if internal.IsMacOS() {
				fmt.Println("🔑 No encryption secret found.")
				fmt.Println("   Would you like to generate a secure key and store it in your System Keychain? (y/n)")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) == "y" {
					newSecret, err := internal.SetupKeychain()
					if err != nil {
						fmt.Printf("❌ Failed to setup keychain: %v\n", err)
						return
					}
					secret = newSecret
					fmt.Println("✅ Secure key generated and stored in Keychain.")
				} else {
					fmt.Println("❌ Operation cancelled. Secret required.")
					return
				}
			} else {
				fmt.Println("❌ Encryption secret required")
				fmt.Println("\n💡 Set the secret:")
				fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
				fmt.Println("   cloudctl mfa-login --source", mfaSourceProfile, "--profile", mfaProfile, "--mfa", mfaDeviceArn)
				os.Exit(1)
			}
		}

		if err := internal.SaveCredentials(mfaProfile, session, secret); err != nil {
			fmt.Printf("❌ Failed to save encrypted session: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✅ MFA session stored as '%s'\n", mfaProfile)

		remaining := time.Until(expiration).Round(time.Minute)
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60

		fmt.Printf("   MFA Device: %s\n", mfaDeviceArn)
		fmt.Printf("   Source: %s\n", mfaSourceProfile)
		fmt.Printf("   Expires: %s (%dh%dm remaining)\n",
			expiration.Local().Format("2006-01-02 15:04:05"), hours, minutes)
		fmt.Printf("\n💡 Now you can assume roles without MFA:\n")
		fmt.Printf("   cloudctl login --source %s --profile <name> --role <role-arn>\n", mfaProfile)
	},
}

func init() {
	mfaLoginCmd.Flags().StringVar(&mfaSourceProfile, "source", "", "Source AWS CLI profile for base credentials")
	mfaLoginCmd.Flags().StringVar(&mfaProfile, "profile", "", "Name to store the MFA session as")
	mfaLoginCmd.Flags().StringVar(&mfaDeviceArn, "mfa", "", "MFA device ARN")
	mfaLoginCmd.Flags().StringVar(&mfaSecretKey, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret for encryption (or set CLOUDCTL_SECRET env var)")
	mfaLoginCmd.Flags().Int32Var(&mfaDuration, "duration", 43200, "Session duration in seconds (default: 43200 = 12 hours, max: 129600 = 36 hours)")
	rootCmd.AddCommand(mfaLoginCmd)
}

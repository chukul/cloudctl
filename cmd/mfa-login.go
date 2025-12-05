package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
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

		// Get session token with MFA
		stsClient := sts.NewFromConfig(cfg)
		input := &sts.GetSessionTokenInput{
			DurationSeconds: &mfaDuration,
			SerialNumber:    &mfaDeviceArn,
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

		if mfaSecretKey != "" {
			if err := internal.SaveCredentials(mfaProfile, session, mfaSecretKey); err != nil {
				fmt.Printf("❌ Failed to save encrypted session: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("✅ MFA session stored as '%s'\n", mfaProfile)
		} else {
			fmt.Println("❌ Encryption secret required")
			fmt.Println("\n💡 Set the secret:")
			fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			fmt.Println("   cloudctl mfa-login --source", mfaSourceProfile, "--profile", mfaProfile, "--mfa", mfaDeviceArn)
			os.Exit(1)
		}

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

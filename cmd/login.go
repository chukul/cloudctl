package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/yourname/cloudctl/internal"
)

var (
	sourceProfile string
	profile       string
	roleArn       string
	secretKey     string
	sessionName   string
)

func init() {
	loginCmd.Flags().StringVar(&sourceProfile, "source", "default", "Base AWS CLI profile used to assume the role")
	loginCmd.Flags().StringVar(&profile, "profile", "", "Name to store credentials under (target profile)")
	loginCmd.Flags().StringVar(&roleArn, "role", "", "Role ARN to assume")
	loginCmd.Flags().StringVar(&secretKey, "secret", "", "32-byte local encryption key")
	loginCmd.Flags().StringVar(&sessionName, "session-name", "cloudctl-session", "STS session name")

	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Assume AWS IAM role and store encrypted credentials",
	Run: func(cmd *cobra.Command, args []string) {
		if roleArn == "" || secretKey == "" || profile == "" {
			log.Fatal("Error: --role, --secret, and --profile are required.")
		}

		fmt.Printf("🔐 Assuming role %s using base profile %s...\n", roleArn, sourceProfile)

		sess, err := internal.AssumeRole(sourceProfile, roleArn, sessionName)
		if err != nil {
			log.Fatalf("Failed to assume role: %v", err)
		}

		sess.Profile = profile

		err = internal.SaveCredentials(profile, sess, secretKey)
		if err != nil {
			log.Fatalf("Failed to save credentials: %v", err)
		}

		fmt.Printf("✅ Credentials stored under profile %s\n", profile)
	},
}

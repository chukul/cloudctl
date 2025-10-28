package cmd

import (
	"fmt"
	"log"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

func init() {
	exportCmd.Flags().StringVar(&profile, "profile", "default", "Profile to export credentials for")
	exportCmd.Flags().StringVar(&secretKey, "secret", "", "Local encryption key (same used during login)")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export AWS credentials as environment variables (auto-refresh if expired)",
	Run: func(cmd *cobra.Command, args []string) {
		if secretKey == "" {
			log.Fatal("Error: --secret is required.")
		}

		sess, err := internal.LoadCredentials(profile, secretKey)
		if err != nil {
			log.Fatalf("Failed to load credentials: %v", err)
		}

		sess, refreshed, err := internal.RefreshIfExpired(sess, secretKey)
		if err != nil {
			log.Fatalf("Error during refresh: %v", err)
		}
		if refreshed {
			fmt.Println("✅ Session refreshed successfully.")
		}

		fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", sess.AccessKey)
		fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", sess.SecretKey)
		fmt.Printf("export AWS_SESSION_TOKEN=%s\n", sess.SessionToken)
	},
}

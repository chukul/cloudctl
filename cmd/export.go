package cmd

import (
	"fmt"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var exportProfile string
var exportSecret string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export stored AWS session as environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		if exportProfile == "" {
			fmt.Println("❌ You must specify --profile to export")
			return
		}

		if exportSecret == "" {
			fmt.Println("❌ You must specify --secret to decrypt credentials")
			return
		}

		s, err := internal.LoadCredentials(exportProfile, exportSecret)
		if err != nil {
			fmt.Printf("❌ Failed to load session for profile '%s': %v\n", exportProfile, err)
			return
		}

		// Output shell-compatible export commands
		fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", s.AccessKey)
		fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", s.SecretKey)
		fmt.Printf("export AWS_SESSION_TOKEN=%s\n", s.SessionToken)
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportProfile, "profile", "", "Profile to export")
	exportCmd.Flags().StringVar(&exportSecret, "secret", "", "Secret key for decryption (optional)")
	rootCmd.AddCommand(exportCmd)
}

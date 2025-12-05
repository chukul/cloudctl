package cmd

import (
	"fmt"
	"os"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var switchSecret string

var switchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Quick switch to a profile and export credentials",
	Long:  `Switch to a profile and export credentials in one command. Use with eval to set environment variables.`,
	Args:  cobra.ExactArgs(1),
	Example: `  # Switch to a profile
  eval $(cloudctl switch prod-admin --secret "your-secret")
  
  # Or set CLOUDCTL_SECRET environment variable
  export CLOUDCTL_SECRET="your-secret"
  eval $(cloudctl switch prod-admin)`,
	Run: func(cmd *cobra.Command, args []string) {
		profile := args[0]

		if switchSecret == "" {
			fmt.Println("❌ Encryption secret required")
			fmt.Println("\n💡 Set the secret:")
			fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			fmt.Println("   eval $(cloudctl switch", profile, ")")
			return
		}

		s, err := internal.LoadCredentials(profile, switchSecret)
		if err != nil {
			fmt.Printf("❌ Profile '%s' not found\n", profile)
			
			// List available profiles
			if profiles, _ := internal.ListProfiles(); len(profiles) > 0 {
				fmt.Println("\n💡 Available profiles:")
				for _, p := range profiles {
					fmt.Printf("   • %s\n", p)
				}
			} else {
				fmt.Println("\n💡 No sessions found. Create one with:")
				fmt.Println("   cloudctl login --source <profile> --profile <name> --role <role-arn>")
			}
			return
		}

		// Output shell-compatible export commands
		fmt.Printf("export AWS_ACCESS_KEY_ID=%s\n", s.AccessKey)
		fmt.Printf("export AWS_SECRET_ACCESS_KEY=%s\n", s.SecretKey)
		fmt.Printf("export AWS_SESSION_TOKEN=%s\n", s.SessionToken)
	},
}

func init() {
	switchCmd.Flags().StringVar(&switchSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	rootCmd.AddCommand(switchCmd)
}

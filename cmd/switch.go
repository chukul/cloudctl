package cmd

import (
	"fmt"
	"os"
	"sort"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var switchSecret string

var switchCmd = &cobra.Command{
	Use:   "switch [profile]",
	Short: "Quick switch to a profile and export credentials",
	Long:  `Switch to a profile and export credentials in one command. Use with eval to set environment variables.`,
	Args:  cobra.MaximumNArgs(1),
	Example: `  # Switch to a profile interactively
  eval $(cloudctl switch)
  
  # Switch to a specific profile
  eval $(cloudctl switch prod-admin --secret "your-secret")
  
  # Or set CLOUDCTL_SECRET environment variable
  export CLOUDCTL_SECRET="your-secret"
  eval $(cloudctl switch prod-admin)`,
	Run: func(cmd *cobra.Command, args []string) {
		var profile string

		if len(args) == 0 {
			// Interactive mode
			profiles, err := internal.ListProfiles()
			if err != nil || len(profiles) == 0 {
				fmt.Fprintln(os.Stderr, "❌ No profiles found. Create one first.")
				return
			}
			sort.Strings(profiles)

			selected, err := ui.SelectProfile("Select Profile", profiles)
			if err != nil {
				// User cancelled or error
				return
			}
			profile = selected
		} else {
			profile = args[0]
		}

		secret, err := internal.GetSecret(switchSecret)
		if err != nil {
			// Ask if user wants to setup keychain if on macOS
			if internal.IsMacOS() {
				// TODO: We could add an interactive prompt here to setup keychain
				// For now, just show the standard error
			}

			fmt.Fprintf(os.Stderr, "❌ Encryption secret required\n")
			fmt.Fprintf(os.Stderr, "\n💡 Set the secret:\n")
			fmt.Fprintf(os.Stderr, "   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"\n")
			fmt.Fprintf(os.Stderr, "   eval $(cloudctl switch %s)\n", profile)
			return
		}

		s, err := internal.LoadCredentials(profile, secret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Profile '%s' not found\n", profile)

			// List available profiles
			if profiles, _ := internal.ListProfiles(); len(profiles) > 0 {
				fmt.Fprintf(os.Stderr, "\n💡 Available profiles:\n")
				for _, p := range profiles {
					fmt.Fprintf(os.Stderr, "   • %s\n", p)
				}
			} else {
				fmt.Fprintf(os.Stderr, "\n💡 No sessions found. Create one with:\n")
				fmt.Fprintf(os.Stderr, "   cloudctl login --source <profile> --profile <name> --role <role-arn>\n")
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

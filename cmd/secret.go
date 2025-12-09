package cmd

import (
	"fmt"
	"strings"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage encryption secret",
	Long:  `Manage the encryption secret used to protect your AWS credentials.`,
}

var secretShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current keychain secret",
	Long:  "Reveal the secret stored in your macOS Keychain. Usage of this command requires Touch ID authentication.",
	Run: func(cmd *cobra.Command, args []string) {
		if !internal.IsMacOS() {
			fmt.Println("❌ Keychain integration is only available on macOS")
			return
		}

		// Re-authentication implicitly handled by System Keychain access control
		// When we request the item, OS will prompt user
		secret, err := internal.GetSecret("")
		if err != nil {
			fmt.Println("❌ No secret found in Keychain or it couldn't be accessed.")
			return
		}

		fmt.Println("🔐 Your CloudCtl Encryption Secret:")
		fmt.Println(strings.Repeat("─", 64))
		fmt.Println(secret)
		fmt.Println(strings.Repeat("─", 64))
		fmt.Println("\n⚠️  KEEP THIS SAFE! You will need it to restore access on another machine.")
		fmt.Println("   To restore: cloudctl secret import <key>")
	},
}

var secretImportCmd = &cobra.Command{
	Use:   "import [key]",
	Short: "Import a secret into keychain",
	Long:  "Save an existing secret key into your macOS Keychain for passwordless operation.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !internal.IsMacOS() {
			fmt.Println("❌ Keychain integration is only available on macOS")
			return
		}

		var key string
		if len(args) > 0 {
			key = args[0]
		} else {
			var err error
			key, err = ui.GetInput("Enter Secret Key to Import", "", true)
			if err != nil {
				return
			}
		}

		if key == "" {
			fmt.Println("❌ Secret key cannot be empty")
			return
		}

		if err := internal.StoreKeychainSecret(key); err != nil {
			fmt.Printf("❌ Failed to store secret: %v\n", err)
			return
		}

		fmt.Println("✅ Secret imported successfully to Keychain!")
	},
}

func init() {
	secretCmd.AddCommand(secretShowCmd)
	secretCmd.AddCommand(secretImportCmd)
	rootCmd.AddCommand(secretCmd)
}

package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var mfaCmd = &cobra.Command{
	Use:   "mfa",
	Short: "Manage AWS MFA devices",
	Long:  `List, add, and remove MFA devices alias.`,
}

var mfaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MFA devices",
	Run: func(cmd *cobra.Command, args []string) {
		devices, err := internal.ListMFADevices()
		if err != nil {
			fmt.Printf("❌ Failed to load MFA devices: %v\n", err)
			return
		}

		if len(devices) == 0 {
			fmt.Println("📭 No MFA devices found.")
			fmt.Println("\n💡 Add one with:")
			fmt.Println("   cloudctl mfa add <name> <arn>")
			return
		}

		// Sort by name
		names := make([]string, 0, len(devices))
		for k := range devices {
			names = append(names, k)
		}
		sort.Strings(names)

		fmt.Println("MFA Devices")
		fmt.Println(strings.Repeat("─", 80))
		for _, name := range names {
			fmt.Printf("%-20s %s\n", name, devices[name])
		}
	},
}

var mfaAddCmd = &cobra.Command{
	Use:   "add <name> <arn>",
	Short: "Add an MFA device alias",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		arn := args[1]

		// Basic validation
		if !strings.HasPrefix(arn, "arn:aws:iam::") || !strings.Contains(arn, ":mfa/") {
			fmt.Println("⚠️  Warning: The ARN provided doesn't look like a standard MFA ARN.")
			fmt.Println("   Standard format: arn:aws:iam::<account-id>:mfa/<username>")
		}

		if err := internal.SaveMFADevice(name, arn); err != nil {
			fmt.Printf("❌ Failed to save device: %v\n", err)
			return
		}

		fmt.Printf("✅ Added MFA device '%s'\n", name)
	},
}

var mfaRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an MFA device",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if err := internal.RemoveMFADevice(name); err != nil {
			fmt.Printf("❌ Failed to remove device: %v\n", err)
			return
		}

		fmt.Printf("✅ Removed MFA device '%s'\n", name)
	},
}

func init() {
	mfaCmd.AddCommand(mfaListCmd)
	mfaCmd.AddCommand(mfaAddCmd)
	mfaCmd.AddCommand(mfaRemoveCmd)
	rootCmd.AddCommand(mfaCmd)
}

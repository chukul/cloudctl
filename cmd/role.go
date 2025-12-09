package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:   "role",
	Short: "Manage AWS IAM Role aliases",
	Long:  `List, add, and remove IAM Role aliases for easier login.`,
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved IAM Roles",
	Run: func(cmd *cobra.Command, args []string) {
		roles, err := internal.ListRoles()
		if err != nil {
			fmt.Printf("❌ Failed to load roles: %v\n", err)
			return
		}

		if len(roles) == 0 {
			fmt.Println("📭 No IAM Roles found.")
			fmt.Println("\n💡 Add one with:")
			fmt.Println("   cloudctl role add <name> <arn>")
			return
		}

		// Sort by name
		names := make([]string, 0, len(roles))
		for k := range roles {
			names = append(names, k)
		}
		sort.Strings(names)

		fmt.Println("IAM Roles")
		fmt.Println(strings.Repeat("─", 80))
		for _, name := range names {
			fmt.Printf("%-20s %s\n", name, roles[name])
		}
	},
}

var roleAddCmd = &cobra.Command{
	Use:   "add <name> <arn>",
	Short: "Add an IAM Role alias",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		arn := args[1]

		// Basic validation
		if !strings.HasPrefix(arn, "arn:aws:iam::") || !strings.Contains(arn, ":role/") {
			fmt.Println("⚠️  Warning: The ARN provided doesn't look like a standard IAM Role ARN.")
			fmt.Println("   Standard format: arn:aws:iam::<account-id>:role/<role-name>")
		}

		if err := internal.SaveRole(name, arn); err != nil {
			fmt.Printf("❌ Failed to save role: %v\n", err)
			return
		}

		fmt.Printf("✅ Added IAM Role '%s'\n", name)
	},
}

var roleRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an IAM Role alias",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		if err := internal.RemoveRole(name); err != nil {
			fmt.Printf("❌ Failed to remove role: %v\n", err)
			return
		}

		fmt.Printf("✅ Removed IAM Role '%s'\n", name)
	},
}

func init() {
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleAddCmd)
	roleCmd.AddCommand(roleRemoveCmd)
	rootCmd.AddCommand(roleCmd)
}

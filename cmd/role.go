package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	roleRemoveAll bool
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
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove one or all IAM Role aliases",
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if roleRemoveAll {
			fmt.Print("⚠️  This will remove ALL saved IAM Role aliases. Type 'yes' to confirm: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(input) != "yes" {
				fmt.Println("❌ Operation cancelled.")
				return
			}

			if err := internal.ClearAllRoles(); err != nil {
				fmt.Printf("❌ Failed to clear roles: %v\n", err)
				return
			}
			fmt.Println("✅ All IAM Role aliases removed successfully.")
			return
		}

		var name string
		if len(args) == 0 {
			roles, err := internal.ListRoles()
			if err != nil || len(roles) == 0 {
				fmt.Println("📭 No IAM Roles found.")
				return
			}

			var names []string
			for k := range roles {
				names = append(names, k)
			}
			sort.Strings(names)

			selected, err := ui.SelectProfile("Select Role Alias to Remove", names)
			if err != nil {
				return
			}
			name = selected
		} else {
			name = args[0]
		}

		if err := internal.RemoveRole(name); err != nil {
			fmt.Printf("❌ Failed to remove role: %v\n", err)
			return
		}

		fmt.Printf("✅ Removed IAM Role '%s'\n", name)
	},
}

var roleExportCmd = &cobra.Command{
	Use:   "export [file.json]",
	Short: "Export all IAM Role aliases to JSON",
	Run: func(cmd *cobra.Command, args []string) {
		roles, err := internal.ListRoles()
		if err != nil {
			fmt.Printf("❌ Failed to load roles: %v\n", err)
			return
		}

		b, _ := json.MarshalIndent(roles, "", "  ")

		if len(args) > 0 {
			err := os.WriteFile(args[0], b, 0644)
			if err != nil {
				fmt.Printf("❌ Failed to write file: %v\n", err)
				return
			}
			fmt.Printf("✅ Exported %d roles to %s\n", len(roles), args[0])
		} else {
			fmt.Println(string(b))
		}
	},
}

var roleImportCmd = &cobra.Command{
	Use:   "import <file.json>",
	Short: "Import IAM Role aliases from JSON",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		filePath := args[0]
		b, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Printf("❌ Failed to read file: %v\n", err)
			return
		}

		var importedRoles map[string]string
		if err := json.Unmarshal(b, &importedRoles); err != nil {
			fmt.Printf("❌ Failed to parse JSON: %v\n", err)
			return
		}

		currentRoles, _ := internal.ListRoles()
		mergedCount := 0
		for name, arn := range importedRoles {
			currentRoles[name] = arn
			mergedCount++
		}

		if err := internal.SaveAllRoles(currentRoles); err != nil {
			fmt.Printf("❌ Failed to save roles: %v\n", err)
			return
		}

		fmt.Printf("✅ Successfully imported/merged %d roles\n", mergedCount)
	},
}

func init() {
	roleRemoveCmd.Flags().BoolVar(&roleRemoveAll, "all", false, "Remove all stored IAM Role aliases")

	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleAddCmd)
	roleCmd.AddCommand(roleRemoveCmd)
	roleCmd.AddCommand(roleExportCmd)
	roleCmd.AddCommand(roleImportCmd)
	rootCmd.AddCommand(roleCmd)
}

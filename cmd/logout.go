package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	logoutProfile string
	logoutAll     bool
)

func init() {
	logoutCmd.Flags().StringVar(&logoutProfile, "profile", "", "Profile name to remove from credential store")
	logoutCmd.Flags().BoolVar(&logoutAll, "all", false, "Remove all stored profiles")
	rootCmd.AddCommand(logoutCmd)
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials for a profile or all profiles",
	Run: func(cmd *cobra.Command, args []string) {
		if !logoutAll && logoutProfile == "" {
			profiles, err := internal.ListProfiles()
			if err != nil || len(profiles) == 0 {
				fmt.Println("❌ No stored profiles found.")
				return
			}

			selected, err := ui.SelectProfile("Select Profile to Logout", profiles)
			if err != nil {
				return
			}
			logoutProfile = selected
		}

		if logoutAll {
			fmt.Print("⚠️  This will remove all stored credentials. Type 'yes' to confirm: ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.TrimSpace(input) != "yes" {
				fmt.Println("❌ Operation cancelled.")
				return
			}

			err := internal.ClearAllCredentials()
			if err != nil {
				log.Fatalf("Failed to clear credentials: %v", err)
			}
			fmt.Println("✅ All profiles removed successfully.")
			return
		}

		err := internal.RemoveProfile(logoutProfile)
		if err != nil {
			log.Fatalf("Failed to remove profile %s: %v", logoutProfile, err)
		}

		fmt.Printf("✅ Profile '%s' removed successfully.\n", logoutProfile)
	},
}

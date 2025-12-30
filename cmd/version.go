package cmd

import (
	"fmt"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cloudctl version %s\n", internal.CurrentVersion)

		// Force check for updates
		latest, url, err := internal.FetchLatestVersion()
		if err != nil {
			fmt.Printf("Unable to check for updates: %v\n", err)
			return
		}

		if internal.IsNewer(latest, internal.CurrentVersion) {
			fmt.Printf("\n💡 Update available: %s → %s\n", internal.CurrentVersion, latest)
			fmt.Printf("   Download: %s\n", url)
		} else {
			fmt.Println("✅ You're running the latest version")
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

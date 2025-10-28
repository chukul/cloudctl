package cmd

import (
	"fmt"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored profiles",
	Run: func(cmd *cobra.Command, args []string) {
		profiles, _ := internal.ListProfiles()
		if len(profiles) == 0 {
			fmt.Println("No profiles found.")
			return
		}
		for _, p := range profiles {
			fmt.Println("📦", p)
		}
	},
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const logo = `
   ________                _________ __  __
  / ____/ /___  __  ______/ / ____// /_/ /
 / /   / / __ \/ / / / __  / /    / __/ / 
/ /___/ / /_/ / /_/ / /_/ / /___ / /_/ /  
\____/_/\____/\__,_/\__,_/\____/ \__/_/   
                                           
`

var rootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "cloudctl is a CLI tool for managing AWS sessions and credentials",
	Long:  logo + `A lightweight Leapp-like CLI for securely managing AWS AssumeRole sessions.`,
}

// Execute runs the CLI
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

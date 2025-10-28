package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "cloudctl is a CLI tool for managing AWS sessions and credentials",
	Long:  `A lightweight Leapp-like CLI for securely managing AWS AssumeRole sessions.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

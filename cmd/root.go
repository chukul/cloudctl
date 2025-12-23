package cmd

import (
	"fmt"
	"os"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

func printLogo() {
	// Gradient colors (Blue -> Purple -> Pink)
	// Blue: 0, 176, 255
	// Purple: 170, 0, 255
	// Pink: 255, 0, 128

	ascii := []string{
		`   ██████╗██╗      ██████╗ ██╗   ██╗██████╗  ██████╗████████╗██╗     `,
		`  ██╔════╝██║     ██╔═══██╗██║   ██║██╔══██╗██╔════╝╚══██╔══╝██║     `,
		`  ██║     ██║     ██║   ██║██║   ██║██║  ██║██║        ██║   ██║     `,
		`  ██║     ██║     ██║   ██║██║   ██║██║  ██║██║        ██║   ██║     `,
		`  ╚██████╗███████╗╚██████╔╝╚██████╔╝██████╔╝╚██████╗   ██║   ███████╗`,
		`   ╚═════╝╚══════╝ ╚═════╝  ╚═════╝ ╚═════╝  ╚═════╝   ╚═╝   ╚══════╝`,
	}

	fmt.Println()
	for _, line := range ascii {
		for i, char := range line {
			// Calculate gradient ratio (0.0 to 1.0)
			ratio := float64(i) / float64(len(line))

			var r, g, b int
			if ratio < 0.5 {
				// Blue to Purple
				subRatio := ratio * 2
				r = int(0*(1-subRatio) + 170*subRatio)
				g = int(176*(1-subRatio) + 0*subRatio)
				b = int(255*(1-subRatio) + 255*subRatio)
			} else {
				// Purple to Pink
				subRatio := (ratio - 0.5) * 2
				r = int(170*(1-subRatio) + 255*subRatio)
				g = int(0*(1-subRatio) + 0*subRatio)
				b = int(255*(1-subRatio) + 128*subRatio)
			}

			fmt.Printf("\x1b[38;2;%d;%d;%dm%c\x1b[0m", r, g, b, char)
		}
		fmt.Println()
	}
	fmt.Println("\x1b[1m  A lightweight tool for securely managing AWS sessions with MFA & Touch ID\x1b[0m")
	fmt.Println("  Author: Chuchai Kultanahiran <chuchaik@outlook.com>")
	fmt.Println()
}

var rootCmd = &cobra.Command{
	Use:   "cloudctl",
	Short: "cloudctl is a CLI tool for managing AWS sessions and credentials",
	Long:  `CloudCtl helps you manage multiple AWS accounts and sessions securely with encryption and system keychain integration.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Check for updates on every command (non-blocking)
		internal.CheckForUpdates()
	},
}

// Execute runs the CLI
func Execute() {
	if len(os.Args) <= 1 || (len(os.Args) > 1 && os.Args[1] == "help") {
		printLogo()
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

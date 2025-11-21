package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var statusSecret string

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show stored AWS sessions",
	Run: func(cmd *cobra.Command, args []string) {
		sessions, err := internal.ListAllSessions(statusSecret)
		if err != nil {
			fmt.Printf("Failed to load sessions: %v\n", err)
			return
		}

		if len(sessions) == 0 {
			fmt.Println("No stored sessions found.")
			return
		}

		fmt.Printf("%-15s %-40s %-20s %-12s %-8s\n", "PROFILE", "ROLE ARN", "EXPIRATION", "REMAINING", "STATUS")
		fmt.Println(strings.Repeat("-", 100))

		now := time.Now()
		for _, s := range sessions {
			remaining := s.Expiration.Sub(now)
			
			var status string
			var statusColor string
			
			if remaining <= 0 {
				status = "EXPIRED"
				statusColor = colorRed
				remaining = 0
			} else if remaining <= 15*time.Minute {
				status = "EXPIRING"
				statusColor = colorYellow
			} else {
				status = "ACTIVE"
				statusColor = colorGreen
			}

			fmt.Printf("%-15s %-40s %-20s %-12s %s%-8s%s\n",
				s.Profile,
				s.RoleArn,
				s.Expiration.Format("2006-01-02 15:04:05"),
				remaining.Round(time.Second),
				statusColor,
				status,
				colorReset,
			)
		}
	},
}

func init() {
	statusCmd.Flags().StringVar(&statusSecret, "secret", "", "Secret key for session decryption")
	rootCmd.AddCommand(statusCmd)
}

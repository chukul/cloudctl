package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var statusSecret string

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
			status := "ACTIVE"
			remaining := s.Expiration.Sub(now)
			if remaining <= 0 {
				status = "EXPIRED"
				remaining = 0
			}

			fmt.Printf("%-15s %-40s %-20s %-12s %-8s\n",
				s.Profile,
				s.RoleArn,
				s.Expiration.Format("2006-01-02 15:04:05"),
				remaining.Round(time.Second),
				status,
			)
		}
	},
}

func init() {
	statusCmd.Flags().StringVar(&statusSecret, "secret", "", "Secret key for session decryption")
	rootCmd.AddCommand(statusCmd)
}

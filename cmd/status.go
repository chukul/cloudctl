package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yourname/cloudctl/internal"
)

var (
	statusProfile string
	outputJSON    bool
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show all stored AWS sessions with status, expiration, and remaining time (like Leapp dashboard)",
	Run: func(cmd *cobra.Command, args []string) {
		if secretKey == "" {
			log.Fatal("Error: --secret is required to decrypt stored sessions")
		}

		sessions, err := internal.ListAllSessions(secretKey)
		if err != nil {
			log.Fatalf("Failed to list sessions: %v", err)
		}

		if len(sessions) == 0 {
			fmt.Println("No stored sessions found.")
			return
		}

		// Filter by profile if provided
		if statusProfile != "" {
			filtered := []*internal.AWSSession{}
			for _, s := range sessions {
				if s.Profile == statusProfile {
					filtered = append(filtered, s)
				}
			}
			sessions = filtered
			if len(sessions) == 0 {
				fmt.Printf("No session found for profile: %s\n", statusProfile)
				return
			}
		}

		// Sort by expiration ascending
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].Expiration.Before(sessions[j].Expiration)
		})

		// Optional JSON output
		if outputJSON {
			jsonData, _ := json.MarshalIndent(sessions, "", "  ")
			fmt.Println(string(jsonData))
			return
		}

		// Fancy table header
		header := color.New(color.FgCyan, color.Bold).SprintFunc()
		fmt.Printf("%-20s %-50s %-25s %-15s %-10s\n",
			header("PROFILE"), header("ROLE ARN"), header("EXPIRATION"), header("REMAINING"), header("STATUS"))
		fmt.Println(strings.Repeat("-", 130))

		now := time.Now()
		for _, s := range sessions {
			status := "ACTIVE"
			statusColor := color.New(color.FgGreen).SprintFunc()

			if s.Revoked {
				status = "REVOKED"
				statusColor = color.New(color.FgRed).SprintFunc()
			} else if s.Expiration.Before(now) {
				status = "EXPIRED"
				statusColor = color.New(color.FgYellow).SprintFunc()
			}

			// Compute remaining time
			var remaining string
			if s.Revoked {
				remaining = "-"
			} else if s.Expiration.Before(now) {
				remaining = "Expired"
			} else {
				diff := s.Expiration.Sub(now)
				h := int(diff.Hours())
				m := int(diff.Minutes()) % 60
				remaining = fmt.Sprintf("%dh%dm left", h, m)
			}

			exp := s.Expiration.Format("2006-01-02 15:04:05")

			fmt.Printf("%-20s %-50s %-25s %-15s %-10s\n",
				s.Profile,
				truncateText(s.RoleArn, 48),
				exp,
				remaining,
				statusColor(status),
			)
		}
	},
}

func init() {
	statusCmd.Flags().StringVar(&secretKey, "secret", "", "Decryption key used in login/export")
	statusCmd.Flags().StringVar(&statusProfile, "profile", "", "Filter by specific profile")
	statusCmd.Flags().BoolVar(&outputJSON, "json", false, "Output results in JSON format for automation")
	rootCmd.AddCommand(statusCmd)
}

func truncateText(text string, max int) string {
	if len(text) > max {
		return text[:max-3] + "..."
	}
	return text
}

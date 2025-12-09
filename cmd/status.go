package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
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
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

type sessionStatus int

const (
	statusActive sessionStatus = iota
	statusExpiring
	statusExpired
)

type sessionDisplay struct {
	session   *internal.AWSSession
	status    sessionStatus
	remaining time.Duration
	icon      string
	isCurrent bool
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show stored AWS sessions",
	Run: func(cmd *cobra.Command, args []string) {
		// Get secret from flag, env, or keychain
		secret, err := internal.GetSecret(statusSecret)
		if err != nil {
			fmt.Println("❌ Encryption secret required to view session status")
			fmt.Println("\n💡 Set the secret:")
			fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			return
		}

		sessions, err := internal.ListAllSessions(secret)
		if err != nil {
			fmt.Printf("❌ Failed to load sessions: %v\n", err)
			return
		}

		if len(sessions) == 0 {
			fmt.Println("📭 No stored sessions found.")
			fmt.Println("\n💡 Get started:")
			fmt.Println("   cloudctl mfa-login --source <profile> --profile mfa-session --mfa <mfa-arn>")
			fmt.Println("   cloudctl login --source <profile> --profile <name> --role <role-arn>")
			return
		}

		// Get current session from environment
		currentAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")

		// Prepare display data
		now := time.Now()
		displays := make([]sessionDisplay, 0, len(sessions))

		for _, s := range sessions {
			remaining := s.Expiration.Sub(now)
			var status sessionStatus
			var icon string

			if remaining <= 0 {
				status = statusExpired
				icon = "🔴"
				remaining = 0
			} else if remaining <= 15*time.Minute {
				status = statusExpiring
				icon = "🟡"
			} else {
				status = statusActive
				icon = "🟢"
			}

			// Check if MFA session
			if s.RoleArn == "MFA-Session" || s.RoleArn == "" {
				icon = "🔒"
			}

			displays = append(displays, sessionDisplay{
				session:   s,
				status:    status,
				remaining: remaining,
				icon:      icon,
				isCurrent: s.AccessKey == currentAccessKey,
			})
		}

		// Sort by status (active -> expiring -> expired), then by remaining time
		sort.Slice(displays, func(i, j int) bool {
			if displays[i].status != displays[j].status {
				return displays[i].status < displays[j].status
			}
			return displays[i].remaining > displays[j].remaining
		})

		// Print grouped by status
		printSessionGroup(displays, statusActive, "Active Sessions")
		printSessionGroup(displays, statusExpiring, "Expiring Soon")
		printSessionGroup(displays, statusExpired, "Expired Sessions")
	},
}

func printSessionGroup(displays []sessionDisplay, status sessionStatus, title string) {
	filtered := make([]sessionDisplay, 0)
	for _, d := range displays {
		if d.status == status {
			filtered = append(filtered, d)
		}
	}

	if len(filtered) == 0 {
		return
	}

	fmt.Printf("\n%s%s%s\n", colorBold, title, colorReset)
	fmt.Println(strings.Repeat("─", 120))

	for _, d := range filtered {
		s := d.session
		accountID := extractAccountID(s.RoleArn)
		roleName := extractRoleName(s.RoleArn)

		// Format profile name with current indicator
		profileDisplay := s.Profile
		if d.isCurrent {
			profileDisplay = fmt.Sprintf("%s%s ← current%s", colorCyan, s.Profile, colorReset)
		}

		// Format role display
		roleDisplay := s.RoleArn
		if roleName != "" && accountID != "" {
			roleDisplay = fmt.Sprintf("%s (%s)", roleName, accountID)
		} else if s.RoleArn == "MFA-Session" || s.RoleArn == "" {
			roleDisplay = fmt.Sprintf("%sMFA Session%s", colorDim, colorReset)
		}

		// Format remaining time
		remainingStr := formatDuration(d.remaining)
		if d.status == statusExpired {
			remainingStr = fmt.Sprintf("%sexpired%s", colorDim, colorReset)
		}

		fmt.Printf("%s %-25s %-50s %s\n",
			d.icon,
			profileDisplay,
			roleDisplay,
			remainingStr,
		)

		// Show expiration time in dim color
		fmt.Printf("   %sExpires: %s%s\n",
			colorDim,
			s.Expiration.Local().Format("2006-01-02 15:04:05"),
			colorReset,
		)
	}
}

func extractAccountID(roleArn string) string {
	re := regexp.MustCompile(`arn:aws:iam::(\d+):role/`)
	matches := re.FindStringSubmatch(roleArn)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func extractRoleName(roleArn string) string {
	re := regexp.MustCompile(`arn:aws:iam::\d+:role/(.+)`)
	matches := re.FindStringSubmatch(roleArn)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm remaining", hours, minutes)
	}
	return fmt.Sprintf("%dm remaining", minutes)
}

func init() {
	statusCmd.Flags().StringVar(&statusSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for session decryption (or set CLOUDCTL_SECRET env var)")
	rootCmd.AddCommand(statusCmd)
}

package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var statusSecret string

// ANSI color codes are replaced with lipgloss styles

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4A90E2")).
		MarginBottom(1)

	rowActiveStyle   = lipgloss.NewStyle().MarginBottom(0)
	rowExpiringStyle = lipgloss.NewStyle().MarginBottom(0)
	rowExpiredStyle  = lipgloss.NewStyle().MarginBottom(0).Faint(true)

	profileActiveStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF")).Bold(true)
	profileExpiringStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623")).Bold(true)
	profileExpiredStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#D0021B")).Bold(true)

	currentStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4A90E2"))
	roleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5"))
	sourceStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#78909C"))
	timeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#90A4AE"))
	
	activeTagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7ED321")).Bold(true)
	expiringTagStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#F5A623")).Bold(true)
	expiredTagStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#D0021B")).Bold(true)
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

		// Add tip for expired sessions
		hasExpired := false
		for _, d := range displays {
			if d.status == statusExpired {
				hasExpired = true
				break
			}
		}
		if hasExpired {
			fmt.Println(lipgloss.NewStyle().MarginTop(1).Foreground(lipgloss.Color("#4A90E2")).Render("💡 Tip: ") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5")).Render("Use ") +
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render("cloudctl refresh [profile]") +
				lipgloss.NewStyle().Foreground(lipgloss.Color("#B0BEC5")).Render(" to quickly restore expired sessions."))
		}
	},
}

func printSessionGroup(displays []sessionDisplay, status sessionStatus, title string) {
	var filtered []sessionDisplay
	for _, d := range displays {
		if d.status == status {
			filtered = append(filtered, d)
		}
	}

	if len(filtered) == 0 {
		return
	}

	var profileStyle lipgloss.Style

	switch status {
	case statusActive:
		profileStyle = profileActiveStyle
	case statusExpiring:
		profileStyle = profileExpiringStyle
	case statusExpired:
		profileStyle = profileExpiredStyle
	}

	fmt.Printf("\n%s\n", titleStyle.Render(title))
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#4A90E2")).Render(strings.Repeat("─", 100)))

	for _, d := range filtered {
		s := d.session
		accountID := extractAccountID(s.RoleArn)
		roleName := extractRoleName(s.RoleArn)

		// Format profile name with current indicator
		profileDisplay := profileStyle.Render(s.Profile)
		if d.isCurrent {
			profileDisplay += " " + currentStyle.Render("← current")
		}

		// Format role display
		roleDisplay := roleStyle.Render(s.RoleArn)
		if roleName != "" && accountID != "" {
			roleDisplay = roleStyle.Render(fmt.Sprintf("%s (%s)", roleName, accountID))
		} else if s.RoleArn == "MFA-Session" || s.RoleArn == "" {
			roleDisplay = sourceStyle.Render("MFA Session")
		}

		// Format remaining time
		remainingStr := timeStyle.Render(formatDuration(d.remaining))
		if d.status == statusExpired {
			remainingStr = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")).Render("Expired")
		}

		// Use lipgloss to format exact widths while respecting ANSI sequences
		profileCol := lipgloss.NewStyle().Width(25).Render(profileDisplay)
		roleCol := lipgloss.NewStyle().Width(50).Render(roleDisplay)
		timeCol := lipgloss.NewStyle().Width(20).Align(lipgloss.Right).Render(remainingStr)

		// Line 1: Profile, Role, Remaining Time
		fmt.Printf("%s %s %s %s\n", d.icon, profileCol, roleCol, timeCol)

		// Line 2: Source Info and Expiration
		sourceInfo := ""
		if s.SourceProfile != "" && s.RoleArn != "MFA-Session" {
			sourceInfo = fmt.Sprintf("Source: %-12s ", s.SourceProfile)
		}
		
		fmt.Printf("   %s%s\n",
			sourceStyle.Render(sourceInfo),
			sourceStyle.Render("Expires: "+internal.FormatBKK(s.Expiration)),
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

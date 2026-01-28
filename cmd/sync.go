package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var syncSecret string
var syncAll bool
var syncProfile string

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync stored sessions to ~/.aws/credentials",
	Long: `Export cloudctl managed sessions to the standard AWS credentials file (~/.aws/credentials).
This allows external tools (Terraform, VS Code, etc.) to use your assumed roles directly.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Get secret from flag, env, or keychain
		secret, err := internal.GetSecret(syncSecret)
		if err != nil {
			fmt.Println("❌ Encryption secret required")
			return
		}

		if syncAll {
			count, err := internal.SyncAllToAWS(secret)
			if err != nil {
				fmt.Printf("❌ Sync failed: %v\n", err)
				return
			}
			credsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
			fmt.Printf("✅ Synced %d profiles to %s\n", count, credsPath)
			return
		}

		profile := syncProfile
		if profile == "" && len(args) > 0 {
			profile = args[0]
		}

		// Load all sessions
		allSessions, err := internal.ListAllSessions(secret)
		if err != nil {
			fmt.Printf("❌ Failed to load sessions: %v\n", err)
			return
		}

		if len(allSessions) == 0 {
			fmt.Println("📭 No stored sessions found.")
			return
		}

		// Filter out expired sessions
		now := time.Now()
		var activeSessions []*internal.AWSSession
		for _, s := range allSessions {
			if s.Expiration.After(now) {
				activeSessions = append(activeSessions, s)
			}
		}

		if len(activeSessions) == 0 {
			fmt.Println("⚠️  No active (non-expired) sessions found to sync.")
			return
		}

		// Filter sessions if profile specified
		var sessionsToSync []*internal.AWSSession
		if profile != "" {
			for _, s := range activeSessions {
				if s.Profile == profile {
					sessionsToSync = append(sessionsToSync, s)
					break
				}
			}
			if len(sessionsToSync) == 0 {
				fmt.Printf("❌ Profile '%s' not found or is expired.\n", profile)
				return
			}
		} else {
			// Interactive Selection
			var options []string
			optionToProfile := make(map[string]string)
			for _, s := range activeSessions {
				sessionType := "Role"
				if s.RoleArn == "MFA-Session" {
					sessionType = "MFA"
				}
				displayName := fmt.Sprintf("%-15s (%s)", s.Profile, sessionType)
				options = append(options, displayName)
				optionToProfile[displayName] = s.Profile
			}
			sort.Strings(options)

			selected, err := ui.SelectProfile("Select Active Profile to Sync (MFA or Role)", options)
			if err != nil {
				return
			}

			selectedProfile := optionToProfile[selected]
			for _, s := range activeSessions {
				if s.Profile == selectedProfile {
					sessionsToSync = append(sessionsToSync, s)
					break
				}
			}
		}

		if len(sessionsToSync) == 0 {
			fmt.Println("⚠️  No sessions to sync.")
			return
		}

		credsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")
		// Read existing credentials file
		content, err := os.ReadFile(credsPath)
		var existingLines []string
		if err == nil {
			existingLines = strings.Split(string(content), "\n")
		}

		// Remove cloudctl managed sections
		newLines := []string{}
		skipSection := false
		for i := 0; i < len(existingLines); i++ {
			line := existingLines[i]
			trimmed := strings.TrimSpace(line)

			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				pName := strings.Trim(trimmed, "[]")
				skipSection = false
				for _, s := range sessionsToSync {
					if s.Profile == pName {
						skipSection = true
						break
					}
				}
			}

			if strings.HasPrefix(trimmed, "; Managed by cloudctl") {
				foundH := ""
				for j := i + 1; j < len(existingLines); j++ {
					tj := strings.TrimSpace(existingLines[j])
					if tj == "" || strings.HasPrefix(tj, ";") {
						continue
					}
					if strings.HasPrefix(tj, "[") && strings.HasSuffix(tj, "]") {
						foundH = strings.Trim(tj, "[]")
					}
					break
				}
				if foundH != "" {
					isReplacing := false
					for _, s := range sessionsToSync {
						if s.Profile == foundH {
							isReplacing = true
							break
						}
					}
					if isReplacing {
						continue
					}
				}
			}

			if !skipSection {
				newLines = append(newLines, line)
			}
		}

		for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
			newLines = newLines[:len(newLines)-1]
		}
		if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
			newLines = append(newLines, "")
		}

		syncedCount := 0
		for _, s := range sessionsToSync {
			sessionType := "Role Session"
			if s.RoleArn == "MFA-Session" {
				sessionType = "MFA Session"
			}
			newLines = append(newLines, fmt.Sprintf("; Managed by cloudctl (%s) - Expires: %s", sessionType, internal.FormatBKK(s.Expiration)))
			newLines = append(newLines, fmt.Sprintf("[%s]", s.Profile))
			newLines = append(newLines, fmt.Sprintf("aws_access_key_id = %s", s.AccessKey))
			newLines = append(newLines, fmt.Sprintf("aws_secret_access_key = %s", s.SecretKey))
			newLines = append(newLines, fmt.Sprintf("aws_session_token = %s", s.SessionToken))
			newLines = append(newLines, "")
			syncedCount++
		}

		if err := os.WriteFile(credsPath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
			fmt.Printf("❌ Failed to write credentials file: %v\n", err)
			return
		}

		fmt.Printf("✅ Synced %d profiles to %s\n", syncedCount, credsPath)
	},
}

func init() {
	syncCmd.Flags().StringVar(&syncSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "Sync all active sessions")
	syncCmd.Flags().StringVar(&syncProfile, "profile", "", "Profile to sync")
	rootCmd.AddCommand(syncCmd)
}

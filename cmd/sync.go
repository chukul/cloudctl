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
			fmt.Println("\n💡 Set the secret or use macOS Keychain:")
			fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			return
		}

		credsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")

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
		if syncAll {
			sessionsToSync = activeSessions
		} else if syncProfile != "" {
			for _, s := range activeSessions {
				if s.Profile == syncProfile {
					sessionsToSync = append(sessionsToSync, s)
					break
				}
			}
			if len(sessionsToSync) == 0 {
				fmt.Printf("❌ Profile '%s' not found or is expired.\n", syncProfile)
				return
			}
		} else {
			// Interactive Selection
			var profiles []string
			for _, s := range activeSessions {
				profiles = append(profiles, s.Profile)
			}
			sort.Strings(profiles)

			selected, err := ui.SelectProfile("Select Active Profile to Sync to ~/.aws/credentials", profiles)
			if err != nil {
				return
			}

			for _, s := range activeSessions {
				if s.Profile == selected {
					sessionsToSync = append(sessionsToSync, s)
					break
				}
			}
		}

		if len(sessionsToSync) == 0 {
			fmt.Println("⚠️  No sessions to sync.")
			return
		}

		// Read existing credentials file
		content, err := os.ReadFile(credsPath)
		var existingLines []string
		if err == nil {
			existingLines = strings.Split(string(content), "\n")
		}

		// Remove cloudctl managed sections and their comments
		newLines := []string{}
		skipSection := false
		for i := 0; i < len(existingLines); i++ {
			line := existingLines[i]
			trimmed := strings.TrimSpace(line)

			// 1. Detect section start
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				profileName := strings.Trim(trimmed, "[]")
				skipSection = false
				for _, s := range sessionsToSync {
					if s.Profile == profileName {
						skipSection = true
						break
					}
				}
			}

			// 2. Identify and skip CloudCtl comments if they belong to a profile being replaced
			if strings.HasPrefix(trimmed, "; Managed by cloudctl") {
				foundHeader := ""
				// Look ahead for the next profile header
				for j := i + 1; j < len(existingLines); j++ {
					tj := strings.TrimSpace(existingLines[j])
					if tj == "" || strings.HasPrefix(tj, ";") {
						continue
					}
					if strings.HasPrefix(tj, "[") && strings.HasSuffix(tj, "]") {
						foundHeader = strings.Trim(tj, "[]")
					}
					break
				}

				if foundHeader != "" {
					isReplacing := false
					for _, s := range sessionsToSync {
						if s.Profile == foundHeader {
							isReplacing = true
							break
						}
					}
					if isReplacing {
						continue // Skip this comment line
					}
				}
			}

			if !skipSection {
				newLines = append(newLines, line)
			}
		}

		// 3. Clean up potential trailing empty lines after filtering
		for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
			newLines = newLines[:len(newLines)-1]
		}

		// Append synced sessions
		if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
			newLines = append(newLines, "")
		}

		syncedCount := 0
		for _, s := range sessionsToSync {
			// Add comment identifying it as cloudctl managed
			newLines = append(newLines, fmt.Sprintf("; Managed by cloudctl - Expires: %s", s.Expiration.Local().Format("2006-01-02 15:04:05")))
			newLines = append(newLines, fmt.Sprintf("[%s]", s.Profile))
			newLines = append(newLines, fmt.Sprintf("aws_access_key_id = %s", s.AccessKey))
			newLines = append(newLines, fmt.Sprintf("aws_secret_access_key = %s", s.SecretKey))
			newLines = append(newLines, fmt.Sprintf("aws_session_token = %s", s.SessionToken))
			newLines = append(newLines, "")
			syncedCount++
		}

		// Write back
		output := strings.Join(newLines, "\n")
		if err := os.WriteFile(credsPath, []byte(output), 0600); err != nil {
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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chukul/cloudctl/internal"
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
		if syncSecret == "" {
			fmt.Println("❌ You must specify --secret to decrypt credentials")
			return
		}

		credsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")

		// Load all sessions
		sessions, err := internal.ListAllSessions(syncSecret)
		if err != nil {
			fmt.Printf("❌ Failed to load sessions: %v\n", err)
			return
		}

		// Filter sessions if profile specified
		var sessionsToSync []*internal.AWSSession
		if syncAll {
			sessionsToSync = sessions
		} else if syncProfile != "" {
			for _, s := range sessions {
				if s.Profile == syncProfile {
					sessionsToSync = append(sessionsToSync, s)
					break
				}
			}
			if len(sessionsToSync) == 0 {
				fmt.Printf("❌ Profile '%s' not found.\n", syncProfile)
				return
			}
		} else {
			fmt.Println("❌ You must specify --profile or --all")
			return
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

		// Remove cloudctl managed sections from existing content to avoid duplicates
		// A robust ini parser would be better, but simple section replacement works for now
		newLines := []string{}
		skipSection := false
		for _, line := range existingLines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
				profileName := strings.Trim(trimmed, "[]")
				// Check if this is one of the profiles we are syncing
				skipSection = false
				for _, s := range sessionsToSync {
					if s.Profile == profileName {
						skipSection = true
						break
					}
				}
			}
			if !skipSection {
				newLines = append(newLines, line)
			}
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

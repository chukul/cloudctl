package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SyncAllToAWS loads all active sessions and syncs them to ~/.aws/credentials.
// This is used by both the 'sync' command and automatically by 'refresh' and the daemon.
func SyncAllToAWS(secret string) (int, error) {
	credsPath := filepath.Join(os.Getenv("HOME"), ".aws", "credentials")

	// 1. Load all sessions
	allSessions, err := ListAllSessions(secret)
	if err != nil {
		return 0, fmt.Errorf("failed to load sessions: %w", err)
	}

	if len(allSessions) == 0 {
		return 0, nil
	}

	// 2. Filter out expired sessions
	now := time.Now()
	var activeSessions []*AWSSession
	for _, s := range allSessions {
		if s.Expiration.After(now) {
			activeSessions = append(activeSessions, s)
		}
	}

	if len(activeSessions) == 0 {
		return 0, nil
	}

	// 3. Read existing credentials file
	content, err := os.ReadFile(credsPath)
	var existingLines []string
	if err == nil {
		existingLines = strings.Split(string(content), "\n")
	}

	// 4. Remove cloudctl managed sections and their comments
	newLines := []string{}
	skipSection := false
	for i := 0; i < len(existingLines); i++ {
		line := existingLines[i]
		trimmed := strings.TrimSpace(line)

		// Detect section start
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			profileName := strings.Trim(trimmed, "[]")
			skipSection = false
			for _, s := range activeSessions {
				if s.Profile == profileName {
					skipSection = true
					break
				}
			}
		}

		// Identify and skip CloudCtl comments if they belong to a profile being replaced
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
				for _, s := range activeSessions {
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

	// 5. Clean up potential trailing empty lines after filtering
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}

	// 6. Append synced sessions
	if len(newLines) > 0 && newLines[len(newLines)-1] != "" {
		newLines = append(newLines, "")
	}

	syncedCount := 0
	for _, s := range activeSessions {
		sessionType := "Role Session"
		if s.RoleArn == "MFA-Session" {
			sessionType = "MFA Session"
		}

		// Add comment identifying it as cloudctl managed
		newLines = append(newLines, fmt.Sprintf("; Managed by cloudctl (%s) - Expires: %s", sessionType, FormatBKK(s.Expiration)))
		newLines = append(newLines, fmt.Sprintf("[%s]", s.Profile))
		newLines = append(newLines, fmt.Sprintf("aws_access_key_id = %s", s.AccessKey))
		newLines = append(newLines, fmt.Sprintf("aws_secret_access_key = %s", s.SecretKey))
		newLines = append(newLines, fmt.Sprintf("aws_session_token = %s", s.SessionToken))
		newLines = append(newLines, "")
		syncedCount++
	}

	// 7. Write back
	output := strings.Join(newLines, "\n")
	if err := os.WriteFile(credsPath, []byte(output), 0600); err != nil {
		return 0, fmt.Errorf("failed to write credentials file: %w", err)
	}

	return syncedCount, nil
}

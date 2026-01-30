package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var promptSecret string

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Display current session info for shell prompt",
	Long:  `Display current AWS session information formatted for shell prompts. Shows profile name and time remaining.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Priority 1: Check by CLOUDCTL_PROFILE (pinned session)
		activeProfile := os.Getenv("CLOUDCTL_PROFILE")
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")

		if activeProfile == "" && accessKey == "" {
			return // No AWS context
		}

		secret, err := internal.GetSecret(promptSecret)
		if err != nil {
			return // Silent fail for prompt
		}

		var currentSession *internal.AWSSession

		// Optimization: If profile is known, just load that one
		if activeProfile != "" {
			s, err := internal.LoadCredentials(activeProfile, secret)
			// Verify it matches the current access key if set
			if err == nil && (accessKey == "" || s.AccessKey == accessKey) {
				currentSession = s
			}
		}

		// Fallback: Scan all sessions if profile didn't match or wasn't set
		if currentSession == nil && accessKey != "" {
			sessions, err := internal.ListAllSessions(secret)
			if err == nil {
				for _, s := range sessions {
					if s.AccessKey == accessKey {
						currentSession = s
						break
					}
				}
			}
		}

		if currentSession == nil {
			return
		}

		// Calculate remaining time
		remaining := time.Until(currentSession.Expiration)
		if remaining <= 0 {
			// Red for expired
			fmt.Printf("\033[31m☁️  %s (expired)\033[0m", currentSession.Profile)
			return
		}

		// Format remaining time
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60

		// ANSI Colors: Green for >15m, Yellow for <=15m
		color := "\033[32m" // Green
		if remaining <= 15*time.Minute {
			color = "\033[33m" // Yellow
		}

		if hours > 0 {
			fmt.Printf("%s☁️  %s (%dh%dm)\033[0m", color, currentSession.Profile, hours, minutes)
		} else {
			fmt.Printf("%s☁️  %s (%dm)\033[0m", color, currentSession.Profile, minutes)
		}
	},
}

var promptInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display detailed session info in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		activeProfile := os.Getenv("CLOUDCTL_PROFILE")
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")

		if activeProfile == "" && accessKey == "" {
			fmt.Println("{}")
			return
		}

		secret, err := internal.GetSecret(promptSecret)
		if err != nil {
			fmt.Println("{}")
			return
		}

		var currentSession *internal.AWSSession
		if activeProfile != "" {
			s, err := internal.LoadCredentials(activeProfile, secret)
			if err == nil && (accessKey == "" || s.AccessKey == accessKey) {
				currentSession = s
			}
		}

		if currentSession == nil && accessKey != "" {
			sessions, err := internal.ListAllSessions(secret)
			if err == nil {
				for _, s := range sessions {
					if s.AccessKey == accessKey {
						currentSession = s
						break
					}
				}
			}
		}

		if currentSession == nil {
			fmt.Println("{}")
			return
		}

		remaining := time.Until(currentSession.Expiration)
		info := map[string]interface{}{
			"profile":    currentSession.Profile,
			"role_arn":   currentSession.RoleArn,
			"expiration": internal.FormatBKK(currentSession.Expiration),
			"remaining":  int(remaining.Seconds()),
			"expired":    remaining <= 0,
		}

		output, _ := json.Marshal(info)
		fmt.Println(string(output))
	},
}

var promptSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Show shell integration setup instructions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print(`
Shell Prompt Integration Setup
================================

Add CloudCtl session info to your shell prompt by adding these lines to your shell config:

Bash (~/.bashrc or ~/.bash_profile):
------------------------------------
cloudctl_prompt() {
  cloudctl prompt 2>/dev/null
}
# Add to your PS1:
PS1='$(cloudctl_prompt)\u@\h:\w\$ '


Zsh (~/.zshrc):
---------------
cloudctl_prompt() {
  cloudctl prompt 2>/dev/null
}
# Add to your prompt:
PROMPT='$(cloudctl_prompt)%n@%m:%~%# '


Fish (~/.config/fish/config.fish):
----------------------------------
function fish_prompt
    cloudctl prompt 2>/dev/null
    set_color normal
    echo -n (whoami)@(hostname):(prompt_pwd)'> '
end


💡 Tip: If you use macOS Keychain or have CLOUDCTL_SECRET set, 
   the prompt will automatically detect and decrypt your active session.
   Use 'eval $(cloudctl switch)' to pin a specific profile.
`)
	},
}

func init() {
	promptCmd.Flags().StringVar(&promptSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	promptInfoCmd.Flags().StringVar(&promptSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")

	promptCmd.AddCommand(promptInfoCmd)
	promptCmd.AddCommand(promptSetupCmd)
	rootCmd.AddCommand(promptCmd)
}

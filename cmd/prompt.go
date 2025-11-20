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
		// Check if AWS credentials are set in environment
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		if accessKey == "" {
			return // No output if no credentials
		}

		if promptSecret == "" {
			return // Need secret to decrypt
		}

		// Load all sessions
		sessions, err := internal.ListAllSessions(promptSecret)
		if err != nil || len(sessions) == 0 {
			return
		}

		// Find matching session by access key
		var currentSession *internal.AWSSession
		for _, s := range sessions {
			if s.AccessKey == accessKey {
				currentSession = s
				break
			}
		}

		if currentSession == nil {
			return
		}

		// Calculate remaining time
		remaining := time.Until(currentSession.Expiration)
		if remaining <= 0 {
			fmt.Printf("☁️  %s (expired)", currentSession.Profile)
			return
		}

		// Format remaining time
		hours := int(remaining.Hours())
		minutes := int(remaining.Minutes()) % 60

		if hours > 0 {
			fmt.Printf("☁️  %s (%dh%dm)", currentSession.Profile, hours, minutes)
		} else {
			fmt.Printf("☁️  %s (%dm)", currentSession.Profile, minutes)
		}
	},
}

var promptInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Display detailed session info in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		if accessKey == "" {
			fmt.Println("{}")
			return
		}

		if promptSecret == "" {
			fmt.Println("{}")
			return
		}

		sessions, err := internal.ListAllSessions(promptSecret)
		if err != nil || len(sessions) == 0 {
			fmt.Println("{}")
			return
		}

		var currentSession *internal.AWSSession
		for _, s := range sessions {
			if s.AccessKey == accessKey {
				currentSession = s
				break
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
			"expiration": currentSession.Expiration.Format(time.RFC3339),
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
		fmt.Println(`
Shell Prompt Integration Setup
================================

Add CloudCtl session info to your shell prompt by adding these lines to your shell config:

Bash (~/.bashrc or ~/.bash_profile):
------------------------------------
export CLOUDCTL_SECRET="your-32-char-secret-key"

cloudctl_prompt() {
  cloudctl prompt --secret "$CLOUDCTL_SECRET" 2>/dev/null
}

# Add to your PS1:
PS1='$(cloudctl_prompt) \u@\h:\w\$ '


Zsh (~/.zshrc):
---------------
export CLOUDCTL_SECRET="your-32-char-secret-key"

cloudctl_prompt() {
  cloudctl prompt --secret "$CLOUDCTL_SECRET" 2>/dev/null
}

# Add to your prompt:
PROMPT='$(cloudctl_prompt) %n@%m:%~%# '


Fish (~/.config/fish/config.fish):
----------------------------------
set -gx CLOUDCTL_SECRET "your-32-char-secret-key"

function fish_prompt
    set_color green
    cloudctl prompt --secret $CLOUDCTL_SECRET 2>/dev/null
    set_color normal
    echo -n ' '
    set_color blue
    echo -n (whoami)@(hostname):(prompt_pwd)
    set_color normal
    echo -n '> '
end


After setup, your prompt will show:
☁️  prod-admin (45m) user@host:~$

Note: Set CLOUDCTL_SECRET environment variable with your encryption key.
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

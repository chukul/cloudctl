package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var execSecret string

var execCmd = &cobra.Command{
	Use:   "exec [profile] -- <command> [args...]",
	Short: "Execute a command with AWS credentials injected into the environment",
	Long:  `Executes a specific command with temporary AWS credentials from the chosen session without altering your global shell environment.`,
	Example: `  # Run terraform with a specific profile
  cloudctl exec prod-admin -- terraform plan
  
  # Run interactively (it will prompt for profile automatically)
  cloudctl exec -- aws s3 ls`,
	Run: func(cmd *cobra.Command, args []string) {
		dashIndex := cmd.ArgsLenAtDash()
		var profile string
		var commandArgs []string

		if dashIndex == 0 {
			// e.g. cloudctl exec -- aws s3 ls
			commandArgs = args
		} else if dashIndex == 1 {
			// e.g. cloudctl exec prod-admin -- aws s3 ls
			profile = args[0]
			commandArgs = args[1:]
		} else if dashIndex > 1 {
			fmt.Fprintln(os.Stderr, "❌ Invalid syntax. Use: cloudctl exec [profile] -- <command>")
			os.Exit(1)
		} else {
			// dashIndex == -1 (no dash provided)
			if len(args) < 1 {
				fmt.Fprintln(os.Stderr, "❌ A command to execute is required.")
				fmt.Fprintln(os.Stderr, "💡 Example: cloudctl exec prod-admin -- aws s3 ls")
				os.Exit(1)
			}
			
			// We can't definitively tell if args[0] is a profile or a command if there's no dash.
			// But traditionally `exec [profile] <command>` means args[0] is profile and args[1:] is command.
			if len(args) >= 2 {
				profile = args[0]
				commandArgs = args[1:]
			} else {
				// Only 1 arg, treat it as the command and ask for profile interactively.
				commandArgs = args
			}
		}

		if len(commandArgs) == 0 {
			fmt.Fprintln(os.Stderr, "❌ A command to execute is required.")
			os.Exit(1)
		}

		// Get secret to decrypt session
		secret, err := internal.GetSecret(execSecret)
		if err != nil {
			fmt.Fprintln(os.Stderr, "❌ Encryption secret required")
			fmt.Fprintln(os.Stderr, "\n💡 Set the secret or use macOS Keychain:")
			fmt.Fprintln(os.Stderr, "   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			os.Exit(1)
		}

		// Interactive profile selection if not provided
		if profile == "" {
			allSessions, err := internal.ListAllSessions(secret)
			if err != nil {
				fmt.Fprintln(os.Stderr, "❌ Failed to load sessions.")
				os.Exit(1)
			}

			now := time.Now()
			var options []string
			optionToProfile := make(map[string]string)

			for _, s := range allSessions {
				// Only show active sessions
				if s.Expiration.After(now) {
					sessionType := "Role"
					if s.RoleArn == "MFA-Session" {
						sessionType = "MFA"
					}
					displayName := fmt.Sprintf("%-15s (%s)", s.Profile, sessionType)
					options = append(options, displayName)
					optionToProfile[displayName] = s.Profile
				}
			}

			if len(options) == 0 {
				fmt.Fprintln(os.Stderr, "📭 No active sessions found. Create one first.")
				os.Exit(1)
			}
			sort.Strings(options)

			selected, err := ui.SelectProfile("Select Account context for Execution", options)
			if err != nil {
				os.Exit(1)
			}
			profile = optionToProfile[selected]
		}

		// Load credentials
		s, err := internal.LoadCredentials(profile, secret)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Profile '%s' not found or expired.\n", profile)
			os.Exit(1)
		}

		// Set up environment
		env := os.Environ()
		
		// Remove existing AWS_* environment variables to avoid conflicts
		var cleanEnv []string
		for _, e := range env {
			if !hasPrefix(e, "AWS_ACCESS_KEY_ID=") &&
				!hasPrefix(e, "AWS_SECRET_ACCESS_KEY=") &&
				!hasPrefix(e, "AWS_SESSION_TOKEN=") &&
				!hasPrefix(e, "AWS_PROFILE=") &&
				!hasPrefix(e, "AWS_REGION=") &&
				!hasPrefix(e, "AWS_DEFAULT_REGION=") {
				cleanEnv = append(cleanEnv, e)
			}
		}

		// Inject new credentials
		cleanEnv = append(cleanEnv, fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", s.AccessKey))
		cleanEnv = append(cleanEnv, fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", s.SecretKey))
		cleanEnv = append(cleanEnv, fmt.Sprintf("AWS_SESSION_TOKEN=%s", s.SessionToken))
		if s.Region != "" {
			cleanEnv = append(cleanEnv, fmt.Sprintf("AWS_REGION=%s", s.Region))
			cleanEnv = append(cleanEnv, fmt.Sprintf("AWS_DEFAULT_REGION=%s", s.Region))
		}
		
		targetCmd := exec.Command(commandArgs[0], commandArgs[1:]...)
		targetCmd.Env = cleanEnv
		targetCmd.Stdin = os.Stdin
		targetCmd.Stdout = os.Stdout
		targetCmd.Stderr = os.Stderr

		if err := targetCmd.Run(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				os.Exit(exitError.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "❌ Failed to execute command: %v\n", err)
			os.Exit(1)
		}
	},
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

func init() {
	execCmd.Flags().StringVar(&execSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	rootCmd.AddCommand(execCmd)
}

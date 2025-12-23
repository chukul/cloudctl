package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var consoleProfile string
var consoleSecret string
var consoleOpen bool
var consoleRegion string

var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Generate AWS console sign-in URL from stored session",
	Run: func(cmd *cobra.Command, args []string) {

		// Get secret from flag, env, or keychain
		secret, err := internal.GetSecret(consoleSecret)
		if err != nil {
			fmt.Println("❌ Encryption secret required")
			fmt.Println("\n💡 Set the secret:")
			fmt.Println("   export CLOUDCTL_SECRET=\"your-32-char-encryption-key\"")
			return
		}

		if consoleProfile == "" {
			// Load all sessions to filter valid ones
			allSessions, err := internal.ListAllSessions(secret)
			if err != nil {
				fmt.Printf("❌ Failed to list sessions: %v\n", err)
				return
			}

			var validProfiles []string
			for _, s := range allSessions {
				// Filter out expired sessions
				if time.Now().After(s.Expiration) {
					continue
				}
				// Filter out MFA sessions
				if s.RoleArn == "MFA-Session" || s.RoleArn == "" {
					continue
				}
				validProfiles = append(validProfiles, s.Profile)
			}

			if len(validProfiles) == 0 {
				fmt.Println("❌ No valid active sessions found.")
				fmt.Println("💡 Please login or refresh your sessions first.")
				return
			}
			sort.Strings(validProfiles)

			selected, err := ui.SelectProfile("Select Profile", validProfiles)
			if err != nil {
				return
			}
			consoleProfile = selected
		}

		s, err := internal.LoadCredentials(consoleProfile, secret)
		if err != nil {
			fmt.Printf("❌ Failed to load session for profile '%s': %v\n", consoleProfile, err)
			return
		}

		// Check if this is an MFA session (can't be used for console federation)
		// Check if session is expired
		if time.Now().After(s.Expiration) {
			fmt.Printf("❌ Session for profile '%s' has expired.\n", s.Profile)
			fmt.Println("💡 Please refresh or login again:")
			fmt.Printf("   cloudctl refresh --profile %s\n", s.Profile)
			return
		}

		// Check if this is an MFA session (can't be used for console federation)
		// MFA sessions (GetSessionToken) do not have a RoleArn usually, or we marked them specifically.
		// Our internal storage marks them as "MFA-Session".
		if s.RoleArn == "MFA-Session" || s.RoleArn == "" {
			fmt.Println("❌ MFA base sessions cannot be used for console access.")
			fmt.Println("💡 You must assume a role first. Try one of these:")
			fmt.Println("   cloudctl login --source <this-mfa-profile> --profile <new-role-profile> --role <role-arn>")
			return
		}

		// Create session JSON
		sessionJSON := map[string]string{
			"sessionId":    s.AccessKey,
			"sessionKey":   s.SecretKey,
			"sessionToken": s.SessionToken,
		}

		sessionData, _ := json.Marshal(sessionJSON)

		// Get signin token
		fmt.Println("🔐 Getting sign-in token...")
		federationURL := "https://signin.aws.amazon.com/federation"
		params := url.Values{}
		params.Add("Action", "getSigninToken")
		params.Add("Session", string(sessionData))

		resp, err := http.Get(fmt.Sprintf("%s?%s", federationURL, params.Encode()))
		if err != nil {
			fmt.Printf("❌ Failed to get sign-in token: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		var tokenResp map[string]string
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			fmt.Printf("❌ Failed to parse token response: %v\n", err)
			return
		}

		signinToken := tokenResp["SigninToken"]
		if signinToken == "" {
			fmt.Println("❌ Failed to get sign-in token")
			return
		}

		// Build console URL
		destination := "https://console.aws.amazon.com/"
		if consoleRegion != "" {
			destination = fmt.Sprintf("https://%s.console.aws.amazon.com/console/home?region=%s", consoleRegion, consoleRegion)
		}
		consoleURL := fmt.Sprintf("%s?Action=login&Issuer=cloudctl&Destination=%s&SigninToken=%s",
			federationURL, url.QueryEscape(destination), signinToken)

		fmt.Printf("\n✅ Console URL generated for profile '%s'\n", s.Profile)
		fmt.Printf("   Role: %s\n", s.RoleArn)
		fmt.Printf("   Expires: %s\n\n", s.Expiration.Format("2006-01-02 15:04:05"))

		if consoleOpen {
			fmt.Println("🌐 Opening AWS Console in browser...")
			if err := openBrowser(consoleURL); err != nil {
				fmt.Printf("❌ Failed to open browser: %v\n", err)
				fmt.Printf("\nPlease open this URL manually:\n%s\n", consoleURL)
			}
		} else {
			fmt.Printf("Console URL:\n%s\n", consoleURL)
		}
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}
	return cmd.Start()
}

func init() {
	consoleCmd.Flags().StringVar(&consoleProfile, "profile", "", "Profile to generate console URL for")
	consoleCmd.Flags().StringVar(&consoleSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption (or set CLOUDCTL_SECRET env var)")
	consoleCmd.Flags().BoolVar(&consoleOpen, "open", false, "Automatically open URL in browser")
	consoleCmd.Flags().StringVar(&consoleRegion, "region", "ap-southeast-1", "AWS region for console (default: ap-southeast-1)")
	rootCmd.AddCommand(consoleCmd)
}

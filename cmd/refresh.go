package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/chukul/cloudctl/internal"
	"github.com/chukul/cloudctl/internal/ui"
	"github.com/spf13/cobra"
)

var (
	refreshSecret  string
	refreshAll     bool
	refreshProfile string
	forceRefresh   bool
)

var refreshCmd = &cobra.Command{
	Use:   "refresh [profile]",
	Short: "Smart refresh or restore AWS sessions",
	Long: `Automatically refreshes active sessions or restores expired ones by re-using metadata.
If a session is still active, it attempts a silent refresh. If expired or requires MFA, it will prompt for input.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		secret, err := internal.GetSecret(refreshSecret)
		if err != nil {
			fmt.Fprintln(os.Stderr, "❌ Encryption secret required")
			return
		}

		if refreshAll {
			refreshAllSessions(secret)
			return
		}

		profile := refreshProfile
		if profile == "" && len(args) > 0 {
			profile = args[0]
		}

		if profile == "" {
			// Interactive Selection
			allSessions, err := internal.ListAllSessions(secret)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Failed to list sessions: %v\n", err)
				return
			}

			if len(allSessions) == 0 {
				fmt.Fprintln(os.Stderr, "📭 No sessions found.")
				return
			}

			var options []string
			for _, s := range allSessions {
				status := "Expired"
				if time.Now().Before(s.Expiration) {
					status = "Active"
				}
				displayName := fmt.Sprintf("%-15s [%s]", s.Profile, status)
				options = append(options, displayName)
			}
			sort.Strings(options)

			selected, err := ui.SelectProfile("Select Session to Refresh/Restore", options)
			if err != nil {
				return
			}
			fmt.Sscanf(selected, "%s", &profile)
		}

		smartRefresh(profile, secret, forceRefresh)
	},
}

func smartRefresh(profile string, secret string, force bool) {
	s, err := internal.LoadCredentials(profile, secret)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Profile '%s' not found.\n", profile)
		return
	}

	now := time.Now()
	isExpired := s.Expiration.Before(now)

	// 1. Try Silent Refresh if not expired and not forced
	if !isExpired && !force && s.RoleArn != "MFA-Session" && s.SourceProfile != "" {
		fmt.Printf("🔄 Attempting silent refresh for '%s'...\n", profile)
		_, err := internal.PerformRefresh(s, secret, s.Region)
		if err == nil {
			fmt.Printf("✅ Session '%s' refreshed silently.\n", profile)
			return
		}
		fmt.Printf("⚠️  Silent refresh failed: %v. Switching to interactive restore...\n", err)
	}

	// 2. Interactive Restore (Relogin)
	fmt.Printf("🔄 Restoring session '%s'...\n", s.Profile)
	fmt.Printf("   Source: %s\n", s.SourceProfile)
	if s.RoleArn != "MFA-Session" {
		fmt.Printf("   Role:   %s\n", s.RoleArn)
	}
	region := s.Region
	if region == "" {
		region = "ap-southeast-1"
	}

	duration := s.Duration
	if duration < 900 {
		duration = 3600 // Default to 1 hour
	}

	fmt.Printf("   Region: %s\n", region)

	ctx := context.TODO()
	var cfg aws.Config

	// Load Source Config
	sourceSession, sourceErr := internal.LoadCredentials(s.SourceProfile, secret)
	if sourceErr == nil {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				sourceSession.AccessKey,
				sourceSession.SecretKey,
				sourceSession.SessionToken,
			)),
		)
	} else {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithSharedConfigProfile(s.SourceProfile),
		)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load source config: %v\n", err)
		return
	}

	stsClient := sts.NewFromConfig(cfg)
	var newSession *internal.AWSSession

	if s.RoleArn == "MFA-Session" {
		// MFA Session Flow
		tokenCode := readMFACode()
		if tokenCode == "" {
			return
		}

		res, err := ui.Spin("Verifying MFA Token...", func() (any, error) {
			return stsClient.GetSessionToken(ctx, &sts.GetSessionTokenInput{
				DurationSeconds: &duration,
				SerialNumber:    &s.MfaArn,
				TokenCode:       &tokenCode,
			})
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ MFA login failed: %v\n", err)
			return
		}

		result := res.(*sts.GetSessionTokenOutput)
		newSession = &internal.AWSSession{
			Profile:       s.Profile,
			AccessKey:     *result.Credentials.AccessKeyId,
			SecretKey:     *result.Credentials.SecretAccessKey,
			SessionToken:  *result.Credentials.SessionToken,
			Expiration:    *result.Credentials.Expiration,
			RoleArn:       "MFA-Session",
			SourceProfile: s.SourceProfile,
			Region:        region,
			MfaArn:        s.MfaArn,
			Duration:      duration,
		}
	} else {
		// Role Assumption Flow
		input := &sts.AssumeRoleInput{
			RoleArn:         &s.RoleArn,
			RoleSessionName: &s.Profile,
			DurationSeconds: &duration,
		}

		if s.MfaArn != "" {
			tokenCode := readMFACode()
			if tokenCode == "" {
				return
			}
			input.SerialNumber = &s.MfaArn
			input.TokenCode = &tokenCode
		}

		res, err := ui.Spin("Assuming role...", func() (any, error) {
			return stsClient.AssumeRole(ctx, input)
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to assume role: %v\n", err)
			return
		}

		result := res.(*sts.AssumeRoleOutput)
		newSession = &internal.AWSSession{
			Profile:       s.Profile,
			AccessKey:     *result.Credentials.AccessKeyId,
			SecretKey:     *result.Credentials.SecretAccessKey,
			SessionToken:  *result.Credentials.SessionToken,
			Expiration:    *result.Credentials.Expiration,
			RoleArn:       s.RoleArn,
			SourceProfile: s.SourceProfile,
			Region:        region,
			MfaArn:        s.MfaArn,
			Duration:      duration,
		}
	}

	if err := internal.SaveCredentials(s.Profile, newSession, secret); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to save refreshed session: %v\n", err)
		return
	}

	fmt.Printf("\n✅ Session '%s' refreshed/restored successfully!\n", s.Profile)
	fmt.Printf("   Expires: %s\n", internal.FormatBKK(newSession.Expiration))
}

func refreshAllSessions(secret string) {
	fmt.Println("🔄 Intelligent batch refresh starting...")

	sessions, err := internal.ListAllSessions(secret)
	if err != nil {
		fmt.Printf("❌ Failed to load sessions: %v\n", err)
		return
	}

	if len(sessions) == 0 {
		fmt.Println("📭 No sessions found.")
		return
	}

	// 1. Sort sessions to process MFA sessions first (they are often sources)
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].RoleArn == "MFA-Session" && sessions[j].RoleArn != "MFA-Session" {
			return true
		}
		return false
	})

	refreshed := 0
	skipped := 0
	failed := 0
	restoredSources := make(map[string]bool)

	for _, s := range sessions {
		now := time.Now()
		isExpired := now.After(s.Expiration)

		// 2. Handle MFA Sessions (Potential Sources)
		if s.RoleArn == "MFA-Session" {
			if !isExpired {
				fmt.Printf("✅ MFA Session '%s' is still active (%v remaining).\n", s.Profile, time.Until(s.Expiration).Round(time.Minute))
				continue
			}

			// Expired MFA Session - Ask to restore
			fmt.Printf("\n⚠️  MFA Session '%s' has expired.\n", s.Profile)
			fmt.Printf("   Would you like to restore it now? (y/n): ")
			var response string
			fmt.Scanln(&response)
			if response == "y" || response == "Y" {
				smartRefresh(s.Profile, secret, false)
				restoredSources[s.Profile] = true
				refreshed++
			} else {
				fmt.Printf("⏭️  Skipping '%s'.\n", s.Profile)
				skipped++
			}
			continue
		}

		// 3. Handle Role Sessions
		// Try silent refresh first
		_, err := internal.PerformRefresh(s, secret, s.Region)
		if err == nil {
			fmt.Printf("✅ Refreshed '%s' silently.\n", s.Profile)
			refreshed++
			continue
		}

		// If silent refresh failed, check if it's because the source is expired
		if s.SourceProfile != "" && isExpired {
			// If we already restored the source or it's active, and it still fails, it's a real failure.
			// But if we haven't tried to restore the source yet, let's offer it.

			// Check if source is a cloudctl session
			sourceSession, sourceErr := internal.LoadCredentials(s.SourceProfile, secret)
			if sourceErr == nil && now.After(sourceSession.Expiration) {
				if _, alreadyTried := restoredSources[s.SourceProfile]; !alreadyTried {
					fmt.Printf("\n⚠️  Profile '%s' needs source '%s', but it is expired.\n", s.Profile, s.SourceProfile)
					fmt.Printf("   Would you like to restore source '%s'? (y/n): ", s.SourceProfile)
					var response string
					fmt.Scanln(&response)
					if response == "y" || response == "Y" {
						smartRefresh(s.SourceProfile, secret, false)
						restoredSources[s.SourceProfile] = true

						// Retry silent refresh for the role after source is restored
						_, retryErr := internal.PerformRefresh(s, secret, s.Region)
						if retryErr == nil {
							fmt.Printf("✅ Refreshed '%s' after source restore.\n", s.Profile)
							refreshed++
							continue
						}
						fmt.Printf("❌ Failed to refresh '%s' even after source restore: %v\n", s.Profile, retryErr)
						failed++
					} else {
						fmt.Printf("⏭️  Skipping '%s' (source expired).\n", s.Profile)
						skipped++
						restoredSources[s.SourceProfile] = false // Mark as declined
					}
					continue
				}
			}
		}

		// If it's already expired and silent refresh failed (and it's not a source issue we can fix)
		if isExpired {
			fmt.Printf("⏭️  Skipping '%s' (expired, needs manual refresh)\n", s.Profile)
			skipped++
		} else {
			fmt.Printf("❌ Failed to refresh '%s': %v\n", s.Profile, err)
			failed++
		}
	}

	fmt.Printf("\n📊 Summary: %d refreshed/active, %d skipped, %d failed\n", refreshed, skipped, failed)

	if refreshed > 0 {
		fmt.Println("🔄 Automatically syncing sessions to credentials file...")
		syncCount, err := internal.SyncAllToAWS(secret)
		if err != nil {
			fmt.Printf("⚠️  Auto-sync failed: %v\n", err)
		} else {
			fmt.Printf("✅ Synced %d sessions to ~/.aws/credentials\n", syncCount)
		}
	}
}

func init() {
	refreshCmd.Flags().StringVar(&refreshSecret, "secret", os.Getenv("CLOUDCTL_SECRET"), "Secret key for decryption")
	refreshCmd.Flags().BoolVar(&refreshAll, "all", false, "Refresh all active sessions silently")
	refreshCmd.Flags().StringVar(&refreshProfile, "profile", "", "Profile to refresh")
	refreshCmd.Flags().BoolVarP(&forceRefresh, "force", "f", false, "Force interactive re-login even if session is active")
	rootCmd.AddCommand(refreshCmd)
}

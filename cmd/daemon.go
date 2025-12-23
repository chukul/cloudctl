package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var daemonInterval int

const (
	daemonPIDFile = ".cloudctl/daemon.pid"
	daemonLogFile = ".cloudctl/daemon.log"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the background auto-refresh daemon",
	Long: `The daemon automatically refreshes your AWS sessions before they expire.
It runs in the background and checks your sessions every few minutes.`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the auto-refresh daemon",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		pidPath := filepath.Join(home, daemonPIDFile)

		// Check if already running
		if _, err := os.Stat(pidPath); err == nil {
			fmt.Println("❌ Daemon is already running (or pid file exists).")
			fmt.Println("� Use 'cloudctl daemon stop' first if you want to restart.")
			return
		}

		fmt.Printf("�🚀 Starting CloudCtl daemon (Interval: %d minutes)...\n", daemonInterval)
		fmt.Printf("📝 Logs: ~/%s\n", daemonLogFile)

		// In a real production app, we'd use a package like 'sevlyar/go-daemon'
		// but for now, we'll implement a clean loop.
		// If user wants it in background, they can use 'cloudctl daemon start &'
		// or we can implement a self-forking logic later.

		startDaemonLoop(daemonInterval)
	},
}

func startDaemonLoop(intervalMins int) {
	home, _ := os.UserHomeDir()
	pidPath := filepath.Join(home, daemonPIDFile)
	logPath := filepath.Join(home, daemonLogFile)

	// Create PID file
	os.MkdirAll(filepath.Dir(pidPath), 0700)
	os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", os.Getpid())), 0600)
	defer os.Remove(pidPath)

	// Setup logging
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	defer logFile.Close()

	fmt.Fprintf(logFile, "[%s] Daemon started\n", time.Now().Format(time.RFC3339))

	ticker := time.NewTicker(time.Duration(intervalMins) * time.Minute)
	defer ticker.Stop()

	for {
		// Run refresh check
		runRefreshCheck(logFile)

		<-ticker.C
	}
}

func runRefreshCheck(logWriter *os.File) {
	fmt.Fprintf(logWriter, "[%s] Checking sessions...\n", time.Now().Format(time.RFC3339))

	secret, err := internal.GetSecret("")
	if err != nil {
		fmt.Fprintf(logWriter, "[%s] Error: encryption secret required\n", time.Now().Format(time.RFC3339))
		return
	}

	sessions, err := internal.ListAllSessions(secret)
	if err != nil {
		fmt.Fprintf(logWriter, "[%s] Error: failed to list sessions: %v\n", time.Now().Format(time.RFC3339), err)
		return
	}

	now := time.Now()
	for _, s := range sessions {
		// Refresh if expiring in less than 15 minutes
		if time.Until(s.Expiration) < 15*time.Minute {
			// Skip if already expired (better to relogin manually)
			if now.After(s.Expiration) {
				continue
			}

			fmt.Fprintf(logWriter, "[%s] Refreshing profile '%s' (expires in %v)...\n",
				time.Now().Format(time.RFC3339), s.Profile, time.Until(s.Expiration).Round(time.Second))

			// Use ap-southeast-1 as default if none specified (or improve this later)
			_, err := internal.PerformRefresh(s, secret, "ap-southeast-1")
			if err != nil {
				fmt.Fprintf(logWriter, "[%s] Failed to refresh '%s': %v\n", time.Now().Format(time.RFC3339), s.Profile, err)
			} else {
				fmt.Fprintf(logWriter, "[%s] Successfully refreshed '%s'\n", time.Now().Format(time.RFC3339), s.Profile)
			}
		}
	}
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the background daemon",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		pidPath := filepath.Join(home, daemonPIDFile)

		data, err := os.ReadFile(pidPath)
		if err != nil {
			fmt.Println("❌ Daemon is not running.")
			return
		}

		var pid int
		fmt.Sscanf(string(data), "%d", &pid)

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("❌ Could not find process %d\n", pid)
			os.Remove(pidPath)
			return
		}

		fmt.Printf("🛑 Stopping CloudCtl daemon (PID: %d)...\n", pid)
		process.Signal(os.Interrupt)
		os.Remove(pidPath)
		fmt.Println("✅ Daemon stopped.")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check daemon status",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		pidPath := filepath.Join(home, daemonPIDFile)

		if _, err := os.Stat(pidPath); err != nil {
			fmt.Println("⚪ Daemon is NOT running.")
			return
		}

		data, _ := os.ReadFile(pidPath)
		fmt.Printf("🟢 Daemon is running (PID: %s)\n", string(data))
	},
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View daemon logs",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		logPath := filepath.Join(home, daemonLogFile)

		data, err := os.ReadFile(logPath)
		if err != nil {
			fmt.Println("❌ No logs found.")
			return
		}

		fmt.Println(string(data))
	},
}

var daemonSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup automatic startup on macOS",
	Run: func(cmd *cobra.Command, args []string) {
		if runtime.GOOS != "darwin" {
			fmt.Println("❌ Setup is only supported on macOS.")
			return
		}

		home, _ := os.UserHomeDir()
		execPath, _ := os.Executable()
		plistPath := filepath.Join(home, "Library/LaunchAgents/com.chukul.cloudctl.plist")

		plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.chukul.cloudctl</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>daemon</string>
        <string>start</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>%s/.cloudctl/daemon.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>%s/.cloudctl/daemon.stderr.log</string>
</dict>
</plist>`, execPath, home, home)

		os.MkdirAll(filepath.Dir(plistPath), 0755)
		err := os.WriteFile(plistPath, []byte(plistContent), 0644)
		if err != nil {
			fmt.Printf("❌ Failed to create plist: %v\n", err)
			return
		}

		fmt.Println("✅ LaunchAgent plist created.")
		fmt.Println("🚀 To enable, run:")
		fmt.Printf("   launchctl load %s\n", plistPath)
	},
}

func init() {
	daemonStartCmd.Flags().IntVarP(&daemonInterval, "interval", "i", 5, "Check interval in minutes")

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonSetupCmd)

	rootCmd.AddCommand(daemonCmd)
}

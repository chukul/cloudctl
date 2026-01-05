package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/chukul/cloudctl/internal"
	"github.com/spf13/cobra"
)

var (
	daemonInterval   int
	daemonForeground bool
)

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
			fmt.Println("💡 Use 'cloudctl daemon stop' first if you want to restart.")
			return
		}

		if daemonForeground {
			fmt.Printf("🚀 Starting CloudCtl daemon in foreground (Interval: %d minutes)...\n", daemonInterval)
			startDaemonLoop(daemonInterval)
			return
		}

		// Self-forking logic
		execPath, _ := os.Executable()
		bgCmd := exec.Command(execPath, "daemon", "start", "--foreground", "--interval", fmt.Sprintf("%d", daemonInterval))

		// Redirect output to log files for the background process
		logDir := filepath.Join(home, ".cloudctl")
		os.MkdirAll(logDir, 0700)

		stdoutFile, _ := os.OpenFile(filepath.Join(logDir, "daemon.stdout.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		stderrFile, _ := os.OpenFile(filepath.Join(logDir, "daemon.stderr.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)

		bgCmd.Stdout = stdoutFile
		bgCmd.Stderr = stderrFile

		err := bgCmd.Start()
		if err != nil {
			fmt.Printf("❌ Failed to start daemon in background: %v\n", err)
			return
		}

		fmt.Printf("🚀 CloudCtl daemon started in background (PID: %d)\n", bgCmd.Process.Pid)
		fmt.Printf("📝 Logs: ~/%s\n", daemonLogFile)
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
	currentDay := time.Now().YearDay()
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Printf("❌ Failed to open log file: %v\n", err)
		return
	}

	fmt.Fprintf(logFile, "[%s] 🚀 [Daemon] Started (Interval: %d mins)\n", time.Now().Format("15:04:05"), intervalMins)

	ticker := time.NewTicker(time.Duration(intervalMins) * time.Minute)
	defer ticker.Stop()

	for {
		// Log Rotation: If day has changed, truncate the log file
		now := time.Now()
		if now.YearDay() != currentDay {
			logFile.Close()
			// Open with O_TRUNC to clear previous day logs
			logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Printf("❌ Failed to rotate log file: %v\n", err)
				return
			}
			currentDay = now.YearDay()
			fmt.Fprintf(logFile, "[%s] 🔄 [Daemon] Log rotated (new day started)\n", now.Format("15:04:05"))
		}

		// Run refresh check
		runRefreshCheck(logFile)

		<-ticker.C
	}
}

func runRefreshCheck(logWriter *os.File) {
	secret, err := internal.GetSecret("")
	if err != nil {
		fmt.Fprintf(logWriter, "[%s] ❌ [Daemon] Error: encryption secret required\n", time.Now().Format("15:04:05"))
		return
	}

	sessions, err := internal.ListAllSessions(secret)
	if err != nil {
		fmt.Fprintf(logWriter, "[%s] ❌ [Daemon] Error: failed to list sessions: %v\n", time.Now().Format("15:04:05"), err)
		return
	}

	fmt.Fprintf(logWriter, "[%s] 🔍 [Daemon] Checking %d sessions...\n", time.Now().Format("15:04:05"), len(sessions))

	now := time.Now()
	actionTaken := false
	for _, s := range sessions {
		// 1. Skip sessions that are not near expiration (> 15 mins)
		if time.Until(s.Expiration) >= 15*time.Minute {
			continue
		}

		// 2. Skip sessions that are already expired (the user must relogin manually)
		if now.After(s.Expiration) {
			continue
		}

		// 3. Skip sessions that cannot be silently refreshed (MFA sessions)
		if s.RoleArn == "MFA-Session" {
			// Silently skip MFA sessions to avoid log noise
			continue
		}

		// 4. Skip sessions with no source
		if s.SourceProfile == "" {
			continue
		}

		// 5. Attempt Refresh
		fmt.Fprintf(logWriter, "[%s] 🔄 [%s] Expiring in %v, starting silent refresh...\n",
			now.Format("15:04:05"), s.Profile, time.Until(s.Expiration).Round(time.Second))

		refreshRegion := s.Region
		if refreshRegion == "" {
			refreshRegion = "ap-southeast-1"
		}

		refreshStart := time.Now()
		_, err := internal.PerformRefresh(s, secret, refreshRegion)
		duration := time.Since(refreshStart).Round(10 * time.Millisecond)

		if err != nil {
			fmt.Fprintf(logWriter, "[%s] ❌ [%s] Refresh failed: %v\n", time.Now().Format("15:04:05"), s.Profile, err)
		} else {
			fmt.Fprintf(logWriter, "[%s] ✅ [%s] Successfully refreshed (took %v)\n", time.Now().Format("15:04:05"), s.Profile, duration)
		}
		actionTaken = true
	}

	if !actionTaken {
		fmt.Fprintf(logWriter, "[%s] 🟢 [Daemon] All sessions healthy. Next check in 5m.\n", time.Now().Format("15:04:05"))
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
        <string>--foreground</string>
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
	daemonStartCmd.Flags().BoolVarP(&daemonForeground, "foreground", "f", false, "Run in foreground")

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonSetupCmd)

	rootCmd.AddCommand(daemonCmd)
}

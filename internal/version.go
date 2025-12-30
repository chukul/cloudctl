package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	CurrentVersion = "v1.1.0" // Will be overwritten by ldflags during build
	GitHubAPI      = "https://api.github.com/repos/chukul/cloudctl/releases/latest"
	CheckInterval  = 24 * time.Hour
)

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type VersionCheck struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
}

// CheckForUpdates checks if a new version is available (non-blocking)
func CheckForUpdates() {
	// Check if we should skip (checked recently)
	if !shouldCheck() {
		return
	}

	go func() {
		latest, url, err := FetchLatestVersion()
		if err != nil {
			return // Silently fail
		}

		if IsNewer(latest, CurrentVersion) {
			fmt.Fprintf(os.Stderr, "\n💡 Update available: %s → %s\n", CurrentVersion, latest)
			fmt.Fprintf(os.Stderr, "   Download: %s\n\n", url)
		}

		saveLastCheck(latest)
	}()
}

func shouldCheck() bool {
	cachePath := filepath.Join(os.Getenv("HOME"), ".cloudctl", "version_check.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return true
	}

	var check VersionCheck
	if err := json.Unmarshal(data, &check); err != nil {
		return true
	}

	return time.Since(check.LastChecked) > CheckInterval
}

func FetchLatestVersion() (string, string, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(GitHubAPI)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("status %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", "", err
	}

	return release.TagName, release.HTMLURL, nil
}

func IsNewer(latest, current string) bool {
	// Simple version comparison (assumes semantic versioning)
	latest = strings.TrimPrefix(latest, "v")
	current = strings.TrimPrefix(current, "v")
	return latest > current
}

func saveLastCheck(version string) {
	cachePath := filepath.Join(os.Getenv("HOME"), ".cloudctl", "version_check.json")
	check := VersionCheck{
		LastChecked:   time.Now(),
		LatestVersion: version,
	}
	data, _ := json.Marshal(check)
	os.WriteFile(cachePath, data, 0600)
}

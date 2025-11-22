package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
)

const (
	GitHubRepoOwner = "lokesh-balla"
	GitHubRepoName  = "gvm"
)

// updateCmd represents the update command
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates gvm to the latest version",
	Long:  `Updates gvm to the latest version available on GitHub Releases.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return updateGVM()
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"` // Format: "sha256:..."
	} `json:"assets"`
}

func updateGVM() error {
	// 1. Get current version
	currentVerStr := strings.Split(rootCmd.Version, " ")[0]
	currentVer, err := version.NewVersion(currentVerStr)
	if err != nil {
		// If we can't parse current version (e.g. dev build), assume we want to update if user asks
		fmt.Printf("Warning: Could not parse current version '%s': %v\n", currentVerStr, err)
	}

	fmt.Println("Checking for updates...")

	// 2. Fetch latest release info
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubRepoOwner, GitHubRepoName))
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to check for updates: status code %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to decode release info: %w", err)
	}

	latestVer, err := version.NewVersion(release.TagName)
	if err != nil {
		return fmt.Errorf("failed to parse latest version '%s': %w", release.TagName, err)
	}

	if currentVer != nil && currentVer.GreaterThanOrEqual(latestVer) {
		fmt.Printf("You are already using the latest version: %s\n", release.TagName)
		return nil
	}

	fmt.Printf("New version available: %s (current: %s)\n", release.TagName, currentVerStr)
	fmt.Println("Downloading update...")

	// 3. Find asset for current OS/Arch
	assetName := fmt.Sprintf("gvm_%s_%s", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	var expectedDigest string
	for _, asset := range release.Assets {
		if asset.Name == assetName {
			downloadURL = asset.BrowserDownloadURL
			expectedDigest = asset.Digest
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// 4. Download new binary
	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download update: status code %d", resp.StatusCode)
	}

	// 5. Prepare for atomic swap
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exeDir := filepath.Dir(exePath)

	// Create a temp file in the SAME DIRECTORY as the executable to ensure atomic rename
	tmpFile, err := os.CreateTemp(exeDir, ".gvm-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Cleanup if something fails

	// Use TeeReader to calculate hash while downloading
	hasher := sha256.New()
	tee := io.TeeReader(resp.Body, hasher)

	_, err = io.Copy(tmpFile, tee)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update to temp file: %w", err)
	}
	tmpFile.Close()

	// 6. Verify Checksum
	if expectedDigest != "" {
		fmt.Println("Verifying checksum...")
		calculatedDigest := hex.EncodeToString(hasher.Sum(nil))
		// expectedDigest format is usually "sha256:..." or just hex
		if after, ok := strings.CutPrefix(expectedDigest, "sha256:"); ok {
			expectedDigest = after
		}

		if calculatedDigest != expectedDigest {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedDigest, calculatedDigest)
		}
		fmt.Println("Checksum verified.")
	} else {
		fmt.Println("Warning: No checksum provided in release info, skipping verification.")
	}

	// Make it executable
	if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
		return fmt.Errorf("failed to make update executable: %w", err)
	}

	// 7. Perform the swap
	// On POSIX, os.Rename is atomic if on the same filesystem.
	// On Windows, we still need the rename dance, but we can try direct rename first.
	if runtime.GOOS == "windows" {
		oldPath := exePath + ".old"
		_ = os.Remove(oldPath) // Ensure no previous .old exists

		if err := os.Rename(exePath, oldPath); err != nil {
			return fmt.Errorf("failed to rename current binary: %w", err)
		}

		if err := os.Rename(tmpFile.Name(), exePath); err != nil {
			// Try to restore
			_ = os.Rename(oldPath, exePath)
			return fmt.Errorf("failed to replace binary: %w", err)
		}
		_ = os.Remove(oldPath)
	} else {
		if err := os.Rename(tmpFile.Name(), exePath); err != nil {
			return fmt.Errorf("failed to replace binary: %w", err)
		}
	}

	fmt.Printf("Successfully updated to %s\n", release.TagName)
	return nil
}

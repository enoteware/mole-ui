package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const (
	githubRepo   = "enoteware/mole-ui"
	githubAPIURL = "https://api.github.com/repos/" + githubRepo + "/releases/latest"
	versionFile  = "VERSION"
)

type UpdateInfo struct {
	CurrentVersion  string       `json:"current_version"`
	LatestVersion   string       `json:"latest_version"`
	UpdateAvailable bool         `json:"update_available"`
	DownloadURL     string       `json:"download_url"`
	ReleaseNotes    string       `json:"release_notes"`
	PublishedAt     string       `json:"published_at"`
	SystemUpdates   []UpdateItem `json:"system_updates,omitempty"`
}

type UpdateItem struct {
	Name    string `json:"name"`
	Label   string `json:"label"`
	Details string `json:"details"`
}

type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getCurrentVersion() string {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}

func handleCheckUpdates(w http.ResponseWriter, r *http.Request) {
	updateInfo, err := checkForUpdates()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateInfo)
}

func checkForUpdates() (*UpdateInfo, error) {
	currentVersion := getCurrentVersion()

	// Fetch latest release from GitHub
	resp, err := http.Get(githubAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch updates: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No releases yet
		return &UpdateInfo{
			CurrentVersion:  currentVersion,
			LatestVersion:   currentVersion,
			UpdateAvailable: false,
		}, nil
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	// Clean version tags (remove 'v' prefix if present)
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	// Find DMG asset
	downloadURL := ""
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".dmg") {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	updateInfo := &UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latestVersion,
		UpdateAvailable: compareVersions(latestVersion, current) > 0,
		DownloadURL:     downloadURL,
		ReleaseNotes:    release.Body,
		PublishedAt:     release.PublishedAt,
	}

	// Add Homebrew checks
	if _, err := exec.LookPath("brew"); err == nil {
		cmd := exec.Command("brew", "outdated", "--quiet")
		if out, err := cmd.Output(); err == nil && len(out) > 0 {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			updateInfo.SystemUpdates = append(updateInfo.SystemUpdates, UpdateItem{
				Name:    "Homebrew",
				Label:   fmt.Sprintf("Homebrew (%d updates available)", len(lines)),
				Details: string(out),
			})
		}
	}

	return updateInfo, nil
}

func handlePerformUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	moleScript := findMoleScript()
	if moleScript == "" {
		http.Error(w, "Mole CLI not found", http.StatusInternalServerError)
		return
	}

	var result CleanResult
	writeLog("Performing update: %s", req.Name)
	if req.Name == "Mole" {
		result = runMoleCommand("update", "--debug")
	} else if req.Name == "Homebrew" {
		// Homebrew upgrade
		writeLog("Executing: brew upgrade")
		cmd := exec.Command("brew", "upgrade")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		result = CleanResult{
			Success: err == nil,
			Message: func() string {
				if err != nil {
					return err.Error()
				}
				return "Homebrew upgraded successfully"
			}(),
			Output: stripANSI(out.String()),
		}
	} else {
		writeLog("ERROR: Unsupported update type: %s", req.Name)
		result = CleanResult{Success: false, Message: "Unsupported update type"}
	}
	
	writeLog("Update result for %s: Success=%v, Message=%s", req.Name, result.Success, result.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// compareVersions returns:
// -1 if v1 < v2
//
//	0 if v1 == v2
//	1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// Simple string comparison for semantic versions
	// For proper semver, use a library like github.com/Masterminds/semver
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			fmt.Sscanf(parts1[i], "%d", &p1)
		}
		if i < len(parts2) {
			fmt.Sscanf(parts2[i], "%d", &p2)
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}

	return 0
}

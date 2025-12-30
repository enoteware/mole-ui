package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type PermissionsStatus struct {
	FullDiskAccess   bool   `json:"full_disk_access"`
	CanReadHome      bool   `json:"can_read_home"`
	CanReadLibrary   bool   `json:"can_read_library"`
	CanReadDownloads bool   `json:"can_read_downloads"`
	Message          string `json:"message"`
}

func handlePermissionsCheck(w http.ResponseWriter, r *http.Request) {
	status := checkPermissions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func checkPermissions() PermissionsStatus {
	status := PermissionsStatus{
		FullDiskAccess:   false,
		CanReadHome:      true,
		CanReadLibrary:   false,
		CanReadDownloads: true,
	}

	home := os.Getenv("HOME")

	// Check if we can read common protected directories
	// Full Disk Access is needed to read certain system directories
	testPaths := []struct {
		path  string
		field *bool
		name  string
	}{
		{filepath.Join(home, "Library", "Safari"), &status.CanReadLibrary, "Library"},
		{filepath.Join(home, "Downloads"), &status.CanReadDownloads, "Downloads"},
		{filepath.Join(home), &status.CanReadHome, "Home"},
	}

	for _, test := range testPaths {
		if _, err := os.ReadDir(test.path); err == nil {
			*test.field = true
		}
	}

	// If we can read Library/Safari, we likely have Full Disk Access
	// Safari bookmarks are a good test for Full Disk Access
	safariBookmarks := filepath.Join(home, "Library", "Safari", "Bookmarks.plist")
	if _, err := os.Stat(safariBookmarks); err == nil {
		status.FullDiskAccess = true
		status.Message = "Full Disk Access granted - all features available"
	} else {
		// Alternative check: try to read TCC database (requires Full Disk Access)
		tccPath := filepath.Join(home, "Library", "Application Support", "com.apple.TCC", "TCC.db")
		if _, err := os.Stat(tccPath); err == nil {
			status.FullDiskAccess = true
			status.Message = "Full Disk Access granted - all features available"
		} else {
			status.Message = "Full Disk Access required for complete system scanning"
		}
	}

	return status
}

func handleOpenSystemSettings(w http.ResponseWriter, r *http.Request) {
	// Open System Settings to Privacy & Security > Full Disk Access
	// On macOS Ventura+, this opens the Privacy & Security pane
	// The user will need to manually navigate to Full Disk Access
	cmd := exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles")
	err := cmd.Run()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

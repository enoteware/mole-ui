package main

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Volume struct {
	Device      string  `json:"device"`
	MountPoint  string  `json:"mount_point"`
	TotalBytes  int64   `json:"total_bytes"`
	UsedBytes   int64   `json:"used_bytes"`
	FreeBytes   int64   `json:"free_bytes"`
	UsedPercent float64 `json:"used_percent"`
	TotalHuman  string  `json:"total_human"`
	UsedHuman   string  `json:"used_human"`
	FreeHuman   string  `json:"free_human"`
	IsMain      bool    `json:"is_main"`
}

type VolumeAnalysis struct {
	Path       string            `json:"path"`
	TotalSize  int64             `json:"total_size"`
	Categories []StorageCategory `json:"categories"`
}

func handleListVolumes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumes, err := getVolumes()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(volumes)
}

func handleAnalyzeVolume(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	volumePath := r.URL.Query().Get("path")
	if volumePath == "" {
		volumePath = "/"
	}

	analysis := analyzeVolumePath(volumePath)
	json.NewEncoder(w).Encode(analysis)
}

func getVolumes() ([]Volume, error) {
	cmd := exec.Command("df", "-k")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var volumes []Volume
	lines := strings.Split(string(output), "\n")

	for _, line := range lines[1:] { // Skip header
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Only show /dev/ devices and useful mount points
		device := fields[0]
		if !strings.HasPrefix(device, "/dev/disk") {
			continue
		}

		mountPoint := fields[len(fields)-1]

		// Skip system volumes we don't want to show
		if strings.Contains(mountPoint, "/Update") ||
			strings.Contains(mountPoint, "/xarts") ||
			strings.Contains(mountPoint, "/iSCPreboot") ||
			strings.Contains(mountPoint, "/Hardware") ||
			strings.Contains(mountPoint, "/Preboot") ||
			strings.Contains(mountPoint, "/VM") {
			continue
		}

		totalKB, _ := strconv.ParseInt(fields[1], 10, 64)
		usedKB, _ := strconv.ParseInt(fields[2], 10, 64)
		freeKB, _ := strconv.ParseInt(fields[3], 10, 64)

		totalBytes := totalKB * 1024
		usedBytes := usedKB * 1024
		freeBytes := freeKB * 1024

		usedPercent := 0.0
		if totalBytes > 0 {
			usedPercent = float64(usedBytes) / float64(totalBytes) * 100
		}

		isMain := mountPoint == "/" || mountPoint == "/System/Volumes/Data"

		volumes = append(volumes, Volume{
			Device:      device,
			MountPoint:  mountPoint,
			TotalBytes:  totalBytes,
			UsedBytes:   usedBytes,
			FreeBytes:   freeBytes,
			UsedPercent: usedPercent,
			TotalHuman:  formatBytes(totalBytes),
			UsedHuman:   formatBytes(usedBytes),
			FreeHuman:   formatBytes(freeBytes),
			IsMain:      isMain,
		})
	}

	return volumes, nil
}

func analyzeVolumePath(volumePath string) VolumeAnalysis {
	var categories []StorageCategory

	// Get top-level directories
	entries, err := filepath.Glob(filepath.Join(volumePath, "*"))
	if err != nil {
		return VolumeAnalysis{Path: volumePath}
	}

	var totalSize int64

	for _, entry := range entries {
		name := filepath.Base(entry)

		// Skip hidden files and system directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		size := getDirSize(entry)
		if size > 100*1024*1024 { // Only show > 100MB
			totalSize += size

			// Assign colors and icons based on name/type
			color, icon := getCategoryStyle(name)

			categories = append(categories, StorageCategory{
				Name:      name,
				Size:      size,
				SizeHuman: formatBytes(size),
				Percent:   0, // Will calculate after
				Color:     color,
				Icon:      icon,
			})
		}
	}

	// Calculate percentages
	for i := range categories {
		if totalSize > 0 {
			categories[i].Percent = float64(categories[i].Size) / float64(totalSize) * 100
		}
	}

	// Sort by size descending
	for i := 0; i < len(categories)-1; i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[j].Size > categories[i].Size {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	return VolumeAnalysis{
		Path:       volumePath,
		TotalSize:  totalSize,
		Categories: categories,
	}
}

func getCategoryStyle(name string) (color, icon string) {
	name = strings.ToLower(name)

	// Match common patterns - icon names correspond to MoleIcons in JavaScript
	switch {
	case strings.Contains(name, "docker"):
		return "#2563eb", "docker"
	case strings.Contains(name, "media") || strings.Contains(name, "movies") || strings.Contains(name, "videos"):
		return "#dc2626", "video"
	case strings.Contains(name, "photo") || strings.Contains(name, "pictures"):
		return "#ec4899", "camera"
	case strings.Contains(name, "music") || strings.Contains(name, "audio"):
		return "#06b6d4", "music"
	case strings.Contains(name, "code") || strings.Contains(name, "dev") || strings.Contains(name, "projects"):
		return "#22c55e", "code"
	case strings.Contains(name, "document"):
		return "#10b981", "document"
	case strings.Contains(name, "download"):
		return "#f59e0b", "download"
	case strings.Contains(name, "library"):
		return "#6366f1", "library"
	case strings.Contains(name, "application"):
		return "#3b82f6", "apps"
	case strings.Contains(name, "backup") || strings.Contains(name, "time machine"):
		return "#8b5cf6", "backup"
	default:
		return "#71717a", "folder"
	}
}

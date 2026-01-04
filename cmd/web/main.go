package main

import (
	"bufio"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

var (
	moleDir      string
	Version      = "dev"
	port         = flag.Int("port", 8080, "Port to run the server on")
	hostAddr     = flag.String("host", "", "Host to bind to (default: localhost, use 0.0.0.0 for all interfaces)")
	openBrowser  = flag.Bool("open", true, "Open browser on start")
	authUser     string
	authPass     string
	logBroadcast = make(chan string, 100)
)

func init() {
	// Auth from environment
	authUser = os.Getenv("MOLE_AUTH_USER")
	authPass = os.Getenv("MOLE_AUTH_PASS")

	// Find the mole directory
	if envDir := os.Getenv("MOLE_DIR"); envDir != "" {
		moleDir = envDir
	} else {
		exe, err := os.Executable()
		if err == nil {
			moleDir = filepath.Dir(filepath.Dir(filepath.Dir(exe)))
		}
		if moleDir == "" || !fileExists(filepath.Join(moleDir, "mole")) {
			wd, _ := os.Getwd()
			moleDir = filepath.Dir(filepath.Dir(wd))
			if !fileExists(filepath.Join(moleDir, "mole")) {
				moleDir = filepath.Dir(wd)
			}
			if !fileExists(filepath.Join(moleDir, "mole")) {
				moleDir = wd
			}
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Basic auth middleware
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if not configured
		if authUser == "" || authPass == "" {
			next(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Mole"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if subtle.ConstantTimeCompare([]byte(pair[0]), []byte(authUser)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pair[1]), []byte(authPass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Mole"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// Wrapper for http.Handler (used for static files)
func basicAuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		basicAuth(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})(w, r)
	})
}

func main() {
	flag.Parse()

	// Override from env if set
	if envPort := os.Getenv("MOLE_PORT"); envPort != "" {
		fmt.Sscanf(envPort, "%d", port)
	}
	if envHost := os.Getenv("MOLE_HOST"); envHost != "" {
		*hostAddr = envHost
	}
	if os.Getenv("MOLE_NO_OPEN") != "" {
		*openBrowser = false
	}

	// Templates
	tmpl := template.Must(template.ParseFS(templateFiles, "templates/*.html"))

	// Static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	http.Handle("/static/", http.StripPrefix("/static/", basicAuthHandler(http.FileServer(http.FS(staticFS)))))

	// Page routes
	http.HandleFunc("/", basicAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		// Add cache-busting headers to prevent WKWebView caching
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		tmpl.ExecuteTemplate(w, "index.html", nil)
	}))

	// Health check (no auth)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": Version})
	})

	// Installer script endpoint (no auth)
	http.HandleFunc("/install.sh", handleInstallScript)

	// API routes (all protected)
	http.HandleFunc("/api/status", basicAuth(handleStatus))
	http.HandleFunc("/api/clean", basicAuth(handleClean))
	http.HandleFunc("/api/clean/preview", basicAuth(handleCleanPreview))
	http.HandleFunc("/api/uninstall/apps", basicAuth(handleListApps))
	http.HandleFunc("/api/uninstall", basicAuth(handleUninstall))
	http.HandleFunc("/api/app/icon", handleAppIcon) // No auth needed for icons
	http.HandleFunc("/api/analyze", basicAuth(handleAnalyze))
	http.HandleFunc("/api/analyze/large", basicAuth(handleAnalyzeLarge))
	http.HandleFunc("/api/analyze/downloads", basicAuth(handleAnalyzeDownloads))
	http.HandleFunc("/api/storage/breakdown", basicAuth(handleStorageBreakdown))
	http.HandleFunc("/api/storage/analyze-other", basicAuth(handleAnalyzeOther))
	http.HandleFunc("/api/volumes", basicAuth(handleListVolumes))
	http.HandleFunc("/api/volumes/analyze", basicAuth(handleAnalyzeVolume))
	http.HandleFunc("/api/open-finder", basicAuth(handleOpenFinder))
	http.HandleFunc("/api/permissions/check", basicAuth(handlePermissionsCheck))
	http.HandleFunc("/api/permissions/open-settings", basicAuth(handleOpenSystemSettings))
	http.HandleFunc("/api/permissions/admin-test", basicAuth(handlePermissionsAdminTest))
	http.HandleFunc("/api/logs/open", basicAuth(handleOpenLogs))
	http.HandleFunc("/api/logs/bundle", basicAuth(handleLogsBundle))
	http.HandleFunc("/api/updates/check", basicAuth(handleCheckUpdates))
	http.HandleFunc("/api/updates/perform", basicAuth(handlePerformUpdate))
	http.HandleFunc("/api/optimize", basicAuth(handleOptimize))
	http.HandleFunc("/api/debug/logs", basicAuth(handleDebugLogs))
	http.HandleFunc("/api/purge", basicAuth(handlePurge))
	http.HandleFunc("/api/purge/scan", basicAuth(handlePurgeScan))
	http.HandleFunc("/api/status/stream", basicAuth(handleStatusStream))
	http.HandleFunc("/api/logs", basicAuth(handleLogsStream))
	http.HandleFunc("/api/files", basicAuth(handleDeleteFiles))
	http.HandleFunc("/api/apps/unsupported", basicAuth(handleUnsupportedApps))

	// Determine bind address
	bindHost := *hostAddr
	if bindHost == "" {
		bindHost = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", bindHost, *port)

	// Display URL (for user)
	displayHost := bindHost
	if bindHost == "0.0.0.0" {
		displayHost = getLocalIP()
	}
	url := fmt.Sprintf("http://%s:%d", displayHost, *port)

	fmt.Printf("\n  🐭 Mole Web UI\n")
	fmt.Printf("  ─────────────────────────────\n")
	fmt.Printf("  Server:  %s\n", url)
	fmt.Printf("  Bind:    %s\n", addr)
	fmt.Printf("  Mole:    %s\n", moleDir)
	if authUser != "" {
		fmt.Printf("  Auth:    enabled (user: %s)\n", authUser)
	} else {
		fmt.Printf("  Auth:    disabled\n")
	}
	fmt.Printf("  ─────────────────────────────\n\n")

	if *openBrowser && bindHost == "localhost" {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openURL(fmt.Sprintf("http://localhost:%d", *port))
		}()
	}

	log.Fatal(http.ListenAndServe(addr, nil))
}

func getLocalIP() string {
	addrs, err := net.Interfaces()
	if err != nil {
		return "localhost"
	}
	for _, iface := range addrs {
		if iface.Name == "en0" || iface.Name == "eth0" {
			for _, addr := range iface.Addrs {
				if strings.Contains(addr.Addr, ".") && !strings.HasPrefix(addr.Addr, "127.") {
					return strings.Split(addr.Addr, "/")[0]
				}
			}
		}
	}
	return "localhost"
}

func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	cmd.Start()
}

// Status API - returns system metrics as JSON
type SystemStatus struct {
	Hostname    string      `json:"hostname"`
	HomeDir     string      `json:"home_dir"`
	LocalIP     string      `json:"local_ip"`
	OS          string      `json:"os"`
	Uptime      string      `json:"uptime"`
	Version     string      `json:"version"`
	CPU         CPUInfo     `json:"cpu"`
	Memory      MemInfo     `json:"memory"`
	Disk        DiskInfo    `json:"disk"`
	Network     NetworkInfo `json:"network"`
	CollectedAt time.Time   `json:"collected_at"`
}

type CPUInfo struct {
	Model string  `json:"model"`
	Cores int     `json:"cores"`
	Usage float64 `json:"usage"`
}

type MemInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Percent   float64 `json:"percent"`
}

type DiskInfo struct {
	Total   uint64  `json:"total"`
	Used    uint64  `json:"used"`
	Free    uint64  `json:"free"`
	Percent float64 `json:"percent"`
}

type NetworkInfo struct {
	BytesSent uint64 `json:"bytes_sent"`
	BytesRecv uint64 `json:"bytes_recv"`
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	status := collectStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func collectStatus() SystemStatus {
	status := SystemStatus{
		CollectedAt: time.Now(),
		Version:     getCurrentVersion(),
		HomeDir:     os.Getenv("HOME"),
	}

	if info, err := host.Info(); err == nil {
		status.Hostname = info.Hostname
		status.OS = fmt.Sprintf("%s %s", info.Platform, info.PlatformVersion)
		status.Uptime = formatUptime(info.Uptime)
	}

	status.LocalIP = getLocalIP()

	if cpus, err := cpu.Info(); err == nil && len(cpus) > 0 {
		status.CPU.Model = cpus[0].ModelName
	}
	status.CPU.Cores = runtime.NumCPU()
	if usage, err := cpu.Percent(0, false); err == nil && len(usage) > 0 {
		status.CPU.Usage = usage[0]
	}

	if m, err := mem.VirtualMemory(); err == nil {
		status.Memory.Total = m.Total
		status.Memory.Used = m.Used
		status.Memory.Available = m.Available
		status.Memory.Percent = m.UsedPercent
	}

	if d, err := disk.Usage("/"); err == nil {
		status.Disk.Total = d.Total
		status.Disk.Used = d.Used
		status.Disk.Free = d.Free
		status.Disk.Percent = d.UsedPercent
	}

	if n, err := net.IOCounters(false); err == nil && len(n) > 0 {
		status.Network.BytesSent = n[0].BytesSent
		status.Network.BytesRecv = n[0].BytesRecv
	}

	return status
}

func formatUptime(seconds uint64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	mins := (seconds % 3600) / 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func handleStatusStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(2 * time.Second) // Slow down status updates slightly
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			status := collectStatus()
			data, _ := json.Marshal(status)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func handleLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "data: Connected to log stream\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-logBroadcast:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			flusher.Flush()
		}
	}
}

type CleanResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Cleaned int64  `json:"cleaned_bytes"`
	Output  string `json:"output,omitempty"`
}

func handleCleanPreview(w http.ResponseWriter, r *http.Request) {
	result := runMoleCommand("clean", "--dry-run")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func handleClean(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	category := r.URL.Query().Get("category")

	// Handle trash emptying separately since mole CLI doesn't support it
	if category == "trash" {
		result := emptyTrash()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	args := []string{"clean"}
	if category != "" && category != "all" {
		args = append(args, "--"+category)
	}
	args = append(args, "--yes")

	result := runMoleCommand(args...)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func emptyTrash() CleanResult {
	// Check if trash has items using Finder API (works with macOS permissions)
	countCmd := exec.Command("osascript", "-e", `tell application "Finder" to count of items of trash`)
	countOutput, err := countCmd.Output()
	if err != nil {
		return CleanResult{Success: false, Message: "Failed to check trash"}
	}

	countStr := strings.TrimSpace(string(countOutput))
	if countStr == "0" {
		return CleanResult{
			Success: true,
			Message: "Trash is already empty",
			Cleaned: 0,
			Output:  "Nothing to clean",
		}
	}

	// Get approximate size using du command on volumes trash folders
	var trashSize int64
	home := os.Getenv("HOME")

	// Try to get size from ~/.Trash (may fail on newer macOS)
	trashSize = getDirSize(filepath.Join(home, ".Trash"))

	// Also check volumes trash
	volTrash := filepath.Join("/", ".Trashes", fmt.Sprintf("%d", os.Getuid()))
	trashSize += getDirSize(volTrash)

	// Empty trash using Finder (handles permissions properly)
	emptyCmd := exec.Command("osascript", "-e", `tell application "Finder" to empty trash`)
	output, err := emptyCmd.CombinedOutput()
	if err != nil {
		return CleanResult{
			Success: false,
			Message: fmt.Sprintf("Failed to empty trash: %v", err),
			Output:  string(output),
		}
	}

	// If we couldn't measure size, estimate based on item count
	if trashSize == 0 {
		// Rough estimate: assume average 50MB per item
		count := 0
		fmt.Sscanf(countStr, "%d", &count)
		trashSize = int64(count) * 50 * 1024 * 1024
	}

	return CleanResult{
		Success: true,
		Message: "Trash emptied",
		Cleaned: trashSize,
		Output:  fmt.Sprintf("Emptied %s items from trash, freed approximately %s", countStr, formatBytes(trashSize)),
	}
}

type AppInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"size_human"`
}

func handleListApps(w http.ResponseWriter, r *http.Request) {
	apps := listApplications()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}

func isProtectedAppPath(path string) bool {
	if strings.HasPrefix(path, "/System/Applications/") {
		return true
	}
	bundleID := getBundleID(path)
	return strings.HasPrefix(bundleID, "com.apple.")
}

func getBundleID(appPath string) string {
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	if !fileExists(plistPath) {
		return ""
	}
	cmd := exec.Command("plutil", "-extract", "CFBundleIdentifier", "raw", plistPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func listApplications() []AppInfo {
	var apps []AppInfo
	cwd, _ := os.Getwd()
	appDirs := []string{
		"/Applications",
		filepath.Join(os.Getenv("HOME"), "Applications"),
		cwd,
	}

	// Helper to add an app if valid
	addApp := func(path string) {
		if isProtectedAppPath(path) {
			return
		}
		name := strings.TrimSuffix(filepath.Base(path), ".app")
		size := getDirSize(path)
		apps = append(apps, AppInfo{
			Name:      name,
			Path:      path,
			Size:      size,
			SizeHuman: formatBytes(size),
		})
	}

	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			// Direct .app bundle at top level
			if strings.HasSuffix(entry.Name(), ".app") {
				addApp(path)
				continue
			}

			// Check subdirectories for .app bundles (e.g., Adobe, Native Instruments)
			// Skip Utilities folder (Apple system apps)
			if entry.IsDir() && entry.Name() != "Utilities" {
				subEntries, err := os.ReadDir(path)
				if err != nil {
					continue
				}
				for _, subEntry := range subEntries {
					if strings.HasSuffix(subEntry.Name(), ".app") {
						subPath := filepath.Join(path, subEntry.Name())
						addApp(subPath)
					}
					// Check one more level deep (e.g., /Applications/Adobe/Subfolder/App.app)
					if subEntry.IsDir() && !strings.HasSuffix(subEntry.Name(), ".app") {
						level2Path := filepath.Join(path, subEntry.Name())
						level2Entries, err := os.ReadDir(level2Path)
						if err != nil {
							continue
						}
						for _, level2Entry := range level2Entries {
							if strings.HasSuffix(level2Entry.Name(), ".app") {
								level2AppPath := filepath.Join(level2Path, level2Entry.Name())
								addApp(level2AppPath)
							}
						}
					}
				}
			}
		}
	}
	return apps
}

func handleAppIcon(w http.ResponseWriter, r *http.Request) {
	appPath := r.URL.Query().Get("path")
	if appPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// Find the icon file in the app bundle
	resourcesPath := filepath.Join(appPath, "Contents", "Resources")

	// First try to read Info.plist to get the icon name
	iconName := ""
	plistPath := filepath.Join(appPath, "Contents", "Info.plist")
	if plistData, err := os.ReadFile(plistPath); err == nil {
		// Simple parsing for CFBundleIconFile
		plistStr := string(plistData)
		if idx := strings.Index(plistStr, "<key>CFBundleIconFile</key>"); idx != -1 {
			rest := plistStr[idx:]
			if startIdx := strings.Index(rest, "<string>"); startIdx != -1 {
				rest = rest[startIdx+8:]
				if endIdx := strings.Index(rest, "</string>"); endIdx != -1 {
					iconName = rest[:endIdx]
				}
			}
		}
	}

	// Try to find the icon file
	var iconPath string
	if iconName != "" {
		// Add .icns extension if not present
		if !strings.HasSuffix(iconName, ".icns") {
			iconName += ".icns"
		}
		iconPath = filepath.Join(resourcesPath, iconName)
	}

	// If not found, try common icon names
	if iconPath == "" || !fileExists(iconPath) {
		commonNames := []string{"AppIcon.icns", "app.icns", "icon.icns", "application.icns"}
		for _, name := range commonNames {
			testPath := filepath.Join(resourcesPath, name)
			if fileExists(testPath) {
				iconPath = testPath
				break
			}
		}
	}

	// If still not found, look for any .icns file
	if iconPath == "" || !fileExists(iconPath) {
		entries, _ := os.ReadDir(resourcesPath)
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".icns") {
				iconPath = filepath.Join(resourcesPath, entry.Name())
				break
			}
		}
	}

	if iconPath == "" || !fileExists(iconPath) {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}

	// Use sips to convert icns to png
	tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("appicon_%d.png", time.Now().UnixNano()))
	defer os.Remove(tmpFile)

	cmd := exec.Command("sips", "-s", "format", "png", "-z", "64", "64", iconPath, "--out", tmpFile)
	if err := cmd.Run(); err != nil {
		http.Error(w, "failed to convert icon", http.StatusInternalServerError)
		return
	}

	// Read and serve the PNG
	pngData, err := os.ReadFile(tmpFile)
	if err != nil {
		http.Error(w, "failed to read icon", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	w.Write(pngData)
}

func handleUninstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Apps []string `json:"apps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Batch uninstall all apps at once (single auth prompt)
	result := uninstallApps(req.Apps)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]CleanResult{result})
}

func writeLog(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Println(msg)

	// Broadcast to SSE clients (non-blocking)
	select {
	case logBroadcast <- msg:
	default:
		// Channel full, skip
	}

	// Also write to a file for user to retrieve
	cacheDir, err := os.UserCacheDir()
	if err == nil {
		logDir := filepath.Join(cacheDir, "Mole")
		os.MkdirAll(logDir, 0755)
		f, err := os.OpenFile(filepath.Join(logDir, "web-ui.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			f.WriteString(fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"), msg))
		}
	}
}

func findMoleScript() string {
	writeLog("Finding Mole CLI script...")

	// Try current directory
	if _, err := os.Stat("./mole"); err == nil {
		abs, _ := filepath.Abs("./mole")
		writeLog("Found Mole CLI at: %s", abs)
		return abs
	}
	// Try two levels up (common in dev: bin/web-go -> ../../mole)
	if _, err := os.Stat("../../mole"); err == nil {
		abs, _ := filepath.Abs("../../mole")
		writeLog("Found Mole CLI at: %s", abs)
		return abs
	}

	// Try relative to executable
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		// Try bundled location (relative to binary in MoleSwift.app/Contents/MacOS/web-go)
		bundled := filepath.Join(exeDir, "..", "Resources", "mole")
		if _, err := os.Stat(bundled); err == nil {
			writeLog("Found Mole CLI at executable-relative bundled location: %s", bundled)
			return bundled
		}
	}
	// Try bundled location (relative to binary in MoleSwift.app/Contents/MacOS/web-go)
	if _, err := os.Stat("../Resources/mole"); err == nil {
		abs, _ := filepath.Abs("../Resources/mole")
		writeLog("Found Mole CLI at bundled location: %s", abs)
		return abs
	}

	// Try command -v mole
	path, err := exec.LookPath("mole")
	if err == nil {
		writeLog("Found Mole CLI via PATH: %s", path)
		return path
	}

	// Try standard install path (LAST RESORT to prevent picking up broken installs)
	if _, err := os.Stat("/usr/local/bin/mole"); err == nil {
		writeLog("Found Mole CLI at standard path: /usr/local/bin/mole")
		return "/usr/local/bin/mole"
	}

	writeLog("ERROR: Mole CLI not found in any expected location")
	return ""
}

func uninstallApps(appPaths []string) CleanResult {
	if len(appPaths) == 0 {
		return CleanResult{Success: false, Message: "No apps specified"}
	}

	moleScript := findMoleScript()
	if moleScript == "" {
		return CleanResult{Success: false, Message: "Mole CLI not found. Please ensure Mole is installed correctly."}
	}

	var successful []string
	var failed []string
	var totalCleaned int64

	for _, appPath := range appPaths {
		writeLog("Attempting to uninstall: %s", appPath)

		if _, err := os.Stat(appPath); os.IsNotExist(err) {
			writeLog("ERROR: Path does not exist: %s", appPath)
			failed = append(failed, fmt.Sprintf("%s (not found)", filepath.Base(appPath)))
			continue
		}

		// Use the mole CLI for robust uninstallation
		// mole uninstall --path <path> --debug
		writeLog("Executing: %s uninstall --path %s --debug", moleScript, appPath)
		cmd := exec.Command(moleScript, "uninstall", "--path", appPath, "--debug")
		// MOLE_NO_CONFIRM=1 skips interactive confirmation
		// MOLE_GUI_MODE=1 tells the script to use osascript for admin privileges instead of TTY-based sudo
		cmd.Env = append(os.Environ(), "MOLE_NO_CONFIRM=1", "MOLE_GUI_MODE=1")

		// Stream output in real-time to SSE clients
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			writeLog("ERROR: Failed to start uninstall for %s: %v", appPath, err)
			failed = append(failed, fmt.Sprintf("%s (start failed)", filepath.Base(appPath)))
			continue
		}

		var output strings.Builder
		var wg sync.WaitGroup
		wg.Add(2)

		streamOutput := func(r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				output.WriteString(line + "\n")
				// Broadcast to SSE in real-time
				select {
				case logBroadcast <- stripANSI(line):
				default:
				}
			}
		}

		go streamOutput(stdout)
		go streamOutput(stderr)

		err := cmd.Wait()
		wg.Wait()

		if err != nil {
			writeLog("ERROR: Uninstallation failed for %s: %v", appPath, err)
			failed = append(failed, fmt.Sprintf("%s (%v)", filepath.Base(appPath), err))
		} else {
			writeLog("SUCCESS: Uninstalled %s", appPath)
			successful = append(successful, filepath.Base(appPath))
		}
	}

	if len(failed) > 0 && len(successful) == 0 {
		return CleanResult{Success: false, Message: fmt.Sprintf("Failed to remove: %s", strings.Join(failed, ", "))}
	}

	msg := fmt.Sprintf("Uninstalled %d app(s)", len(successful))
	if len(failed) > 0 {
		msg += fmt.Sprintf(" (Failed %d: %s)", len(failed), strings.Join(failed, ", "))
	}

	return CleanResult{
		Success: true,
		Message: msg,
		Cleaned: totalCleaned, // Will be 0 for now as we don't parse script output
		Output:  fmt.Sprintf("Used Mole CLI for comprehensive cleanup of: %s", strings.Join(successful, ", ")),
	}
}

type DirEntry struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"size_human"`
	IsDir     bool   `json:"is_dir"`
}

func handleAnalyze(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = os.Getenv("HOME")
	}

	entries := analyzePath(path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func analyzePath(path string) []DirEntry {
	entries := make([]DirEntry, 0)
	items, err := os.ReadDir(path)
	if err != nil {
		return entries
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range items {
		if strings.HasPrefix(item.Name(), ".") {
			continue
		}
		wg.Add(1)
		go func(item os.DirEntry) {
			defer wg.Done()
			fullPath := filepath.Join(path, item.Name())
			var size int64
			if item.IsDir() {
				size = getDirSize(fullPath)
			} else if info, err := item.Info(); err == nil {
				size = info.Size()
			}

			mu.Lock()
			entries = append(entries, DirEntry{
				Path:      fullPath,
				Name:      item.Name(),
				Size:      size,
				SizeHuman: formatBytes(size),
				IsDir:     item.IsDir(),
			})
			mu.Unlock()
		}(item)
	}
	wg.Wait()

	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Size > entries[i].Size {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries
}

func handleOpenFinder(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// Verify path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		http.Error(w, "path does not exist", http.StatusNotFound)
		return
	}

	// Open path in Finder using macOS 'open' command
	cmd := exec.Command("open", path)
	if err := cmd.Run(); err != nil {
		http.Error(w, fmt.Sprintf("failed to open in Finder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": path})
}

var errStopWalk = fmt.Errorf("stop walk")

// isCompactLargeFolder returns true for folder types that should be shown as single items
// (not drilled into) because they contain many small files
func isCompactLargeFolder(name string) bool {
	compactFolders := []string{
		"node_modules",
		".git",
		"vendor",
		"Pods",
		"DerivedData",
		"Build",
		"build",
		"dist",
		"target",
		"__pycache__",
		".venv",
		"venv",
		"env",
		".cache",
		"cache",
		"Cache",
		"Caches",
		".npm",
		".yarn",
		".pnpm-store",
		"go",
		".cargo",
		".rustup",
		".gradle",
		".m2",
		".cocoapods",
	}
	for _, cf := range compactFolders {
		if name == cf {
			return true
		}
	}
	return false
}

func handleAnalyzeLarge(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = os.Getenv("HOME")
	}

	minSizeStr := r.URL.Query().Get("min_size")
	minSize := int64(104857600) // Default 100MB
	if minSizeStr != "" {
		if parsed, err := strconv.ParseInt(minSizeStr, 10, 64); err == nil {
			minSize = parsed
		}
	}

	largeItems := make([]DirEntry, 0)
	var topLevelFolders []string
	var mu sync.Mutex

	// Scan recursively to find large files and compact folders
	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip the root path itself
		if filePath == path {
			return nil
		}

		name := info.Name()

		// Skip hidden files/dirs (but scan inside .Trash for large items)
		if strings.HasPrefix(name, ".") && name != ".Trash" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip Library folder to avoid system files
		if name == "Library" && info.IsDir() {
			return filepath.SkipDir
		}

		if info.IsDir() {
			parentDir := filepath.Dir(filePath)
			isTopLevel := parentDir == path
			isCompact := isCompactLargeFolder(name)

			if isTopLevel {
				// Collect top-level folders for parallel size calculation
				topLevelFolders = append(topLevelFolders, filePath)
			} else if isCompact {
				// Calculate size for compact folders immediately
				dirSize := getDirSizeWithLimit(filePath, 5)
				if dirSize >= minSize {
					largeItems = append(largeItems, DirEntry{
						Path:      filePath,
						Name:      name,
						Size:      dirSize,
						SizeHuman: formatBytes(dirSize),
						IsDir:     true,
					})
				}
				return filepath.SkipDir
			}
		} else {
			// Regular file - add if large enough
			if info.Size() >= minSize {
				largeItems = append(largeItems, DirEntry{
					Path:      filePath,
					Name:      name,
					Size:      info.Size(),
					SizeHuman: formatBytes(info.Size()),
					IsDir:     false,
				})
			}
		}

		// Limit to 200 items to avoid timeout
		if len(largeItems) >= 200 {
			return errStopWalk
		}
		return nil
	})

	// Calculate top-level folder sizes in parallel
	var wg sync.WaitGroup
	for _, folderPath := range topLevelFolders {
		wg.Add(1)
		go func(fp string) {
			defer wg.Done()
			dirSize := getDirSizeWithLimit(fp, 5)
			if dirSize >= minSize {
				mu.Lock()
				largeItems = append(largeItems, DirEntry{
					Path:      fp,
					Name:      filepath.Base(fp),
					Size:      dirSize,
					SizeHuman: formatBytes(dirSize),
					IsDir:     true,
				})
				mu.Unlock()
			}
		}(folderPath)
	}
	wg.Wait()

	// Sort by size descending
	for i := 0; i < len(largeItems); i++ {
		for j := i + 1; j < len(largeItems); j++ {
			if largeItems[j].Size > largeItems[i].Size {
				largeItems[i], largeItems[j] = largeItems[j], largeItems[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(largeItems)
}

// handleAnalyzeDownloads returns contents of ~/Downloads sorted by size
func handleAnalyzeDownloads(w http.ResponseWriter, r *http.Request) {
	downloadsPath := filepath.Join(os.Getenv("HOME"), "Downloads")

	entries, err := os.ReadDir(downloadsPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := make([]DirEntry, 0, len(entries))
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		size := info.Size()
		if entry.IsDir() {
			// Calculate directory size
			size = getDirSize(filepath.Join(downloadsPath, entry.Name()))
		}

		result = append(result, DirEntry{
			Path:      filepath.Join(downloadsPath, entry.Name()),
			Name:      entry.Name(),
			Size:      size,
			SizeHuman: formatBytes(size),
			IsDir:     entry.IsDir(),
		})
	}

	// Sort by size descending
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Size > result[i].Size {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Storage breakdown types
type StorageCategory struct {
	Name      string  `json:"name"`
	Size      int64   `json:"size"`
	SizeHuman string  `json:"size_human"`
	Percent   float64 `json:"percent"`
	Color     string  `json:"color"`
	Icon      string  `json:"icon"`
}

type CleanupSuggestion struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	SizeHuman   string `json:"size_human"`
	Action      string `json:"action"`
	Category    string `json:"category"`
}

type StorageBreakdown struct {
	Total       int64               `json:"total"`
	Used        int64               `json:"used"`
	Free        int64               `json:"free"`
	TotalHuman  string              `json:"total_human"`
	UsedHuman   string              `json:"used_human"`
	FreeHuman   string              `json:"free_human"`
	Categories  []StorageCategory   `json:"categories"`
	Suggestions []CleanupSuggestion `json:"suggestions"`
}

func handleStorageBreakdown(w http.ResponseWriter, r *http.Request) {
	home := os.Getenv("HOME")

	// Get disk usage
	usage, err := disk.Usage("/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	breakdown := StorageBreakdown{
		Total:      int64(usage.Total),
		Used:       int64(usage.Used),
		Free:       int64(usage.Free),
		TotalHuman: formatBytes(int64(usage.Total)),
		UsedHuman:  formatBytes(int64(usage.Used)),
		FreeHuman:  formatBytes(int64(usage.Free)),
	}

	// Calculate sizes for different categories
	var wg sync.WaitGroup
	var mu sync.Mutex

	categories := []struct {
		name  string
		path  string
		color string
		icon  string
	}{
		{"Applications", "/Applications", "#3b82f6", "apps"},
		{"Documents", filepath.Join(home, "Documents"), "#10b981", "document"},
		{"Downloads", filepath.Join(home, "Downloads"), "#f59e0b", "download"},
		{"Desktop", filepath.Join(home, "Desktop"), "#8b5cf6", "desktop"},
		{"Pictures", filepath.Join(home, "Pictures"), "#ec4899", "image"},
		{"Movies", filepath.Join(home, "Movies"), "#ef4444", "video"},
		{"Music", filepath.Join(home, "Music"), "#06b6d4", "music"},
		{"Code Projects", filepath.Join(home, "code"), "#22c55e", "code"},
		{"Docker Data", "/Volumes/data/docker", "#2563eb", "docker"},
		{"Media Hub", "/Volumes/data/media-hub", "#dc2626", "video"},
		{"Photos Library", "/Volumes/data/Photos Library.photoslibrary", "#ec4899", "camera"},
		{"System Library", filepath.Join(home, "Library"), "#6366f1", "library"},
	}

	for _, cat := range categories {
		wg.Add(1)
		go func(name, path, color, icon string) {
			defer wg.Done()
			size := getDirSize(path)
			if size > 0 {
				mu.Lock()
				breakdown.Categories = append(breakdown.Categories, StorageCategory{
					Name:      name,
					Size:      size,
					SizeHuman: formatBytes(size),
					Percent:   float64(size) / float64(usage.Used) * 100,
					Color:     color,
					Icon:      icon,
				})
				mu.Unlock()
			}
		}(cat.name, cat.path, cat.color, cat.icon)
	}

	// Check cache sizes for suggestions
	wg.Add(1)
	go func() {
		defer wg.Done()
		cacheSize := getDirSize(filepath.Join(home, "Library/Caches"))
		if cacheSize > 100*1024*1024 { // > 100MB
			mu.Lock()
			breakdown.Suggestions = append(breakdown.Suggestions, CleanupSuggestion{
				Title:       "Clear System Cache",
				Description: "Temporary files that can be safely removed",
				Size:        cacheSize,
				SizeHuman:   formatBytes(cacheSize),
				Action:      "clean",
				Category:    "cache",
			})
			mu.Unlock()
		}
	}()

	// Check logs
	wg.Add(1)
	go func() {
		defer wg.Done()
		logSize := getDirSize(filepath.Join(home, "Library/Logs"))
		if logSize > 50*1024*1024 { // > 50MB
			mu.Lock()
			breakdown.Suggestions = append(breakdown.Suggestions, CleanupSuggestion{
				Title:       "Clear Old Logs",
				Description: "Log files from apps and system",
				Size:        logSize,
				SizeHuman:   formatBytes(logSize),
				Action:      "clean",
				Category:    "logs",
			})
			mu.Unlock()
		}
	}()

	// Check Downloads for old files
	wg.Add(1)
	go func() {
		defer wg.Done()
		var oldDownloadsSize int64
		downloadsPath := filepath.Join(home, "Downloads")
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

		filepath.Walk(downloadsPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if info.ModTime().Before(thirtyDaysAgo) {
				oldDownloadsSize += info.Size()
			}
			return nil
		})

		if oldDownloadsSize > 100*1024*1024 { // > 100MB
			mu.Lock()
			breakdown.Suggestions = append(breakdown.Suggestions, CleanupSuggestion{
				Title:       "Old Downloads",
				Description: "Files in Downloads older than 30 days",
				Size:        oldDownloadsSize,
				SizeHuman:   formatBytes(oldDownloadsSize),
				Action:      "clean",
				Category:    "downloads",
			})
			mu.Unlock()
		}
	}()

	// Check Trash
	wg.Add(1)
	go func() {
		defer wg.Done()
		trashSize := getDirSize(filepath.Join(home, ".Trash"))
		if trashSize > 10*1024*1024 { // > 10MB
			mu.Lock()
			breakdown.Suggestions = append(breakdown.Suggestions, CleanupSuggestion{
				Title:       "Empty Trash",
				Description: "Files waiting to be permanently deleted",
				Size:        trashSize,
				SizeHuman:   formatBytes(trashSize),
				Action:      "clean",
				Category:    "trash",
			})
			mu.Unlock()
		}
	}()

	// Check Xcode derived data
	wg.Add(1)
	go func() {
		defer wg.Done()
		xcodeSize := getDirSize(filepath.Join(home, "Library/Developer/Xcode/DerivedData"))
		if xcodeSize > 500*1024*1024 { // > 500MB
			mu.Lock()
			breakdown.Suggestions = append(breakdown.Suggestions, CleanupSuggestion{
				Title:       "Xcode Build Files",
				Description: "Developer build cache (safe to delete)",
				Size:        xcodeSize,
				SizeHuman:   formatBytes(xcodeSize),
				Action:      "clean",
				Category:    "xcode",
			})
			mu.Unlock()
		}
	}()

	wg.Wait()

	// Sort categories by size
	for i := 0; i < len(breakdown.Categories); i++ {
		for j := i + 1; j < len(breakdown.Categories); j++ {
			if breakdown.Categories[j].Size > breakdown.Categories[i].Size {
				breakdown.Categories[i], breakdown.Categories[j] = breakdown.Categories[j], breakdown.Categories[i]
			}
		}
	}

	// Sort suggestions by size
	for i := 0; i < len(breakdown.Suggestions); i++ {
		for j := i + 1; j < len(breakdown.Suggestions); j++ {
			if breakdown.Suggestions[j].Size > breakdown.Suggestions[i].Size {
				breakdown.Suggestions[i], breakdown.Suggestions[j] = breakdown.Suggestions[j], breakdown.Suggestions[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakdown)
}

// OtherCategory represents a directory contributing to "Other" storage
type OtherCategory struct {
	Path      string  `json:"path"`
	Name      string  `json:"name"`
	Size      int64   `json:"size"`
	SizeHuman string  `json:"size_human"`
	Percent   float64 `json:"percent"`
	Type      string  `json:"type"`
	Icon      string  `json:"icon"`
}

type OtherBreakdown struct {
	TotalOther      int64           `json:"total_other"`
	TotalOtherHuman string          `json:"total_other_human"`
	Categories      []OtherCategory `json:"categories"`
}

func handleAnalyzeOther(w http.ResponseWriter, r *http.Request) {
	home := os.Getenv("HOME")

	// Get disk usage
	usage, err := disk.Usage("/")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Define the known categorized paths (same as in handleStorageBreakdown)
	categorizedPaths := map[string]bool{
		"/Applications":                              true,
		filepath.Join(home, "Documents"):             true,
		filepath.Join(home, "Downloads"):             true,
		filepath.Join(home, "Desktop"):               true,
		filepath.Join(home, "Pictures"):              true,
		filepath.Join(home, "Movies"):                true,
		filepath.Join(home, "Music"):                 true,
		filepath.Join(home, "code"):                  true,
		"/Volumes/data/docker":                       true,
		"/Volumes/data/media-hub":                    true,
		"/Volumes/data/Photos Library.photoslibrary": true,
		filepath.Join(home, "Library"):               true,
	}

	// Calculate categorized size
	var categorizedSize int64
	for path := range categorizedPaths {
		categorizedSize += getDirSize(path)
	}
	otherSize := int64(usage.Used) - categorizedSize

	// Define directories to scan for "Other" breakdown
	// These are major directories that might contain significant data
	otherDirs := []struct {
		path    string
		name    string
		dirType string
		icon    string
	}{
		// System directories
		{"/private/var", "System Data (var)", "system", "settings"},
		{"/System", "macOS System", "system", "apple"},
		{"/usr", "Unix Programs", "system", "terminal"},
		{"/opt", "Optional Software", "system", "package"},
		// User hidden directories
		{filepath.Join(home, ".local"), "Local Data", "user", "folder"},
		{filepath.Join(home, ".cache"), "User Cache", "cache", "trash"},
		{filepath.Join(home, ".docker"), "Docker Config", "developer", "docker"},
		{filepath.Join(home, ".npm"), "NPM Cache", "developer", "package"},
		{filepath.Join(home, ".cargo"), "Rust/Cargo", "developer", "code"},
		{filepath.Join(home, ".rustup"), "Rustup", "developer", "code"},
		{filepath.Join(home, ".gradle"), "Gradle Cache", "developer", "code"},
		{filepath.Join(home, ".m2"), "Maven Cache", "developer", "code"},
		{filepath.Join(home, ".vscode"), "VS Code", "developer", "code"},
		{filepath.Join(home, ".cursor"), "Cursor IDE", "developer", "code"},
		{filepath.Join(home, ".orbstack"), "OrbStack", "developer", "docker"},
		{filepath.Join(home, ".lima"), "Lima VMs", "developer", "docker"},
		{filepath.Join(home, ".vagrant.d"), "Vagrant", "developer", "docker"},
		// Other Volumes
		{"/Volumes", "External Volumes", "volumes", "harddrive"},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var categories []OtherCategory

	for _, dir := range otherDirs {
		// Skip if this path is categorized
		if categorizedPaths[dir.path] {
			continue
		}
		wg.Add(1)
		go func(path, name, dirType, icon string) {
			defer wg.Done()
			size := getDirSize(path)
			if size > 10*1024*1024 { // Only include if > 10MB
				mu.Lock()
				categories = append(categories, OtherCategory{
					Path:      path,
					Name:      name,
					Size:      size,
					SizeHuman: formatBytes(size),
					Percent:   float64(size) / float64(otherSize) * 100,
					Type:      dirType,
					Icon:      icon,
				})
				mu.Unlock()
			}
		}(dir.path, dir.name, dir.dirType, dir.icon)
	}

	wg.Wait()

	// Sort by size descending
	for i := 0; i < len(categories); i++ {
		for j := i + 1; j < len(categories); j++ {
			if categories[j].Size > categories[i].Size {
				categories[i], categories[j] = categories[j], categories[i]
			}
		}
	}

	// Calculate unaccounted space
	var accountedOther int64
	for _, cat := range categories {
		accountedOther += cat.Size
	}
	unaccounted := otherSize - accountedOther
	if unaccounted > 100*1024*1024 { // > 100MB
		categories = append(categories, OtherCategory{
			Path:      "",
			Name:      "Unaccounted (System/Protected)",
			Size:      unaccounted,
			SizeHuman: formatBytes(unaccounted),
			Percent:   float64(unaccounted) / float64(otherSize) * 100,
			Type:      "system",
			Icon:      "lock",
		})
	}

	breakdown := OtherBreakdown{
		TotalOther:      otherSize,
		TotalOtherHuman: formatBytes(otherSize),
		Categories:      categories,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(breakdown)
}

func handleOptimize(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Run optimization tasks directly (no sudo required)
	var output strings.Builder
	output.WriteString("System Optimization\n")
	output.WriteString("==================\n\n")

	// 1. Flush DNS cache (works without sudo on modern macOS)
	output.WriteString("DNS Cache: ")
	if err := exec.Command("dscacheutil", "-flushcache").Run(); err == nil {
		output.WriteString("Flushed\n")
	} else {
		// Attempt with sudo via osascript
		adminCmd := exec.Command("osascript", "-e", `do shell script "dscacheutil -flushcache; killall -HUP mDNSResponder" with administrator privileges`)
		if err := adminCmd.Run(); err == nil {
			output.WriteString("Flushed (with admin)\n")
		} else {
			output.WriteString("Skipped (requires admin)\n")
		}
	}

	// 2. Clear QuickLook thumbnails
	output.WriteString("QuickLook Cache: ")
	qlPath := filepath.Join(os.Getenv("HOME"), "Library", "Caches", "com.apple.QuickLook.thumbnailcache")
	if err := os.RemoveAll(qlPath); err == nil {
		output.WriteString("Cleared\n")
	} else {
		output.WriteString("Skipped\n")
	}

	// 3. Clear icon services cache
	output.WriteString("Icon Cache: ")
	iconPath := filepath.Join(os.Getenv("HOME"), "Library", "Caches", "com.apple.iconservices.store")
	if err := os.RemoveAll(iconPath); err == nil {
		output.WriteString("Cleared\n")
	} else {
		output.WriteString("Skipped\n")
	}

	// 4. Purge inactive memory
	output.WriteString("Memory: ")
	if err := exec.Command("purge").Run(); err == nil {
		output.WriteString("Inactive memory purged\n")
	} else {
		// Attempt with sudo via osascript as requested by user
		writeLog("Memory purge requires admin privileges, prompting user...")
		adminCmd := exec.Command("osascript", "-e", `do shell script "purge" with administrator privileges`)
		if err := adminCmd.Run(); err == nil {
			output.WriteString("Memory purged (with admin)\n")
		} else {
			output.WriteString("Skipped (requires admin)\n")
		}
	}

	// 5. Rebuild Spotlight index for user folders
	output.WriteString("Spotlight: ")
	homeDir := os.Getenv("HOME")
	exec.Command("mdutil", "-i", "on", homeDir).Run()
	output.WriteString("Index refreshed\n")

	// 6. Clear font caches
	output.WriteString("Font Caches: ")
	fontCaches := []string{
		filepath.Join(os.Getenv("HOME"), "Library", "Caches", "com.apple.FontRegistry"),
	}
	for _, fc := range fontCaches {
		os.RemoveAll(fc)
	}
	output.WriteString("Cleared\n")

	// 7. Restart Finder to apply changes
	output.WriteString("\nRestarting Finder to apply changes...\n")
	exec.Command("killall", "Finder").Run()

	output.WriteString("\nOptimization complete!")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CleanResult{
		Success: true,
		Message: "System optimized",
		Output:  output.String(),
	})
}

type PurgeItem struct {
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"size_human"`
	Type      string `json:"type"`
}

func handlePurgeScan(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = os.Getenv("HOME")
	}

	items := scanForPurge(path)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func scanForPurge(root string) []PurgeItem {
	var items []PurgeItem
	targets := []string{"node_modules", "target", "build", "dist", ".next", "__pycache__", "venv", ".venv"}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		for _, t := range targets {
			if info.Name() == t {
				size := getDirSize(path)
				items = append(items, PurgeItem{
					Path:      path,
					Size:      size,
					SizeHuman: formatBytes(size),
					Type:      t,
				})
				return filepath.SkipDir
			}
		}
		return nil
	})

	return items
}

func handlePurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var totalCleaned int64
	for _, p := range req.Paths {
		size := getDirSize(p)
		if err := os.RemoveAll(p); err == nil {
			totalCleaned += size
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(CleanResult{
		Success: true,
		Message: fmt.Sprintf("Removed %d items", len(req.Paths)),
		Cleaned: totalCleaned,
	})
}

func handleInstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	installerScript := `#!/bin/bash
# Mole Web UI Installer
# Install and run Mole web UI on this Mac

set -e

echo "🐭 Mole Web UI Installer"
echo "========================"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "📦 Installing Go..."
    if command -v brew &> /dev/null; then
        brew install go
    else
        echo "❌ Error: Homebrew not found. Please install Homebrew first:"
        echo "   /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        exit 1
    fi
fi

# Clone or download Mole
INSTALL_DIR="$HOME/.mole"
if [ -d "$INSTALL_DIR" ]; then
    echo "📂 Mole already installed at $INSTALL_DIR"
    cd "$INSTALL_DIR"
    git pull 2>/dev/null || echo "   (Unable to update, continuing with existing installation)"
else
    echo "📥 Downloading Mole..."
    git clone https://github.com/tw93/mole.git "$INSTALL_DIR"
    cd "$INSTALL_DIR"
fi

# Build web server
echo "🔨 Building web server..."
go build -o bin/web-go ./cmd/web/

# Create .env config
if [ ! -f "deploy/.env" ]; then
    echo "⚙️  Creating configuration..."
    cat > deploy/.env << 'EOF'
# Mole Web UI Configuration
MOLE_PORT=8081
MOLE_HOST=0.0.0.0

# Authentication (optional - disabled for local network)
# MOLE_AUTH_USER=admin
# MOLE_AUTH_PASS=changeme

TZ=America/New_York
EOF
fi

# Start server
echo "🚀 Starting Mole Web UI..."
./deploy/start.sh

echo ""
echo "✅ Installation complete!"
echo ""
echo "   Access your dashboard at:"
echo "   http://$(hostname).local:8081"
echo "   http://$(ipconfig getifaddr en0 2>/dev/null || echo "localhost"):8081"
echo ""
echo "   To stop:  cd $INSTALL_DIR && ./deploy/stop.sh"
echo "   To start: cd $INSTALL_DIR && ./deploy/start.sh"
echo ""
`

	fmt.Fprint(w, installerScript)
}

func runMoleCommand(args ...string) CleanResult {
	mole := findMoleScript()
	if mole == "" {
		return CleanResult{
			Success: false,
			Message: "Mole CLI not found",
		}
	}

	cmd := exec.Command(mole, args...)
	// Try to set Dir to mole's parent dir if possible
	cmd.Dir = filepath.Dir(mole)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	writeLog("Running command: %s %v", mole, args)
	if err := cmd.Start(); err != nil {
		errMsg := fmt.Sprintf("Error starting command: %v", err)
		fmt.Println(errMsg)
		select {
		case logBroadcast <- errMsg:
		default:
		}
		return CleanResult{
			Success: false,
			Message: err.Error(),
		}
	}

	var output strings.Builder
	var wg sync.WaitGroup
	wg.Add(2)

	processLine := func(r io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			// Broadcast to SSE
			select {
			case logBroadcast <- line:
			default:
				// Channel full, drop or handle
			}
		}
	}

	go processLine(stdout)
	go processLine(stderr)

	err := cmd.Wait()
	wg.Wait()

	return CleanResult{
		Success: err == nil,
		Message: func() string {
			if err != nil {
				return err.Error()
			}
			return "Completed successfully"
		}(),
		Output: stripANSI(output.String()),
	}
}

// stripANSI removes ANSI escape codes from a string
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func getDirSize(path string) int64 {
	return getDirSizeWithLimit(path, 3) // Limit depth to 3 levels for speed
}

func getDirSizeWithLimit(path string, maxDepth int) int64 {
	var size int64
	baseDepth := strings.Count(path, string(os.PathSeparator))

	filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}

		// Calculate current depth relative to base
		currentDepth := strings.Count(filePath, string(os.PathSeparator)) - baseDepth

		// Skip directories that are too deep
		if info.IsDir() && currentDepth >= maxDepth {
			return filepath.SkipDir
		}

		// Skip slow directories
		name := info.Name()
		if info.IsDir() && (name == "node_modules" || name == ".git" || name == "Library" || name == "Caches") {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Delete files/folders API
type DeleteRequest struct {
	Paths []string `json:"paths"`
}

type DeleteResult struct {
	Success      bool     `json:"success"`
	DeletedCount int      `json:"deleted_count"`
	DeletedSize  int64    `json:"deleted_size"`
	SizeHuman    string   `json:"size_human"`
	Failed       []string `json:"failed,omitempty"`
	Errors       []string `json:"errors,omitempty"`
}

func isProtectedPath(path string) bool {
	protectedPrefixes := []string{
		"/System",
		"/Library",
		"/usr",
		"/bin",
		"/sbin",
		"/private",
		"/var",
		"/etc",
	}
	for _, prefix := range protectedPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	// Protect Apple apps in /Applications
	if strings.HasPrefix(path, "/Applications/") && isProtectedAppPath(path) {
		return true
	}
	return false
}

func handleDeleteFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Paths) == 0 {
		http.Error(w, "No paths specified", http.StatusBadRequest)
		return
	}

	result := DeleteResult{}

	for _, path := range req.Paths {
		// Safety check: prevent deleting protected paths
		if isProtectedPath(path) {
			result.Failed = append(result.Failed, path)
			result.Errors = append(result.Errors, "Protected system path")
			continue
		}

		// Verify path exists
		info, err := os.Stat(path)
		if os.IsNotExist(err) {
			result.Failed = append(result.Failed, path)
			result.Errors = append(result.Errors, "Path does not exist")
			continue
		}

		// Get size before deletion
		var size int64
		if info.IsDir() {
			size = getDirSize(path)
		} else {
			size = info.Size()
		}

		// Delete the file/folder
		if err := os.RemoveAll(path); err != nil {
			result.Failed = append(result.Failed, path)
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.DeletedCount++
			result.DeletedSize += size
			writeLog("Deleted: %s (%s)", path, formatBytes(size))
		}
	}

	result.Success = len(result.Failed) == 0
	result.SizeHuman = formatBytes(result.DeletedSize)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Unsupported apps detection (Intel apps on Apple Silicon)
type UnsupportedApp struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Size          int64    `json:"size"`
	SizeHuman     string   `json:"size_human"`
	Architectures []string `json:"architectures"`
	Reason        string   `json:"reason"`
}

func getAppArchitectures(appPath string) ([]string, error) {
	// Find the main executable in the app bundle
	execDir := filepath.Join(appPath, "Contents", "MacOS")
	entries, err := os.ReadDir(execDir)
	if err != nil {
		return nil, err
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no executable found")
	}

	// Usually the first entry is the main executable
	executable := filepath.Join(execDir, entries[0].Name())

	// Run lipo -archs to get architectures
	cmd := exec.Command("lipo", "-archs", executable)
	output, err := cmd.Output()
	if err != nil {
		// Fallback to file command
		cmd = exec.Command("file", executable)
		output, err = cmd.Output()
		if err != nil {
			return nil, err
		}
		// Parse file output for architectures
		outputStr := string(output)
		var archs []string
		if strings.Contains(outputStr, "x86_64") {
			archs = append(archs, "x86_64")
		}
		if strings.Contains(outputStr, "arm64") {
			archs = append(archs, "arm64")
		}
		return archs, nil
	}

	// Parse lipo output: "x86_64 arm64" -> ["x86_64", "arm64"]
	archStr := strings.TrimSpace(string(output))
	return strings.Fields(archStr), nil
}

func handleUnsupportedApps(w http.ResponseWriter, r *http.Request) {
	// Check if we're on Apple Silicon
	isAppleSilicon := runtime.GOARCH == "arm64"

	var unsupported []UnsupportedApp

	appDirs := []string{
		"/Applications",
		filepath.Join(os.Getenv("HOME"), "Applications"),
	}

	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".app") {
				continue
			}

			appPath := filepath.Join(dir, entry.Name())

			// Skip system apps
			if isProtectedAppPath(appPath) {
				continue
			}

			archs, err := getAppArchitectures(appPath)
			if err != nil {
				continue
			}

			// On Apple Silicon, flag apps that are Intel-only
			if isAppleSilicon {
				hasARM := false
				for _, arch := range archs {
					if arch == "arm64" {
						hasARM = true
						break
					}
				}

				if !hasARM && len(archs) > 0 {
					size := getDirSize(appPath)
					unsupported = append(unsupported, UnsupportedApp{
						Name:          strings.TrimSuffix(entry.Name(), ".app"),
						Path:          appPath,
						Size:          size,
						SizeHuman:     formatBytes(size),
						Architectures: archs,
						Reason:        "Intel-only app (runs via Rosetta 2)",
					})
				}
			}
		}
	}

	// Sort by size descending
	for i := 0; i < len(unsupported); i++ {
		for j := i + 1; j < len(unsupported); j++ {
			if unsupported[j].Size > unsupported[i].Size {
				unsupported[i], unsupported[j] = unsupported[j], unsupported[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(unsupported)
}

func handleDebugLogs(w http.ResponseWriter, r *http.Request) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		http.Error(w, "Could not find cache dir", http.StatusInternalServerError)
		return
	}
	logPath := filepath.Join(cacheDir, "Mole", "web-ui.log")
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("No log file found yet."))
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(content)
}

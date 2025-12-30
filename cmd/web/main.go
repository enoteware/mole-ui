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
	"runtime"
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
	http.HandleFunc("/api/analyze", basicAuth(handleAnalyze))
	http.HandleFunc("/api/optimize", basicAuth(handleOptimize))
	http.HandleFunc("/api/purge", basicAuth(handlePurge))
	http.HandleFunc("/api/purge/scan", basicAuth(handlePurgeScan))
	http.HandleFunc("/api/status/stream", basicAuth(handleStatusStream))
	http.HandleFunc("/api/logs", basicAuth(handleLogsStream))

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
	LocalIP     string      `json:"local_ip"`
	OS          string      `json:"os"`
	Uptime      string      `json:"uptime"`
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
	args := []string{"clean"}
	if category != "" && category != "all" {
		args = append(args, "--"+category)
	}
	args = append(args, "--yes")

	result := runMoleCommand(args...)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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

func listApplications() []AppInfo {
	var apps []AppInfo
	appDirs := []string{"/Applications", filepath.Join(os.Getenv("HOME"), "Applications")}

	for _, dir := range appDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".app") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			size := getDirSize(path)
			apps = append(apps, AppInfo{
				Name:      strings.TrimSuffix(entry.Name(), ".app"),
				Path:      path,
				Size:      size,
				SizeHuman: formatBytes(size),
			})
		}
	}
	return apps
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

	var results []CleanResult
	for _, app := range req.Apps {
		result := runMoleCommand("uninstall", app, "--yes")
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
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
	var entries []DirEntry
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

func handleOptimize(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result := runMoleCommand("optimize", "--yes")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
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
	mole := filepath.Join(moleDir, "mole")
	if !fileExists(mole) {
		mole = filepath.Join(moleDir, "mo")
	}

	cmd := exec.Command(mole, args...)
	cmd.Dir = moleDir

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	fmt.Printf("Running command: %s %v\n", mole, args)
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
		Output: output.String(),
	}
}

func getDirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
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

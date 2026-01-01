package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LogFile struct {
	Name string
	Path string
}

func listLogFiles() []LogFile {
	home, _ := os.UserHomeDir()
	cacheDir, _ := os.UserCacheDir()
	configDir, _ := os.UserConfigDir()

	return []LogFile{
		{
			Name: "server.log",
			Path: filepath.Join(configDir, "Mole", "server.log"),
		},
		{
			Name: "web-ui.log",
			Path: filepath.Join(cacheDir, "Mole", "web-ui.log"),
		},
		{
			Name: "mole.log",
			Path: filepath.Join(home, ".config", "mole", "mole.log"),
		},
		{
			Name: "mole_debug_session.log",
			Path: filepath.Join(home, ".config", "mole", "mole_debug_session.log"),
		},
	}
}

func handleOpenLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logsDir, err := logsBundleDir()
	if err != nil {
		http.Error(w, "Could not find logs directory", http.StatusInternalServerError)
		return
	}

	if err := collectLogs(logsDir); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := exec.Command("open", logsDir).Run(); err != nil {
		http.Error(w, "Failed to open logs folder", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "path": logsDir})
}

func handleLogsBundle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="Mole-logs.zip"`)
	w.Header().Set("Cache-Control", "no-store")

	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	wroteFile := false
	for _, logFile := range listLogFiles() {
		if !fileExists(logFile.Path) {
			continue
		}

		f, err := os.Open(logFile.Path)
		if err != nil {
			continue
		}

		writer, err := zipWriter.Create(logFile.Name)
		if err == nil {
			if _, err := io.Copy(writer, f); err == nil {
				wroteFile = true
			}
		}
		f.Close()
	}

	if !wroteFile {
		readme, _ := zipWriter.Create("README.txt")
		if readme != nil {
			readme.Write([]byte("No logs were found yet.\n"))
		}
	}
}

func logsBundleDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "Mole", "Logs"), nil
}

func collectLogs(dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	var lines []string
	for _, logFile := range listLogFiles() {
		lines = append(lines, logFile.Name+" -> "+logFile.Path)
		if !fileExists(logFile.Path) {
			continue
		}
		if err := copyFile(logFile.Path, filepath.Join(dstDir, logFile.Name)); err != nil {
			return err
		}
	}

	readmePath := filepath.Join(dstDir, "README.txt")
	readme := "Mole Logs\n\n" + strings.Join(lines, "\n") + "\n"
	_ = os.WriteFile(readmePath, []byte(readme), 0644)

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

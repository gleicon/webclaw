package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// handleHealth returns server status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}{
		Status:  "ok",
		Version: version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// FileReadRequest is the request body for file read operations
type FileReadRequest struct {
	Path string `json:"path"`
}

// FileReadResponse is the response body for file read operations
type FileReadResponse struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// handleFileRead reads a file from the local filesystem
func handleFileRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FileReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	path, err := sanitizePath(req.Path)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Read error: %v", err), http.StatusInternalServerError)
		return
	}

	resp := FileReadResponse{
		Content: string(content),
		Path:    path,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// FileWriteRequest is the request body for file write operations
type FileWriteRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// FileWriteResponse is the response body for file write operations
type FileWriteResponse struct {
	Path string `json:"path"`
	Size int    `json:"size"`
}

// handleFileWrite writes content to a file
func handleFileWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FileWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Security: prevent directory traversal
	path, err := sanitizePath(req.Path)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Create parent directories if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, fmt.Sprintf("Create directory error: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(path, []byte(req.Content), 0644); err != nil {
		http.Error(w, fmt.Sprintf("Write error: %v", err), http.StatusInternalServerError)
		return
	}

	resp := FileWriteResponse{
		Path: path,
		Size: len(req.Content),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// FileListRequest is the request body for file list operations
type FileListRequest struct {
	Path string `json:"path"`
}

// FileInfo describes a file or directory entry
type FileInfo struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

// FileListResponse is the response body for file list operations
type FileListResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
}

// handleFileList lists directory contents
func handleFileList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req FileListRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	path, err := sanitizePath(req.Path)
	if err != nil {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Read error: %v", err), http.StatusInternalServerError)
		return
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, _ := entry.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileInfo{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Size:  size,
		})
	}

	resp := FileListResponse{
		Path:  path,
		Files: files,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// sanitizePath prevents directory traversal attacks
func sanitizePath(path string) (string, error) {
	// Clean the path
	path = filepath.Clean(path)

	// Reject paths with .. components after cleaning
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path contains ..")
	}

	return path, nil
}

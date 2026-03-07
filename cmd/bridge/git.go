package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
)

// GitCloneRequest is the request body for git clone
type GitCloneRequest struct {
	URL       string `json:"url"`
	Directory string `json:"directory"`
	Branch    string `json:"branch,omitempty"`
}

// GitCloneResponse is the response body for git clone
type GitCloneResponse struct {
	Directory string `json:"directory"`
	Success   bool   `json:"success"`
}

// handleGitClone clones a git repository
func handleGitClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GitCloneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" || req.Directory == "" {
		http.Error(w, "url and directory are required", http.StatusBadRequest)
		return
	}

	dir, err := sanitizePath(req.Directory)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	args := []string{"clone"}
	if req.Branch != "" {
		args = append(args, "-b", req.Branch)
	}
	args = append(args, req.URL, dir)

	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		http.Error(w, fmt.Sprintf("Git clone failed: %s", output), http.StatusInternalServerError)
		return
	}

	resp := GitCloneResponse{
		Directory: dir,
		Success:   true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// GitCommitRequest is the request body for git commit
type GitCommitRequest struct {
	Directory string   `json:"directory"`
	Message   string   `json:"message"`
	Files     []string `json:"files,omitempty"` // empty = all
}

// GitCommitResponse is the response body for git commit
type GitCommitResponse struct {
	Success bool   `json:"success"`
	Hash    string `json:"hash,omitempty"`
}

// handleGitCommit stages and commits files
func handleGitCommit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GitCommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Directory == "" || req.Message == "" {
		http.Error(w, "directory and message are required", http.StatusBadRequest)
		return
	}

	dir, err := sanitizePath(req.Directory)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	// Add files
	var addCmd *exec.Cmd
	if len(req.Files) > 0 {
		addArgs := append([]string{"add"}, req.Files...)
		addCmd = exec.Command("git", addArgs...)
	} else {
		addCmd = exec.Command("git", "add", ".")
	}
	addCmd.Dir = dir
	if output, err := addCmd.CombinedOutput(); err != nil {
		http.Error(w, fmt.Sprintf("Git add failed: %s", output), http.StatusInternalServerError)
		return
	}

	// Commit
	commitCmd := exec.Command("git", "commit", "-m", req.Message)
	commitCmd.Dir = dir
	output, err := commitCmd.CombinedOutput()

	if err != nil {
		http.Error(w, fmt.Sprintf("Git commit failed: %s", output), http.StatusInternalServerError)
		return
	}

	resp := GitCommitResponse{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// GitPushRequest is the request body for git push
type GitPushRequest struct {
	Directory string `json:"directory"`
	Remote    string `json:"remote,omitempty"`
	Branch    string `json:"branch,omitempty"`
}

// GitPushResponse is the response body for git push
type GitPushResponse struct {
	Success bool `json:"success"`
}

// handleGitPush pushes commits to remote
func handleGitPush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GitPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Directory == "" {
		http.Error(w, "directory is required", http.StatusBadRequest)
		return
	}

	dir, err := sanitizePath(req.Directory)
	if err != nil {
		http.Error(w, "Invalid directory", http.StatusBadRequest)
		return
	}

	args := []string{"push"}
	if req.Remote != "" {
		args = append(args, req.Remote)
	}
	if req.Branch != "" {
		args = append(args, req.Branch)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()

	if err != nil {
		http.Error(w, fmt.Sprintf("Git push failed: %s", output), http.StatusInternalServerError)
		return
	}

	resp := GitPushResponse{Success: true}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

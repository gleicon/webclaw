package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// ExecRequest is the request body for shell execution
type ExecRequest struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Dir     string            `json:"dir,omitempty"`
	Timeout int               `json:"timeout_seconds,omitempty"`
}

// ExecResponse is the response body for shell execution
type ExecResponse struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

// handleExec runs a shell command and returns stdout/stderr
func handleExec(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	// Security: block dangerous commands
	if isDangerousCommand(req.Command, req.Args) {
		http.Error(w, "Command not allowed", http.StatusForbidden)
		return
	}

	// Set timeout (max 5 minutes)
	timeout := 30
	if req.Timeout > 0 && req.Timeout <= 300 {
		timeout = req.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, req.Command, req.Args...)

	if req.Dir != "" {
		dir, err := sanitizePath(req.Dir)
		if err != nil {
			http.Error(w, "Invalid directory", http.StatusBadRequest)
			return
		}
		cmd.Dir = dir
	}

	// Set environment variables
	if len(req.Env) > 0 {
		env := make([]string, 0, len(req.Env))
		for k, v := range req.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	resp := ExecResponse{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// isDangerousCommand blocks unsafe shell operations
func isDangerousCommand(cmd string, args []string) bool {
	dangerous := []string{
		"rm -rf /",
		"sudo",
		"su",
		"dd if=/dev/zero",
		"mkfs",
		"fdisk",
		":(){ :|:& };:", // Fork bomb
	}

	fullCmd := cmd + " " + strings.Join(args, " ")
	for _, d := range dangerous {
		if strings.Contains(fullCmd, d) {
			return true
		}
	}
	return false
}

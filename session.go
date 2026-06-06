package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// InitDirectorySession initializes the session's directory if required.
func InitDirectorySession(s Session) error {
	if _, err := os.Stat(s.Directory); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", s.Directory)
	}

	if s.Service == "agx" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "agx", "init", "--auto")
		cmd.Dir = s.Directory

		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to run agx init --auto (output: %s): %w", string(output), err)
		}
	}
	return nil
}

// RunQuery executes the appropriate CLI tool in the target directory and returns the result.
func RunQuery(s Session, prompt string) (string, error) {
	if _, err := os.Stat(s.Directory); os.IsNotExist(err) {
		return "", fmt.Errorf("directory does not exist: %s", s.Directory)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var cmd *exec.Cmd
	switch s.Service {
	case "agx":
		cmd = exec.CommandContext(ctx, "agx", prompt)
	case "grok":
		cmd = exec.CommandContext(ctx, "grok", "-c", "-p", prompt)
	default:
		return "", fmt.Errorf("unsupported service type: %s", s.Service)
	}

	cmd.Dir = s.Directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command execution failed: %w", err)
	}

	return string(output), nil
}

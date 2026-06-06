package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// GetChatSessionsBasePath returns the path to the hidden chats folder in the home directory.
func GetChatSessionsBasePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to local workspace if home directory is unavailable
		return NormalizePath("./.remote-controller/chats")
	}
	return NormalizePath(filepath.Join(home, ".remote-controller", "chats"))
}

// CreateChatSessionDir generates a unique directory under the chats base path and creates it.
func CreateChatSessionDir() (string, string, error) {
	u := uuid.New().String()
	dir := filepath.Join(GetChatSessionsBasePath(), u)
	normalized := NormalizePath(dir)

	if err := os.MkdirAll(normalized, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create chat directory: %w", err)
	}

	return u, normalized, nil
}

package main

import (
	"path/filepath"
	"strings"
)

// NormalizePath cleans the path and standardizes separators to forward slashes.
func NormalizePath(p string) string {
	cleaned := filepath.Clean(p)
	// Replace backslashes with forward slashes for cross-platform regex matching
	normalized := strings.ReplaceAll(cleaned, "\\", "/")
	return normalized
}

// IsValidPath checks if the cleaned path matches the configured allowed directories regex.
func IsValidPath(p string) bool {
	normalized := NormalizePath(p)
	if AllowedDirsRegexp == nil {
		return true
	}
	return AllowedDirsRegexp.MatchString(normalized)
}

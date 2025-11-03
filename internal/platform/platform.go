package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// IsMacOS checks if the current operating system is macOS
func IsMacOS() bool {
	_, err := os.Stat("/System/Library/CoreServices/Finder.app")
	return err == nil
}

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ValidateDir validates and cleans a directory path
func ValidateDir(dir string) (string, error) {
	if dir == "" {
		return "", fmt.Errorf("directory path is empty")
	}

	// Get absolute path
	cleaned, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Verify it's a real directory
	info, err := os.Stat(cleaned)
	if err != nil {
		return "", fmt.Errorf("directory does not exist: %w", err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", cleaned)
	}

	return cleaned, nil
}

// ValidatePID checks if a PID is within valid bounds
func ValidatePID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("pid must be positive: %d", pid)
	}
	if pid > 2147483647 {
		return fmt.Errorf("pid exceeds maximum value: %d", pid)
	}
	return nil
}

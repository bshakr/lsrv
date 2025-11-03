package git

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bassemshaker/lsrv/internal/platform"
)

// IsRepo checks if the given directory is a git repository
func IsRepo(dir string) bool {
	// Validate directory path
	cleanedDir, err := platform.ValidateDir(dir)
	if err != nil {
		return false
	}

	// Check if .git directory exists
	gitDir := filepath.Join(cleanedDir, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		return true
	}

	// Try git command
	cmd := exec.Command("git", "-C", cleanedDir, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetRepoName returns the repository name from git remote or directory name
func GetRepoName(dir string) string {
	// Validate directory path
	cleanedDir, err := platform.ValidateDir(dir)
	if err != nil {
		log.Printf("git: failed to validate directory for GetRepoName: %v", err)
		return filepath.Base(dir)
	}

	// Try to get from git remote
	cmd := exec.Command("git", "-C", cleanedDir, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		url := strings.TrimSpace(string(output))
		// Extract repo name from URL
		parts := strings.Split(url, "/")
		if len(parts) == 0 {
			return filepath.Base(cleanedDir)
		}
		name := parts[len(parts)-1]
		return strings.TrimSuffix(name, ".git")
	}

	// Fall back to directory name
	return filepath.Base(cleanedDir)
}

// GetBranch returns the current git branch name
func GetBranch(dir string) string {
	// Validate directory path
	cleanedDir, err := platform.ValidateDir(dir)
	if err != nil {
		log.Printf("git: failed to validate directory for GetBranch: %v", err)
		return "N/A"
	}

	cmd := exec.Command("git", "-C", cleanedDir, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("git: failed to get branch for %s: %v", cleanedDir, err)
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

package detector

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/bassemshaker/lsrv/internal/git"
	"github.com/bassemshaker/lsrv/internal/platform"
	"github.com/bassemshaker/lsrv/internal/types"
)

// Compile regex once at package level for performance
var portRegex = regexp.MustCompile(`:(\d+)\s+\(LISTEN\)`)

// gitInfo holds the result of parallel git operations
type gitInfo struct {
	repo   string
	branch string
}

// processInfo holds initial process data before CWD lookup
type processInfo struct {
	pid     int
	command string
	port    int
}

// FindServers discovers all running development servers
func FindServers() ([]types.Server, error) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run lsof: %w", err)
	}

	// First pass: collect all PIDs and process info
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var processes []processInfo
	var pids []int

	for scanner.Scan() {
		line := scanner.Text()

		// Skip header
		if strings.HasPrefix(line, "COMMAND") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		pidStr := fields[1]
		command := fields[0]

		// Extract port from the line
		port := extractPort(line)
		if port == 0 || !isDevPort(port) {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		// Validate PID before collecting
		if err := platform.ValidatePID(pid); err != nil {
			continue
		}

		processes = append(processes, processInfo{
			pid:     pid,
			command: command,
			port:    port,
		})
		pids = append(pids, pid)
	}

	// Batch get all CWDs in a single lsof call
	cwdMap := batchGetProcessCWDs(pids)

	// Collect unique CWDs and check if they're git repos in parallel
	uniqueCWDs := make(map[string]bool)
	for _, cwd := range cwdMap {
		if cwd != "" {
			uniqueCWDs[cwd] = true
		}
	}

	// Batch check all unique directories for git repos in parallel
	gitRepoCache := batchCheckGitRepos(uniqueCWDs)

	// Collect git repos that passed the check for batch git info fetching
	gitRepoDirs := make(map[string]bool)
	for dir, isRepo := range gitRepoCache {
		if isRepo {
			gitRepoDirs[dir] = true
		}
	}

	// Batch fetch git info (repo name and branch) for all git repos in parallel
	gitInfoCache := batchGetGitInfo(gitRepoDirs)

	// Second pass: build server list using cached results
	seenServers := make(map[string]bool)
	var servers []types.Server

	for _, proc := range processes {
		cwd, ok := cwdMap[proc.pid]
		if !ok || cwd == "" {
			continue
		}

		// Only show servers in git repositories (use cached result)
		if !gitRepoCache[cwd] {
			continue
		}

		// Get repo name and branch from cache
		info, ok := gitInfoCache[cwd]
		if !ok {
			continue
		}

		// Create unique key to deduplicate
		key := fmt.Sprintf("%s|%s|%s|%d", info.repo, info.branch, proc.command, proc.port)
		if seenServers[key] {
			continue
		}
		seenServers[key] = true

		servers = append(servers, types.Server{
			Repo:    info.repo,
			Branch:  info.branch,
			Process: proc.command,
			Port:    proc.port,
			CWD:     cwd,
		})
	}

	// Sort servers by repo, branch, port
	sort.Slice(servers, func(i, j int) bool {
		if servers[i].Repo != servers[j].Repo {
			return servers[i].Repo < servers[j].Repo
		}
		if servers[i].Branch != servers[j].Branch {
			return servers[i].Branch < servers[j].Branch
		}
		return servers[i].Port < servers[j].Port
	})

	return servers, nil
}

func extractPort(line string) int {
	// Use pre-compiled regex for :PORT (LISTEN) pattern
	matches := portRegex.FindStringSubmatch(line)
	if len(matches) > 1 {
		port, err := strconv.Atoi(matches[1])
		if err != nil {
			return 0
		}
		return port
	}

	// Fallback: try to extract from the 9th field
	fields := strings.Fields(line)
	if len(fields) >= 9 {
		parts := strings.Split(fields[8], ":")
		if len(parts) > 0 {
			port, err := strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return 0
			}
			return port
		}
	}

	return 0
}

func isDevPort(port int) bool {
	// Skip well-known system ports (< 1024)
	if port < 1024 {
		return false
	}
	// Accept all ports >= 3000
	if port >= 3000 {
		return true
	}
	// Accept specific common dev ports between 1024 and 3000
	if port == 2000 {
		return true
	}
	return false
}

// batchCheckGitRepos checks multiple directories for git repos in parallel
func batchCheckGitRepos(dirs map[string]bool) map[string]bool {
	results := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			isRepo := git.IsRepo(d)
			mu.Lock()
			results[d] = isRepo
			mu.Unlock()
		}(dir)
	}

	wg.Wait()
	return results
}

// batchGetGitInfo fetches git info for multiple directories in parallel
func batchGetGitInfo(dirs map[string]bool) map[string]gitInfo {
	results := make(map[string]gitInfo)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for dir := range dirs {
		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			info := getGitInfoParallel(d)
			mu.Lock()
			results[d] = info
			mu.Unlock()
		}(dir)
	}

	wg.Wait()
	return results
}

// batchGetProcessCWDs gets working directories for multiple PIDs in a single call
func batchGetProcessCWDs(pids []int) map[int]string {
	cwdMap := make(map[int]string)

	if len(pids) == 0 {
		return cwdMap
	}

	if platform.IsMacOS() {
		// macOS: use lsof with comma-separated PIDs
		pidStrs := make([]string, len(pids))
		for i, pid := range pids {
			pidStrs[i] = strconv.Itoa(pid)
		}
		pidList := strings.Join(pidStrs, ",")

		cmd := exec.Command("lsof", "-a", "-p", pidList, "-d", "cwd", "-Fn")
		output, err := cmd.Output()
		if err != nil {
			// If batch fails, fall back to individual lookups
			return fallbackGetCWDs(pids)
		}

		// Parse lsof output: format is "p<pid>\nn<path>\np<pid>\nn<path>..."
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		var currentPID int
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "p") {
				pidStr := strings.TrimPrefix(line, "p")
				pid, err := strconv.Atoi(pidStr)
				if err == nil {
					currentPID = pid
				}
			} else if strings.HasPrefix(line, "n") && currentPID != 0 {
				cwd := strings.TrimPrefix(line, "n")
				cleaned, err := filepath.Abs(cwd)
				if err == nil {
					cwdMap[currentPID] = cleaned
				}
				currentPID = 0 // Reset after processing
			}
		}
	} else {
		// Linux: read from /proc/<pid>/cwd for each PID (already fast)
		for _, pid := range pids {
			link := fmt.Sprintf("/proc/%d/cwd", pid)
			info, err := os.Lstat(link)
			if err != nil {
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				continue
			}

			cwd, err := os.Readlink(link)
			if err != nil {
				continue
			}

			cleaned, err := filepath.Abs(cwd)
			if err != nil {
				continue
			}
			cwdMap[pid] = cleaned
		}
	}

	return cwdMap
}

// fallbackGetCWDs handles individual CWD lookups if batch fails
func fallbackGetCWDs(pids []int) map[int]string {
	cwdMap := make(map[int]string)
	for _, pid := range pids {
		cwd, err := getProcessCWD(pid)
		if err == nil && cwd != "" {
			cwdMap[pid] = cwd
		}
	}
	return cwdMap
}

func getProcessCWD(pid int) (string, error) {
	// Validate PID is within reasonable bounds
	if err := platform.ValidatePID(pid); err != nil {
		return "", err
	}

	if platform.IsMacOS() {
		// macOS
		cmd := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		scanner := bufio.NewScanner(strings.NewReader(string(output)))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "n") {
				cwd := strings.TrimPrefix(line, "n")
				// Clean and validate the path
				cleaned, err := filepath.Abs(cwd)
				if err != nil {
					return "", err
				}
				return cleaned, nil
			}
		}
	} else {
		// Linux - validate the symlink exists first
		link := fmt.Sprintf("/proc/%d/cwd", pid)
		info, err := os.Lstat(link)
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("not a symlink: %s", link)
		}

		cwd, err := os.Readlink(link)
		if err != nil {
			return "", err
		}

		// Clean and validate the resolved path
		cleaned, err := filepath.Abs(cwd)
		if err != nil {
			return "", err
		}
		return cleaned, nil
	}
	return "", fmt.Errorf("could not determine cwd")
}

// DetectProjectType identifies the project type by checking for marker files
func DetectProjectType(dir string) types.ProjectType {
	// Check for Go project
	if platform.FileExists(filepath.Join(dir, "go.mod")) || platform.FileExists(filepath.Join(dir, "go.sum")) {
		return types.ProjectTypeGo
	}

	// Check for Rust project
	if platform.FileExists(filepath.Join(dir, "Cargo.toml")) {
		return types.ProjectTypeRust
	}

	// Check for Node.js project
	if platform.FileExists(filepath.Join(dir, "package.json")) {
		return types.ProjectTypeNode
	}

	// Check for Python project
	if platform.FileExists(filepath.Join(dir, "requirements.txt")) ||
		platform.FileExists(filepath.Join(dir, "pyproject.toml")) ||
		platform.FileExists(filepath.Join(dir, "setup.py")) {
		return types.ProjectTypePython
	}

	// Check for Ruby project
	if platform.FileExists(filepath.Join(dir, "Gemfile")) {
		return types.ProjectTypeRuby
	}

	return types.ProjectTypeUnknown
}

// getGitInfoParallel fetches git repo name and branch in parallel using goroutines
func getGitInfoParallel(cwd string) gitInfo {
	var wg sync.WaitGroup
	info := gitInfo{}

	// Launch goroutine for repo name
	wg.Add(1)
	go func() {
		defer wg.Done()
		info.repo = git.GetRepoName(cwd)
	}()

	// Launch goroutine for branch
	wg.Add(1)
	go func() {
		defer wg.Done()
		info.branch = git.GetBranch(cwd)
	}()

	// Wait for both to complete
	wg.Wait()
	return info
}

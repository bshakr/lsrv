#!/bin/bash

# Colors for output
BOLD='\033[1m'
RESET='\033[0m'

# Show help message
if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Lists all running Rails, Node.js, Python, and Go servers across repos and worktrees."
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo ""
    echo "Output columns:"
    echo "  REPO     - Repository name (from git remote or directory name)"
    echo "  BRANCH   - Current git branch"
    echo "  PROCESS  - Process running the server (ruby, node, puma, etc.)"
    echo "  URL      - Clickable HTTP URL to access the server"
    exit 0
fi

# Check if lsof is available
if ! command -v lsof &> /dev/null; then
    echo "Error: lsof command not found. Please install it." >&2
    echo "" >&2
    if [[ "$(uname)" == "Darwin" ]]; then
        echo "On macOS, lsof should be pre-installed. If missing, reinstall Command Line Tools:" >&2
        echo "  xcode-select --install" >&2
    else
        echo "On Linux, install lsof:" >&2
        echo "  sudo apt-get install lsof  # Debian/Ubuntu" >&2
        echo "  sudo yum install lsof      # RHEL/CentOS" >&2
    fi
    exit 1
fi

# Helper function to check if directory is a git repository
is_git_repo() {
    local dir="$1"
    [ -z "$dir" ] && return 1
    [ -d "$dir/.git" ] || git -C "$dir" rev-parse --git-dir > /dev/null 2>&1
}

# Function to get repo name from directory
get_repo_name() {
    local dir="$1"

    # Validate input
    [ -z "$dir" ] && { echo "unknown"; return 1; }
    [ ! -d "$dir" ] && { echo "unknown"; return 1; }

    if is_git_repo "$dir"; then
        # Try to get repo name from git remote
        local remote_url=$(git -C "$dir" config --get remote.origin.url 2>/dev/null)
        if [ -n "$remote_url" ]; then
            # Extract repo name from URL
            basename "$remote_url" .git
        else
            # Fall back to directory name
            basename "$dir"
        fi
    else
        basename "$dir"
    fi
}

# Function to get git branch
get_git_branch() {
    local dir="$1"

    # Validate input
    [ -z "$dir" ] && { echo "N/A"; return 1; }
    [ ! -d "$dir" ] && { echo "N/A"; return 1; }

    if is_git_repo "$dir"; then
        git -C "$dir" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "N/A"
    else
        echo "N/A"
    fi
}

# Function to get working directory of a process
get_process_cwd() {
    local pid="$1"

    # Validate input
    [ -z "$pid" ] && return 1
    [[ ! "$pid" =~ ^[0-9]+$ ]] && return 1

    if [ "$(uname)" = "Darwin" ]; then
        # macOS
        lsof -a -p "$pid" -d cwd -Fn 2>/dev/null | grep '^n' | cut -c2-
    else
        # Linux
        readlink -f "/proc/$pid/cwd" 2>/dev/null
    fi
}

# Store results
declare -a results

# Delimiter for storing data (ASCII Unit Separator - very unlikely in normal text)
readonly DELIM=$'\x1F'

# Find all processes listening on TCP ports
# Look for common web server patterns
while IFS= read -r line; do
    # Extract PID from lsof output
    pid=$(echo "$line" | awk '{print $2}')
    [ -z "$pid" ] && continue

    # Extract command name
    command=$(echo "$line" | awk '{print $1}')
    [ -z "$command" ] && continue

    # Extract port more robustly - look for :PORT in the line
    port=$(echo "$line" | grep -oE ':([0-9]+) \(LISTEN\)' | grep -oE '[0-9]+' | head -1)
    # Fallback to field-based extraction if pattern didn't match
    [ -z "$port" ] && port=$(echo "$line" | awk '{print $9}' | grep -oE '[0-9]+$')
    # Skip if we still couldn't extract port
    [ -z "$port" ] && continue

    # Get working directory for this process
    cwd=$(get_process_cwd "$pid")

    # Skip if we couldn't get working directory (separate conditions for clarity)
    [ -z "$cwd" ] && continue
    [ ! -d "$cwd" ] && continue

    # Get repo name and branch
    repo=$(get_repo_name "$cwd")
    branch=$(get_git_branch "$cwd")

    # Store result with safer delimiter
    results+=("${repo}${DELIM}${branch}${DELIM}${command}${DELIM}http://localhost:${port}")

done < <(lsof -iTCP -sTCP:LISTEN -n -P 2>/dev/null | grep -E '(ruby|rails|puma|node|npm|yarn|python|gunicorn|uvicorn|go|java)' | grep -v "grep")

# Sort and deduplicate results
IFS=$'\n' sorted=($(printf '%s\n' "${results[@]}" | sort -u))

# Print header
printf "${BOLD}%-30s %-30s %-15s %-30s${RESET}\n" "REPO" "BRANCH" "PROCESS" "URL"
printf "%.0s-" {1..107}
echo

# Print results
if [ ${#sorted[@]} -eq 0 ]; then
    echo "No running web servers found."
else
    for result in "${sorted[@]}"; do
        IFS="$DELIM" read -r repo branch process url <<< "$result"
        printf "%-30s %-30s %-15s %-30s\n" "$repo" "$branch" "$process" "$url"
    done
fi

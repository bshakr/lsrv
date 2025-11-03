# lsrv - List your local running servers

A bash script to list all running servers across your repos and worktrees.

## Usage

Run the script from anywhere:

```bash
./lsrv.sh
```

Create a convenient alias:

```bash
# Add to ~/.zshrc or ~/.bashrc
alias lsrv="/path/to/lsrv/lsrv.sh"

# Then run from anywhere:
lsrv
```

## Output

The script displays a table with:

- **REPO**: Repository name (from git remote or directory name)
- **BRANCH**: Current git branch
- **PROCESS**: The process running the server (ruby, node, puma, etc.)
- **URL**: Clickable HTTP URL to access the server

Example:

```
REPO                           BRANCH                         PROCESS         URL
-----------------------------------------------------------------------------------------------------------
project_a                      main                           ruby            http://localhost:3000
project_a                      feature-branch                 node            http://localhost:3001
project_b                      develop                        ruby            http://localhost:4000
```

## How It Works

1. Uses `lsof` to find all processes listening on TCP ports
2. Filters for common web server processes:
   - Ruby/Rails (ruby, rails, puma)
   - Node.js (node, npm, yarn)
   - Python (python, gunicorn, uvicorn)
   - Go (go)
   - Java (java)
3. Extracts the working directory for each process
4. Gets the git branch and repo name from that directory
5. Displays results in a sorted table

## Requirements

- macOS or Linux
- `lsof` command (pre-installed on macOS)
- Git repositories for branch detection

## Features

- ✅ Works with any method of starting servers (rails s, bundle exec, npm start, yarn dev, etc.)
- ✅ Detects servers running in git worktrees
- ✅ Supports Rails, Node.js, Python, Go, and Java servers
- ✅ Robust port detection with fallback mechanisms
- ✅ Input validation and error handling
- ✅ Handles special characters in repo and branch names
- ✅ Help flag (`--help` or `-h`) for usage information

## License

MIT License - see [LICENSE](LICENSE) file for details.

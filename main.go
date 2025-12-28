package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/bassemshaker/lsrv/internal/detector"
	"github.com/bassemshaker/lsrv/internal/formatter"
	"github.com/bassemshaker/lsrv/internal/platform"
	"github.com/felixge/fgprof"
)

const version = "0.3.0"

func main() {
	// CLI flags
	helpFlag := flag.Bool("help", false, "Show help message")
	flag.BoolVar(helpFlag, "h", false, "Show help message (shorthand)")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.BoolVar(versionFlag, "v", false, "Show version information (shorthand)")
	profileFlag := flag.String("profile", "", "Write fgprof profile to file (e.g., --profile=lsrv.prof)")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("lsrv version %s\n", version)
		os.Exit(0)
	}

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Start profiling if requested
	if *profileFlag != "" {
		f, err := os.Create(*profileFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not create profile file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()

		// Start fgprof (captures wall-clock time including I/O waits)
		stopProfile := fgprof.Start(f, fgprof.FormatPprof)
		defer stopProfile()

		fmt.Fprintf(os.Stderr, "Profiling enabled, writing to %s\n", *profileFlag)
	}

	// Check if lsof is available
	if !commandExists("lsof") {
		printLsofError()
		os.Exit(1)
	}

	servers, err := detector.FindServers()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: finding servers: %v\n", err)
		os.Exit(1)
	}

	formatter.PrintResults(servers)

	if *profileFlag != "" {
		fmt.Fprintf(os.Stderr, "Profile written to %s\n", *profileFlag)
		fmt.Fprintf(os.Stderr, "Analyze with: go tool pprof -http=:8080 %s\n", *profileFlag)
	}
}

func printHelp() {
	fmt.Printf("lsrv version %s\n", version)
	fmt.Println("")
	fmt.Println("Usage: lsrv [OPTIONS]")
	fmt.Println("")
	fmt.Println("Lists all running web servers across repos and worktrees.")
	fmt.Println("")
	fmt.Println("Supported languages/frameworks:")
	fmt.Println("  Ruby (rails, puma), Node.js (node, npm, yarn), Python (gunicorn, uvicorn),")
	fmt.Println("  Go, Java, PHP (php-fpm, apache2, httpd), Rust (cargo), .NET (dotnet, kestrel),")
	fmt.Println("  Deno, Bun, Elixir/Phoenix (beam.smp, mix)")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  -h, --help           Show this help message")
	fmt.Println("  -v, --version        Show version information")
	fmt.Println("  --profile=FILE       Write performance profile to FILE for analysis")
	fmt.Println("")
	fmt.Println("Output columns:")
	fmt.Println("  REPO     - Repository name (from git remote or directory name)")
	fmt.Println("  BRANCH   - Current git branch")
	fmt.Println("  PROCESS  - Process running the server with icon (💎 ruby, ⬢ node, 🐹 go, etc.)")
	fmt.Println("  PID      - Process ID")
	fmt.Println("  URL      - Clickable HTTP URL to access the server")
}

func printLsofError() {
	fmt.Fprintln(os.Stderr, "error: lsof command not found, please install it")
	fmt.Fprintln(os.Stderr, "")
	if platform.IsMacOS() {
		fmt.Fprintln(os.Stderr, "On macOS, lsof should be pre-installed. If missing, reinstall Command Line Tools:")
		fmt.Fprintln(os.Stderr, "  xcode-select --install")
	} else {
		fmt.Fprintln(os.Stderr, "On Linux, install lsof:")
		fmt.Fprintln(os.Stderr, "  sudo apt-get install lsof  # Debian/Ubuntu")
		fmt.Fprintln(os.Stderr, "  sudo yum install lsof      # RHEL/CentOS")
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

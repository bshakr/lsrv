package formatter

import (
	"fmt"
	"strings"

	"github.com/bassemshaker/lsrv/internal/detector"
	"github.com/bassemshaker/lsrv/internal/types"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// PrintResults outputs the servers in a formatted table
func PrintResults(servers []types.Server) {
	if len(servers) == 0 {
		fmt.Println("No running web servers found.")
		return
	}

	printRoundedTable(servers)
}

func getProcessIcon(process string, cwd string) string {
	// First check process name
	switch process {
	case "ruby", "rails", "puma":
		return ""
	case "node", "npm", "yarn":
		return "⬢"
	case "python", "gunicorn", "uvicorn":
		return "🐍"
	case "go":
		return ""
	case "java":
		return ""
	case "php", "php-fpm", "apache2", "httpd":
		return "🐘"
	case "cargo":
		return "" // Nerd Fonts Rust icon
	case "dotnet", "kestrel":
		return "" // Nerd Fonts C# icon
	case "bun":
		return "🍞"
	case "elixir", "beam.smp", "mix":
		return "" // Nerd Fonts Elixir icon
	}

	// If process name didn't match, check project type from directory
	projectType := detector.DetectProjectType(cwd)
	switch projectType {
	case types.ProjectTypeGo:
		return ""
	case types.ProjectTypeRust:
		return ""
	case types.ProjectTypeNode:
		return "⬢"
	case types.ProjectTypePython:
		return "🐍"
	case types.ProjectTypeRuby:
		return ""
	}

	// Default fallback
	return "🌐"
}

// ============================================================================
// TABLE RENDERING
// ============================================================================

// printRoundedTable renders the table with rounded borders
func printRoundedTable(servers []types.Server) {
	rows := serversToRows(servers)

	// Header style - bold, white text
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Padding(0, 2)

	// Cell style - normal weight
	cellStyle := lipgloss.NewStyle().
		Padding(0, 2)

	// Create table with rounded borders
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))).
		Headers("REPO", "BRANCH", "PROCESS", "URL").
		StyleFunc(func(row, col int) lipgloss.Style {
			// Use table.HeaderRow constant for header detection
			if row == table.HeaderRow {
				return headerStyle
			}

			// Data rows
			if row < 0 || row >= len(servers) {
				return cellStyle
			}

			return getCellStyle(servers[row], col, cellStyle)
		}).
		Rows(rows...)

	fmt.Println(t)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// serversToRows converts servers to table row format
func serversToRows(servers []types.Server) [][]string {
	rows := make([][]string, len(servers))
	for i, server := range servers {
		icon := getProcessIcon(server.Process, server.CWD)
		url := fmt.Sprintf("http://localhost:%d", server.Port)
		rows[i] = []string{
			server.Repo,
			server.Branch,
			fmt.Sprintf("%s %s", icon, server.Process),
			url,
		}
	}
	return rows
}

// getCellStyle returns the appropriate lipgloss style for a cell
func getCellStyle(server types.Server, col int, baseStyle lipgloss.Style) lipgloss.Style {
	// Define colors for process types (used as fallback)
	colors := map[string]lipgloss.Color{
		"ruby":   lipgloss.Color("1"), // Red
		"node":   lipgloss.Color("2"), // Green
		"python": lipgloss.Color("3"), // Yellow
		"cargo":  lipgloss.Color("1"), // Red for Rust
	}

	// Color the process column based on type
	if col == 2 {
		// Detect color based on project type or process name
		projectType := detector.DetectProjectType(server.CWD)
		switch projectType {
		case types.ProjectTypeGo:
			return baseStyle.Foreground(lipgloss.Color("6")) // Cyan
		case types.ProjectTypeRust:
			return baseStyle.Foreground(lipgloss.Color("1")) // Red
		case types.ProjectTypeNode:
			return baseStyle.Foreground(lipgloss.Color("2")) // Green
		case types.ProjectTypePython:
			return baseStyle.Foreground(lipgloss.Color("3")) // Yellow
		case types.ProjectTypeRuby:
			return baseStyle.Foreground(lipgloss.Color("1")) // Red
		}

		// Fallback to process name matching using standard library
		for processType, color := range colors {
			if strings.Contains(server.Process, processType) {
				return baseStyle.Foreground(color)
			}
		}

		// Default for unknown processes
		return baseStyle.Foreground(lipgloss.Color("7")) // White
	}

	// Color URLs blue
	if col == 3 {
		return baseStyle.Foreground(lipgloss.Color("4")) // Blue
	}

	// Return base style (already has UnsetBold from cellStyle)
	return baseStyle
}

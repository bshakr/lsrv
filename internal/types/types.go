package types

// Server represents a running development server
type Server struct {
	Repo    string
	Branch  string
	Process string
	Port    int
	PID     int
	CWD     string
}

// ProjectType represents the detected project type
type ProjectType string

const (
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeRust    ProjectType = "rust"
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeRuby    ProjectType = "ruby"
	ProjectTypeJava    ProjectType = "java"
	ProjectTypePHP     ProjectType = "php"
	ProjectTypeDotNet  ProjectType = "dotnet"
	ProjectTypeDeno    ProjectType = "deno"
	ProjectTypeBun     ProjectType = "bun"
	ProjectTypeElixir  ProjectType = "elixir"
	ProjectTypeUnknown ProjectType = "unknown"
)

// Package stdiorunner provides stdio-based command execution for skill running.
// It implements secure script execution with configurable runtimes, security profiles,
// and I/O handling. The package supports multiple runtime environments including bash,
// Python, Node.js, and binary execution with process isolation and security constraints.
package stdiorunner

import (
	"os"
	"path/filepath"

	"github.com/tansive/tansive/internal/tangent/config"
)

// RunnerConfig holds configuration for the stdio runner.
// Contains script directory and runtime settings for command execution.
type RunnerConfig struct {
	ScriptDir string `json:"scriptDir"` // directory containing executable scripts
}

var runnerConfig *RunnerConfig

// Init initializes the stdio runner with configuration from the main config.
// Sets up the script directory and runtime environment for command execution.
func Init() {
	runnerConfig = &RunnerConfig{
		ScriptDir: config.Config().StdioRunner.ScriptDir,
	}
}

// TestInit initializes the stdio runner for testing purposes.
// Overrides the script directory with the project's skillset_scripts directory.
func TestInit() {
	// Override the default script directory with project root path
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Check if we're already in the project root by looking for go.mod
	projectRoot := wd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			panic("could not find project root (go.mod)")
		}
		projectRoot = parent
	}
	if runnerConfig == nil {
		runnerConfig = &RunnerConfig{}
	}

	runnerConfig.ScriptDir = filepath.Join(projectRoot, "examples", "skillset_scripts")
}

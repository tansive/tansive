// Package shellcommandrunner provides a runner for executing shell commands and scripts.
// It supports multiple runtime environments and configurable security settings.
// The package requires valid io.Writer implementations for output handling.
package stdiorunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/h2non/filetype"
	"github.com/rs/zerolog/log"

	"github.com/mitchellh/mapstructure"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

// runner implements the runners.Runner interface.
// It manages command execution lifecycle and output streaming.
type runner struct {
	sessionID   string
	config      Config
	homeDirPath string
	writers     []*tangentcommon.IOWriters
}

func (r *runner) ID() string {
	return catcommon.StdioRunnerID
}

func (r *runner) AddWriters(writers ...*tangentcommon.IOWriters) {
	r.writers = append(r.writers, writers...)
}

// New creates a new runner with the given configuration.
// The configuration must be valid JSON that can be unmarshaled into a Config.
// The writers must provide non-nil io.Writer implementations for both stdout and stderr.
// Returns an error if the configuration is invalid or writers are not properly configured.
func New(ctx context.Context, sessionID string, configMap map[string]any, writers ...*tangentcommon.IOWriters) (*runner, apperrors.Error) {
	var config Config

	for _, writer := range writers {
		if writer == nil || writer.Out == nil || writer.Err == nil {
			return nil, ErrInvalidWriters
		}
	}

	if err := mapstructure.Decode(configMap, &config); err != nil {
		return nil, ErrInvalidConfig
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	runner := &runner{
		sessionID: sessionID,
		config:    config,
		writers:   writers,
	}

	return runner, nil
}

// Run executes the configured command.
// The context can be used to cancel the execution.
// Returns an error if execution fails or is cancelled.
func (r *runner) Run(ctx context.Context, args *api.SkillInputArgs) apperrors.Error {
	if args == nil {
		return ErrInvalidArgs.Msg("args is nil")
	}

	if r.config.Security.Type == SecurityTypeDefault {
		return r.runWithDefaultSecurity(ctx, args)
	}
	return ErrInvalidSecurity.Msg("security type not supported: " + string(r.config.Security.Type))
}

func (r *runner) runWithDefaultSecurity(ctx context.Context, args *api.SkillInputArgs) apperrors.Error {
	scriptPath := filepath.Join(runnerConfig.ScriptDir, filepath.Clean(r.config.Script))
	if !strings.HasPrefix(scriptPath, filepath.Clean(runnerConfig.ScriptDir)+string(os.PathSeparator)) {
		return ErrInvalidScript.Msg("script path escapes trusted directory")
	}

	if _, err := os.Stat(scriptPath); err != nil {
		return ErrInvalidScript.Msg("script not found: " + err.Error())
	}

	homeDirPath := filepath.Join(os.TempDir(), r.sessionID)
	if err := os.MkdirAll(homeDirPath, 0755); err != nil {
		return ErrExecutionFailed.Msg("failed to create home directory: " + err.Error())
	}

	r.homeDirPath = homeDirPath

	// Normalize line endings for scripts to handle Windows \r\n
	// This is especially important for shell scripts but can help with other script types too
	normalizedScriptPath := scriptPath
	if r.shouldNormalizeLineEndings(scriptPath) {
		ext := filepath.Ext(scriptPath)
		normalizedPath := filepath.Join(homeDirPath, "normalized_script"+ext)
		if err := r.normalizeLineEndings(scriptPath, normalizedPath); err != nil {
			// Check if the error is specifically about dos2unix not being available
			if errors.Is(err, ErrDos2UnixNotAvailable) {
				log.Ctx(ctx).Warn().Msg("dos2unix not available, proceeding with original script")
			} else {
				// For other errors, fail the execution
				return ErrExecutionFailed.Msg("failed to normalize script line endings: " + err.Error())
			}
		} else {
			normalizedScriptPath = normalizedPath
		}
	}

	wrappedScriptPath := filepath.Join(homeDirPath, "wrapped.sh")
	if err := r.writeWrappedScript(wrappedScriptPath, normalizedScriptPath, args); err != nil {
		return ErrExecutionFailed.Msg("failed to create wrapped script: " + err.Error())
	}
	if err := os.Chmod(wrappedScriptPath, 0755); err != nil {
		return ErrExecutionFailed.Msg("failed to set permissions on wrapped script: " + err.Error())
	}

	baseEnv := os.Environ()
	env := appendOrReplaceEnv(baseEnv, "HOME", homeDirPath)
	for k, v := range r.config.Env {
		env = appendOrReplaceEnv(env, k, v)
	}

	outWriter := NewWriter(StdoutWriter, r.writers...)
	errWriter := NewWriter(StderrWriter, r.writers...)

	cmd := exec.CommandContext(ctx, "/bin/bash", wrappedScriptPath)
	cmd.Dir = homeDirPath
	cmd.Env = env
	// cmd.Stdout = outWriter
	// cmd.Stderr = errWriter

	// if err := cmd.Run(); err != nil {
	// 	return ErrExecutionFailed.Msg("command failed: " + err.Error())
	// }

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return ErrExecutionFailed.Msg("failed to get stdout pipe: " + err.Error())
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return ErrExecutionFailed.Msg("failed to get stderr pipe: " + err.Error())
	}

	if err := cmd.Start(); err != nil {
		return ErrExecutionFailed.Msg("startcommand failed: " + err.Error())
	}
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		io.Copy(outWriter, stdoutPipe)
	}()

	go func() {
		defer wg.Done()
		io.Copy(errWriter, stderrPipe)
	}()

	err = cmd.Wait()
	wg.Wait()

	if err != nil {
		return ErrExecutionFailed.Msg("command execution failed: " + err.Error())
	}

	return nil
}

func (r *runner) writeWrappedScript(wrappedPath, scriptPath string, args *api.SkillInputArgs) error {
	jsonArgs, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("could not normalize JSON args: %w", err)
	}

	escapedArgs := strings.ReplaceAll(string(jsonArgs), "'", "'\\''")

	var content string
	if r.config.Runtime == RuntimeBinary {
		isBinary, err := isBinaryExecutable(scriptPath)
		if err != nil {
			return fmt.Errorf("failed to check if script is binary: %w", err)
		}
		if !isBinary {
			return fmt.Errorf("script is not a binary: %s", scriptPath)
		}

		content = fmt.Sprintf(`#!/bin/bash
set -euo pipefail

exec '%s' '%s'
`, scriptPath, escapedArgs)
	} else {
		runtimeCmd, err := resolveRuntimeCommand(r.config.Runtime)
		if err != nil {
			return fmt.Errorf("unsupported runtime: %w", err)
		}

		content = fmt.Sprintf(`#!/bin/bash
set -euo pipefail

exec %s '%s' '%s'
`, strings.Join(runtimeCmd, " "), scriptPath, escapedArgs)
	}

	return os.WriteFile(wrappedPath, []byte(content), 0644)
}

func resolveRuntimeCommand(runtime Runtime) ([]string, error) {
	switch runtime {
	case RuntimeBash:
		return []string{"/bin/bash"}, nil
	case RuntimePython:
		return []string{"python3", "-u"}, nil
	case RuntimeNode:
		return []string{"node"}, nil
	case RuntimeNPX:
		return []string{"npx"}, nil
	case RuntimeNPM:
		return []string{"npm"}, nil
	case RuntimeBinary:
		return nil, nil
	default:
		return nil, fmt.Errorf("invalid runtime: %s", runtime)
	}
}

func appendOrReplaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func (r *runner) GetHomeDirPath() string {
	return r.homeDirPath
}

// Known executable binary types
var binaryTypes = map[string]bool{
	"elf":   true, // Linux
	"macho": true, // macOS
	"pe":    true, // Windows
}

func isBinaryExecutable(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Read first 261 bytes (enough for filetype sniffing)
	header := make([]byte, 261)
	_, err = file.Read(header)
	if err != nil {
		return false, err
	}

	kind, err := filetype.Match(header)
	if err != nil {
		return false, err
	}
	if kind == filetype.Unknown {
		return false, nil
	}

	return binaryTypes[kind.Extension], nil
}

// shouldNormalizeLineEndings determines if a script file should have its line endings normalized
// This is primarily for shell scripts but can be extended for other script types
func (r *runner) shouldNormalizeLineEndings(scriptPath string) bool {
	// Normalize for common script extensions that might have Windows line endings
	ext := strings.ToLower(filepath.Ext(scriptPath))
	switch ext {
	case ".sh", ".bash", ".zsh", ".ksh":
		return true
	case ".py", ".js", ".ts", ".rb", ".pl", ".php":
		// These can also benefit from line ending normalization
		return true
	default:
		return false
	}
}

// hasDos2Unix checks if dos2unix is available in PATH
func hasDos2Unix() bool {
	_, err := exec.LookPath("dos2unix")
	return err == nil
}

// normalizeLineEndings converts Windows line endings (\r\n) to Unix line endings (\n)
// This is necessary for shell scripts that may have been edited on Windows
func (r *runner) normalizeLineEndings(source, target string) apperrors.Error {
	// First, copy the source file to the target location
	sourceContent, err := os.ReadFile(source)
	if err != nil {
		return ErrExecutionFailed.Msg("failed to read source file: " + err.Error())
	}

	if err := os.WriteFile(target, sourceContent, 0644); err != nil {
		return ErrExecutionFailed.Msg("failed to write target file: " + err.Error())
	}

	if !hasDos2Unix() {
		return ErrDos2UnixNotAvailable.Msg("dos2unix not found in PATH")
	}

	// Use dos2unix to convert line endings in place
	cmd := exec.Command("dos2unix", target)
	if err := cmd.Run(); err != nil {
		return ErrDos2UnixNotAvailable.Msg("dos2unix command failed: " + err.Error())
	}

	return nil
}

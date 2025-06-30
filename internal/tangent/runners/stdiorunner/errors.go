package stdiorunner

import "github.com/tansive/tansive/internal/common/apperrors"

// Error definitions for the package.
// All errors are derived from ErrShellCommandRunnerError.
var (
	// ErrShellCommandRunnerError is the base error for the package.
	// All other errors in this package are derived from this error.
	ErrShellCommandRunnerError = apperrors.New("shell command runner error")

	// ErrInvalidRuntime is returned for unsupported runtimes.
	// Occurs when Config.Runtime is not in ValidRunTimes.
	ErrInvalidRuntime = ErrShellCommandRunnerError.New("invalid runtime")

	// ErrInvalidSecurity is returned for invalid security settings.
	// Occurs when Config.Security.Type is not in ValidSecurityTypes.
	ErrInvalidSecurity = ErrShellCommandRunnerError.New("invalid security")

	// ErrInvalidScript is returned for invalid script paths.
	// Occurs when Config.Script is empty.
	ErrInvalidScript = ErrShellCommandRunnerError.New("invalid script")

	// ErrInvalidRuntimeConfig is returned for invalid runtime settings.
	// Occurs when Config.RuntimeConfig contains invalid values for the specified runtime.
	ErrInvalidRuntimeConfig = ErrShellCommandRunnerError.New("invalid runtime config")

	// ErrInvalidEnv is returned for invalid environment variables.
	// Occurs when environment variables cannot be set for the process.
	ErrInvalidEnv = ErrShellCommandRunnerError.New("invalid env")

	// ErrInvalidVersion is returned for incompatible versions.
	// Occurs when Config.Version does not satisfy the semver constraint.
	ErrInvalidVersion = ErrShellCommandRunnerError.New("invalid version")

	// ErrInvalidConfig is returned for invalid configurations.
	// Occurs when the JSON configuration cannot be unmarshaled into a Config.
	ErrInvalidConfig = ErrShellCommandRunnerError.New("invalid config")

	// ErrInvalidWriters is returned for invalid I/O writers.
	// Occurs when either IOWriters.Out or IOWriters.Err is nil.
	ErrInvalidWriters = ErrShellCommandRunnerError.New("invalid writers")

	// ErrExecutionFailed is returned when the command fails to execute.
	// Occurs when the command returns a non-zero exit code.
	ErrExecutionFailed = ErrShellCommandRunnerError.New("execution failed")

	// ErrInvalidHomeDir is returned for invalid home directories.
	// Occurs when the home directory cannot be created.
	ErrInvalidHomeDir = ErrShellCommandRunnerError.New("invalid home directory")

	// ErrInvalidArgs is returned for invalid arguments.
	// Occurs when the arguments are nil.
	ErrInvalidArgs = ErrShellCommandRunnerError.New("invalid args")
)

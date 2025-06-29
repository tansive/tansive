package stdiorunner

import (
	"github.com/tansive/tansive/internal/common/apperrors"
)

// Config defines the configuration for command execution.
// All fields are required except RuntimeConfig and Env.
//
// Example:
//
//	{
//	  "version": "0.1.0",
//	  "runtime": "bash",
//	  "runtimeConfig": {
//	    "key1": "value1"
//	  },
//	  "env": {
//	    "VAR1": "value1"
//	  },
//	  "script": "my-script.sh",
//	  "security": {
//	    "type": "default"
//	  }
//	}
type Config struct {
	Version       string            `json:"version"`       // must be compatible with current version
	Runtime       Runtime           `json:"runtime"`       // must be one of ValidRunTimes
	RuntimeConfig map[string]any    `json:"runtimeConfig"` // optional runtime-specific settings
	Env           map[string]string `json:"env"`           // optional environment variables
	Script        string            `json:"script"`        // must be non-empty
	Security      Security          `json:"security"`      // defaults to "default" if empty
}

// Runtime specifies the command execution environment.
// The value must be one of the constants defined below.
type Runtime string

const (
	// RuntimeBash uses /bin/bash for shell script execution.
	RuntimeBash Runtime = "bash"

	// RuntimePython uses python interpreter for Python script execution.
	RuntimePython Runtime = "python"

	// RuntimeNode uses node.js for JavaScript execution.
	RuntimeNode Runtime = "node"

	// RuntimeNPX uses npx for Node.js package execution.
	RuntimeNPX Runtime = "npx"

	// RuntimeNPM uses npm for Node.js package management.
	RuntimeNPM Runtime = "npm"

	// RuntimeBinary uses binary for direct executable execution.
	RuntimeBinary Runtime = "binary"
)

// SecurityType specifies the security profile for command execution.
// The value must be one of the constants defined below.
type SecurityType string

const (
	// SecurityTypeDefault provides basic process isolation, not intended for production use.
	SecurityTypeDefault SecurityType = "default"

	// SecurityTypeSandboxed provides enhanced security constraints.
	SecurityTypeSandboxed SecurityType = "sandboxed"
)

// Security defines the security settings for command execution.
// Type defaults to "default" if empty.
type Security struct {
	Type SecurityType `json:"type"` // must be one of ValidSecurityTypes
}

// ValidRunTimes defines the supported runtime environments.
// Only runtimes in this map are allowed in Config.Runtime.
var ValidRunTimes = map[Runtime]struct{}{
	RuntimeBash:   {},
	RuntimePython: {},
	RuntimeNode:   {},
	RuntimeNPX:    {},
	RuntimeNPM:    {},
	RuntimeBinary: {},
}

// ValidSecurityTypes defines the supported security profiles.
// Only security types in this map are allowed in Config.Security.Type.
var ValidSecurityTypes = map[SecurityType]struct{}{
	SecurityTypeDefault:   {},
	SecurityTypeSandboxed: {},
}

// Validate checks the configuration for validity.
// It verifies version compatibility, runtime validity, security settings,
// and required fields. Returns an error if any validation fails.
func (c *Config) Validate() apperrors.Error {
	if !IsVersionCompatible(c.Version) {
		return ErrInvalidVersion
	}

	if _, ok := ValidRunTimes[c.Runtime]; !ok {
		return ErrInvalidRuntime
	}

	if c.Security.Type == "" {
		c.Security.Type = SecurityTypeDefault
	} else if _, ok := ValidSecurityTypes[c.Security.Type]; !ok {
		return ErrInvalidSecurity
	}

	if c.Script == "" {
		return ErrInvalidScript
	}

	if c.Env == nil {
		c.Env = make(map[string]string)
	}

	return nil
}

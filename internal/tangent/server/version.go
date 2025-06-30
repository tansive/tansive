package server

import (
	"github.com/Masterminds/semver/v3"
)

// Version is the current version of the package.
// The version follows semantic versioning (MAJOR.MINOR.PATCH).
const Version = "0.1.0-alpha.1"

// versionConstraint defines the compatible version range.
// It accepts any version with the same major version and greater or equal minor version.
var versionConstraint *semver.Constraints

func init() {
	var err error
	versionConstraint, err = semver.NewConstraint("=" + Version)
	if err != nil {
		panic(err)
	}
}

// IsVersionCompatible reports whether the given version is compatible
// with the current version. The version must be a valid semantic version string.
// Returns false for invalid version strings.
func IsVersionCompatible(version string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false
	}
	return versionConstraint.Check(v)
}

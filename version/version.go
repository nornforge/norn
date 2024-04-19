package version

import (
	"github.com/Masterminds/semver/v3"
)

var VersionInput string

// GetProgramVersion returns the program version as a *semver.Version object.
// It retrieves the build metadata from the debug build info and constructs
// a semver version object using the provided VersionInput. If the VersionInput
// is invalid, it creates a development version with build metadata.
// If the build metadata is available, it sets the metadata for the version object.
// Returns the constructed version object.
func GetProgramVersion() *semver.Version {
	version, err := semver.NewVersion(VersionInput)
	if err != nil {
		version = semver.New(0, 0, 0, "devel", "")
	}
	return version
}

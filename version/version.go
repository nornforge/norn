package version

import "fmt"

const (
	Major = 0
	Minor = 0
	Patch = 3
)

var ProgramVersion = fmt.Sprintf("v%d.%d.%d", Major, Minor, Patch)

package version

import "fmt"

const (
	Major = 2
	Minor = 2
	Patch = 0
)

var ProgramVersion = fmt.Sprintf("v%d.%d.%d", Major, Minor, Patch)

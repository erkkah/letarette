package protocol

import (
	"fmt"
	"strconv"
	"strings"
)

// Semver is used for protocol versioning
type Semver struct {
	major    int
	minor    int
	revision int
}

// ParseSemver creates a Semver struct from a string
func ParseSemver(version string) (result Semver, err error) {
	parts := strings.Split(version, ".")

	switch len(parts) {
	case 3:
		result.revision, err = strconv.Atoi(parts[2])
		if err != nil {
			return
		}
		fallthrough
	case 2:
		result.minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return
		}
		fallthrough
	case 1:
		result.major, err = strconv.Atoi(parts[0])
		if err != nil {
			return
		}
	default:
		err = fmt.Errorf("Invalid version string")
	}
	return
}

func (v Semver) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.revision)
}

// CompatibleWith returns true when major versions match
func (v Semver) CompatibleWith(other Semver) bool {
	return v.major == other.major
}

// NewerThan returns true when this version is newer than the other
func (v Semver) NewerThan(other Semver) bool {
	return v.major > other.major ||
		(v.major == other.major &&
			(v.minor > other.minor ||
				(v.minor == other.minor && v.revision > other.revision)))
}

// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		err = fmt.Errorf("invalid version string")
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

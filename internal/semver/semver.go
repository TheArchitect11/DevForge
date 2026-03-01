// Package semver provides semantic version parsing and comparison for
// DevForge's version pinning system.
package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a parsed semantic version (major.minor.patch).
type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease string
	Raw        string
}

// semverPattern matches versions like "1", "1.2", "1.2.3", "v1.2.3",
// "1.2.3-beta.1", or version strings embedded in tool output.
var semverPattern = regexp.MustCompile(`v?(\d+)(?:\.(\d+))?(?:\.(\d+))?(?:-([a-zA-Z0-9.]+))?`)

// Parse extracts a semantic version from a string. It handles common
// formats: "18", "18.0", "18.0.1", "v1.2.3", "v1.2.3-rc.1".
func Parse(raw string) (Version, error) {
	if raw == "" || strings.EqualFold(raw, "latest") {
		return Version{Raw: raw}, nil
	}

	matches := semverPattern.FindStringSubmatch(raw)
	if matches == nil {
		return Version{}, fmt.Errorf("cannot parse version from %q", raw)
	}

	v := Version{Raw: raw}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version in %q: %w", raw, err)
	}
	v.Major = major

	if matches[2] != "" {
		minor, err := strconv.Atoi(matches[2])
		if err != nil {
			return Version{}, fmt.Errorf("invalid minor version in %q: %w", raw, err)
		}
		v.Minor = minor
	}

	if matches[3] != "" {
		patch, err := strconv.Atoi(matches[3])
		if err != nil {
			return Version{}, fmt.Errorf("invalid patch version in %q: %w", raw, err)
		}
		v.Patch = patch
	}

	if matches[4] != "" {
		v.PreRelease = matches[4]
	}

	return v, nil
}

// String returns the version as "major.minor.patch[-prerelease]".
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		s += "-" + v.PreRelease
	}
	return s
}

// IsZero returns true if the version was not meaningfully set.
func (v Version) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.PreRelease == "" && v.Raw == ""
}

// Compare returns:
//
//	-1 if v < other
//	 0 if v == other
//	 1 if v > other
//
// Pre-release versions are considered less than the same version
// without a pre-release tag.
func (v Version) Compare(other Version) int {
	if v.Major != other.Major {
		return compareInt(v.Major, other.Major)
	}
	if v.Minor != other.Minor {
		return compareInt(v.Minor, other.Minor)
	}
	if v.Patch != other.Patch {
		return compareInt(v.Patch, other.Patch)
	}

	// Pre-release handling: no pre-release > has pre-release.
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1
	}
	if v.PreRelease < other.PreRelease {
		return -1
	}
	if v.PreRelease > other.PreRelease {
		return 1
	}
	return 0
}

// MajorMatches returns true if the major version of v matches the
// major version of other. This is used for version pinning where
// only the major version is specified (e.g. "18" matches "18.x.x").
func (v Version) MajorMatches(other Version) bool {
	return v.Major == other.Major
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	return 1
}

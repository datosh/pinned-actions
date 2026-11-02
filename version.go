package main

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
)

// versionPattern matches the numeric start of a version. Whatever follows is
// a pre-release or build suffix in one of several spellings (1.32.0-rc2,
// 1.32.0rc2, 1.32.0+build) and is ignored.
var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

// requireVersion checks that version (e.g. "v1.31.1" or "1.32.0-rc.1") is at
// least min, comparing major, minor and patch only. tool only names the
// program in error messages.
func requireVersion(tool, version string, min [3]int) error {
	match := versionPattern.FindStringSubmatch(version)
	if match == nil {
		return fmt.Errorf("unexpected %s version %q", tool, version)
	}
	var got [3]int
	for i := range got {
		n, err := strconv.Atoi(match[i+1])
		if err != nil {
			return fmt.Errorf("unexpected %s version %q: %w", tool, version, err)
		}
		got[i] = n
	}

	if slices.Compare(got[:], min[:]) < 0 {
		return fmt.Errorf("%s %s is too old; need at least %d.%d.%d", tool, version, min[0], min[1], min[2])
	}
	return nil
}

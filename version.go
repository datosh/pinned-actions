package main

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// requireVersion checks that version (e.g. "v1.31.1" or "1.32.0-rc.1") is at
// least min. tool only names the program in error messages.
func requireVersion(tool, version string, min [3]int) error {
	// Drop the leading v and any pre-release or build suffix.
	core, _, _ := strings.Cut(strings.TrimPrefix(version, "v"), "-")
	core, _, _ = strings.Cut(core, "+")

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return fmt.Errorf("unexpected %s version %q", tool, version)
	}
	var got [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
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

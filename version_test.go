package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequireVersion(t *testing.T) {
	min := [3]int{1, 31, 1}

	tests := []struct {
		version string
		wantErr bool
	}{
		{version: "1.31.1"},
		{version: "v1.31.1"},
		{version: "1.31.2"},
		{version: "1.40.0"},
		{version: "2.0.0"},
		{version: "1.32.0-rc2"},    // Cargo spelling of a pre-release
		{version: "1.32.0-rc.1"},   // semver spelling
		{version: "1.32.0rc2"},     // PyPI spelling
		{version: "1.32.0+build5"}, // build metadata
		{version: "1.31.0", wantErr: true},
		{version: "1.30.9", wantErr: true},
		{version: "0.99.99", wantErr: true},
		{version: "1.31", wantErr: true},
		{version: "dev", wantErr: true},
		{version: "one.two.three", wantErr: true},
		{version: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.version, func(t *testing.T) {
			err := requireVersion("tool", test.version, min)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

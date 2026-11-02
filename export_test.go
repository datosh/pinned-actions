package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseExportArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got := parseExportArgs("export-test", "default-out.json", nil)
		assert.Equal(t, exportArgs{ResultDir: "results", Out: "default-out.json"}, got)
	})

	t.Run("flags override defaults", func(t *testing.T) {
		got := parseExportArgs("export-test", "default-out.json", []string{"--result-dir=r", "--out=o.json"})
		assert.Equal(t, exportArgs{ResultDir: "r", Out: "o.json"}, got)
	})
}

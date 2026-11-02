package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(file, nil, 0o644))

	t.Run("existing file", func(t *testing.T) {
		got, err := exists(file)
		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("existing directory", func(t *testing.T) {
		got, err := exists(dir)
		assert.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("missing path", func(t *testing.T) {
		got, err := exists(filepath.Join(dir, "missing"))
		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("path below a regular file", func(t *testing.T) {
		// e.g. a repo where .github is a file, so .github/renovate.json is ENOTDIR.
		got, err := exists(filepath.Join(file, "child"))
		assert.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("permission error is returned, not a panic", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root ignores directory permissions")
		}
		locked := filepath.Join(dir, "locked")
		require.NoError(t, os.Mkdir(locked, 0o000))
		t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })

		got, err := exists(filepath.Join(locked, "child"))
		assert.Error(t, err)
		assert.False(t, got)
	})
}

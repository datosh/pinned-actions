package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBucketPinned(t *testing.T) {
	results := []Analysis{
		{Repository: "a/none", ActionsTotal: 0},
		{Repository: "a/all", ActionsTotal: 3, ActionsPinned: 3},
		{Repository: "b/all", ActionsTotal: 1, ActionsPinned: 1},
		{Repository: "a/some", ActionsTotal: 4, ActionsPinned: 1},
		{Repository: "a/unpinned", ActionsTotal: 2, ActionsPinned: 0},
		{Repository: "b/unpinned", ActionsTotal: 1, ActionsPinned: 0},
		{Repository: "c/unpinned", ActionsTotal: 5, ActionsPinned: 0},
	}

	got := bucketPinned(results)

	assert.Equal(t, PinnedBuckets{Pinned: 2, PartiallyPinned: 1, Unpinned: 3, NoActions: 1}, got)
	assert.Equal(t, PinnedBuckets{}, bucketPinned(nil))
}

func TestSetYear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pinned.json")

	// The file does not exist yet.
	require.NoError(t, setYear(path, "2025", PinnedBuckets{Pinned: 1, PartiallyPinned: 2, Unpinned: 3}))
	// A second year is added, the first one kept.
	require.NoError(t, setYear(path, "2026", PinnedBuckets{Pinned: 4, PartiallyPinned: 5, Unpinned: 6}))
	// An existing year is replaced.
	require.NoError(t, setYear(path, "2025", PinnedBuckets{Pinned: 7, PartiallyPinned: 8, Unpinned: 9}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	// Same layout as the hand-maintained file: four-space indent, newline at the end.
	assert.Equal(t, `{
    "2025": {
        "pinned": 7,
        "partially-pinned": 8,
        "unpinned": 9
    },
    "2026": {
        "pinned": 4,
        "partially-pinned": 5,
        "unpinned": 6
    }
}
`, string(got))
}

func TestSetYearWritesNewFieldsOnlyWhenSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pinned.json")
	require.NoError(t, setYear(path, "2025", PinnedBuckets{Pinned: 1}))
	require.NoError(t, setYear(path, "2026", PinnedBuckets{Pinned: 2, NoActions: 3, Scanned: "2026-11-02"}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"2025": {"pinned": 1, "partially-pinned": 0, "unpinned": 0},
		"2026": {"pinned": 2, "partially-pinned": 0, "unpinned": 0, "no-actions": 3, "scanned": "2026-11-02"}
	}`, string(got))
}

func TestSetYearRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pinned.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	assert.ErrorContains(t, setYear(path, "2026", PinnedBuckets{}), "parsing")
}

func TestParseExportArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got := parseExportArgs("export-test", "default-out.json", nil)
		assert.Equal(t, exportArgs{ResultDir: "results", Out: "default-out.json"}, got)
	})

	t.Run("flags override defaults", func(t *testing.T) {
		got := parseExportArgs("export-test", "default-out.json", []string{"--result-dir=r", "--out=o.json"})
		assert.Equal(t, exportArgs{ResultDir: "r", Out: "o.json"}, got)
	})

	t.Run("subcommand flags", func(t *testing.T) {
		var year string
		parseExportArgs("export-test", "o.json", []string{"--year=2026"}, func(flags *flag.FlagSet) {
			flags.StringVar(&year, "year", "", "")
		})
		assert.Equal(t, "2026", year)
	})
}

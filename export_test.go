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

	got := bucketPinned(results, nil)

	assert.Equal(t, PinnedBuckets{Pinned: 2, PartiallyPinned: 1, Unpinned: 3, NoActions: 1}, got)
	assert.Equal(t, PinnedBuckets{}, bucketPinned(nil, nil))
	assert.Nil(t, got.WithLockfile, "without lockfile data the count is unknown, not zero")
}

func TestBucketPinnedWithLockfiles(t *testing.T) {
	results := []Analysis{
		{Repository: "a/pinned-and-locked", ActionsTotal: 2, ActionsPinned: 2},
		{Repository: "a/locked", ActionsTotal: 2, ActionsPinned: 0},
		{Repository: "a/locked-partly-pinned", ActionsTotal: 3, ActionsPinned: 1},
		{Repository: "a/incomplete-lockfile", ActionsTotal: 2, ActionsPinned: 1},
		{Repository: "a/stale-lockfile", ActionsTotal: 2, ActionsPinned: 0},
		{Repository: "a/malformed-lockfile", ActionsTotal: 1, ActionsPinned: 0},
		{Repository: "a/no-lockfile", ActionsTotal: 1, ActionsPinned: 0},
		{Repository: "a/not-in-lockfile-data", ActionsTotal: 1, ActionsPinned: 0},
		{Repository: "a/lockfile-without-actions", ActionsTotal: 0},
	}
	lockfiles := map[string]LockfileResult{
		"a/pinned-and-locked":        {HasLockfile: true, Complete: true},
		"a/locked":                   {HasLockfile: true, Complete: true},
		"a/locked-partly-pinned":     {HasLockfile: true, Complete: true},
		"a/incomplete-lockfile":      {HasLockfile: true, Findings: []string{"not-pinned"}},
		"a/stale-lockfile":           {HasLockfile: true, Findings: []string{"stale"}},
		"a/malformed-lockfile":       {HasLockfile: true, Malformed: true},
		"a/no-lockfile":              {},
		"a/lockfile-without-actions": {HasLockfile: true},
	}

	got := bucketPinned(results, lockfiles)

	withLockfile, lockfileComplete := 7, 3
	assert.Equal(t, PinnedBuckets{
		Pinned:           1, // all SHA-pinned wins over a complete lockfile
		Locked:           2, // complete lockfile, not all SHA-pinned
		PartiallyPinned:  1, // a lockfile that is not complete does not help
		Unpinned:         4, // including repos missing from the lockfile data
		NoActions:        1,
		WithLockfile:     &withLockfile,
		LockfileComplete: &lockfileComplete,
	}, got)
}

func TestPinnedBucketsString(t *testing.T) {
	withLockfile, lockfileComplete := 4, 3

	assert.Equal(t,
		"pinned=221 locked=3 partially-pinned=210 unpinned=371 no-actions=185 with-lockfile=4 lockfile-complete=3 scanned=2026-11-02",
		PinnedBuckets{Pinned: 221, Locked: 3, PartiallyPinned: 210, Unpinned: 371, NoActions: 185, WithLockfile: &withLockfile, LockfileComplete: &lockfileComplete, Scanned: "2026-11-02"}.String())
	assert.Equal(t,
		"pinned=1 locked=0 partially-pinned=0 unpinned=0 no-actions=0 with-lockfile=unknown lockfile-complete=unknown scanned=",
		PinnedBuckets{Pinned: 1}.String())
}

func TestReadLockfileResults(t *testing.T) {
	dir := t.TempDir()

	got, err := readLockfileResults(dir)
	require.NoError(t, err)
	assert.Nil(t, got, "the file is optional")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "lockfile.jsonl"),
		[]byte("{\"repository\":\"a/b\",\"has_lockfile\":true,\"complete\":true}\n"), 0o644))
	got, err = readLockfileResults(dir)
	require.NoError(t, err)
	assert.Equal(t, map[string]LockfileResult{"a/b": {Repository: "a/b", HasLockfile: true, Complete: true}}, got)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "lockfile.jsonl"), []byte("not json\n"), 0o644))
	_, err = readLockfileResults(dir)
	assert.ErrorContains(t, err, "parsing")
}

func TestSetScan(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scans.json")

	// The file does not exist yet.
	require.NoError(t, setScan(path, ScanEntry{Label: "2026-12", PinnedBuckets: PinnedBuckets{Pinned: 1, PartiallyPinned: 2, Unpinned: 3}, Data: "december.tar.gz"}))
	// An earlier scan is added before the existing one.
	require.NoError(t, setScan(path, ScanEntry{Label: "2026-11", PinnedBuckets: PinnedBuckets{Pinned: 4, PartiallyPinned: 5, Unpinned: 6}}))
	// A scan with an existing label is replaced.
	require.NoError(t, setScan(path, ScanEntry{Label: "2026-12", PinnedBuckets: PinnedBuckets{Pinned: 7, PartiallyPinned: 8, Unpinned: 9}, Data: "december.tar.gz"}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	// Four-space indent, newline at the end.
	assert.Equal(t, `[
    {
        "label": "2026-11",
        "pinned": 4,
        "partially-pinned": 5,
        "unpinned": 6
    },
    {
        "label": "2026-12",
        "pinned": 7,
        "partially-pinned": 8,
        "unpinned": 9,
        "data": "december.tar.gz"
    }
]
`, string(got))
}

func TestSetScanWritesOptionalFieldsOnlyWhenSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scans.json")
	withLockfile, lockfileComplete := 21, 19
	require.NoError(t, setScan(path, ScanEntry{Label: "2025-06", PinnedBuckets: PinnedBuckets{Pinned: 1}}))
	require.NoError(t, setScan(path, ScanEntry{Label: "2026-11", PinnedBuckets: PinnedBuckets{
		Pinned: 2, Locked: 4, NoActions: 3, WithLockfile: &withLockfile, LockfileComplete: &lockfileComplete, Scanned: "2026-11-02",
	}}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"label": "2025-06", "pinned": 1, "partially-pinned": 0, "unpinned": 0},
		{"label": "2026-11", "pinned": 2, "partially-pinned": 0, "unpinned": 0, "locked": 4, "no-actions": 3,
		 "with-lockfile": 21, "lockfile-complete": 19, "scanned": "2026-11-02"}
	]`, string(got))
}

func TestSetScanKeepsZeroLockfileCounts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scans.json")
	none := 0
	require.NoError(t, setScan(path, ScanEntry{Label: "2026-12", PinnedBuckets: PinnedBuckets{WithLockfile: &none, LockfileComplete: &none}}))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(got), `"with-lockfile": 0`, "0 differs from unknown")
	assert.Contains(t, string(got), `"lockfile-complete": 0`)
}

func TestSetScanRejectsInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scans.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o644))

	assert.ErrorContains(t, setScan(path, ScanEntry{Label: "2026-12"}), "parsing")
}

func TestWebResultsSkipsRepositoriesZizmorFailedOn(t *testing.T) {
	raw := []ZizmorResult{
		{Repository: "a/ok", UsesGHA: true, Findings: []ZizmorFinding{
			{Rule: "artipacked", Severity: "Low"},
			{Rule: "artipacked", Severity: "Low"},
		}},
		{Repository: "a/crashed", UsesGHA: true, Findings: []ZizmorFinding{}, Error: "zizmor exited with status 1"},
		{Repository: "a/no-actions"},
	}

	web, failed := webResults(raw, map[string]int{"a/ok": 10, "a/crashed": 20, "a/no-actions": 30})

	assert.Equal(t, 1, failed)
	assert.Equal(t, []ZizmorWebResult{
		{Repository: "a/ok", UsesGHA: true, Stars: 10, Rules: []ZizmorWebRule{{Rule: "artipacked", Severity: "Low", Count: 2}}},
		{Repository: "a/no-actions", Stars: 30, Rules: []ZizmorWebRule{}},
	}, web)
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

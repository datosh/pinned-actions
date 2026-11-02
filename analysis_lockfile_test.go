package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockfileAnalyzerFixtures(t *testing.T) {
	tests := []struct {
		name         string
		dir          string
		repo         string
		wantLockfile bool
		wantTotal    int
	}{
		{name: "locked", dir: "testdata/lockfile", repo: "locked", wantLockfile: true, wantTotal: 2},
		{name: "stale", dir: "testdata/lockfile", repo: "stale", wantLockfile: true, wantTotal: 2},
		{name: "not onboarded", dir: "testdata/lockfile", repo: "not-onboarded", wantLockfile: true, wantTotal: 2},
		{name: "malformed", dir: "testdata/lockfile", repo: "malformed", wantLockfile: true, wantTotal: 1},
		{name: "workflow without lockfile", dir: "testdata", repo: "pinned", wantTotal: 1},
		{name: "no workflows directory", dir: "testdata", repo: "no-workflows"},
		{name: "no .github directory", dir: "testdata", repo: "no-github"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NewLockfileAnalyzer().Analyze(context.Background(), test.dir, test.repo)
			require.NoError(t, err)

			assert.Equal(t, LockfileResult{
				Repository:     test.repo,
				HasLockfile:    test.wantLockfile,
				WorkflowsTotal: test.wantTotal,
			}, got)
		})
	}
}

func TestLockfileAnalyzerCountsOnlyWorkflowFiles(t *testing.T) {
	root := t.TempDir()
	workflows := filepath.Join(root, "o", "r", ".github", "workflows")
	require.NoError(t, os.MkdirAll(filepath.Join(workflows, "nested"), 0o755))
	for _, name := range []string{
		"a.yml", "b.yaml", // counted
		"notes.txt", "C.YML", "actions.lock", // not workflows
		"nested/d.yml", // not run by GitHub
	} {
		require.NoError(t, os.WriteFile(filepath.Join(workflows, name), nil, 0o644))
	}

	got, err := NewLockfileAnalyzer().Analyze(context.Background(), root, "o/r")
	require.NoError(t, err)

	assert.Equal(t, LockfileResult{Repository: "o/r", HasLockfile: true, WorkflowsTotal: 2}, got)
}

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockfileAnalyzerFixtures(t *testing.T) {
	tests := []struct {
		name string
		dir  string
		repo string
		want LockfileResult
	}{
		{
			name: "locked", dir: "testdata/lockfile", repo: "locked",
			want: LockfileResult{HasLockfile: true, Version: "v0.0.3", WorkflowsTotal: 2, WorkflowsLocked: 2},
		},
		{
			name: "stale", dir: "testdata/lockfile", repo: "stale",
			want: LockfileResult{HasLockfile: true, Version: "v0.0.3", WorkflowsTotal: 2, WorkflowsLocked: 2},
		},
		{
			name: "not onboarded", dir: "testdata/lockfile", repo: "not-onboarded",
			want: LockfileResult{HasLockfile: true, Version: "v0.0.3", WorkflowsTotal: 2, WorkflowsLocked: 1},
		},
		{
			name: "malformed", dir: "testdata/lockfile", repo: "malformed",
			want: LockfileResult{HasLockfile: true, WorkflowsTotal: 1, Malformed: true},
		},
		{
			name: "workflow without lockfile", dir: "testdata", repo: "pinned",
			want: LockfileResult{WorkflowsTotal: 1},
		},
		{name: "no workflows directory", dir: "testdata", repo: "no-workflows"},
		{name: "no .github directory", dir: "testdata", repo: "no-github"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := (&LockfileAnalyzer{}).Analyze(context.Background(), test.dir, test.repo)
			require.NoError(t, err)

			want := test.want
			want.Repository = test.repo
			assert.Equal(t, want, got)
		})
	}
}

func TestLockfileAnalyzerCountsOnlyWorkflowFiles(t *testing.T) {
	root := t.TempDir()
	workflows := filepath.Join(root, "o", "r", ".github", "workflows")
	require.NoError(t, os.MkdirAll(filepath.Join(workflows, "nested"), 0o755))
	for _, name := range []string{
		"a.yml", "b.yaml", // counted
		"notes.txt", "C.YML", // not workflows
		"nested/d.yml", // not run by GitHub
	} {
		require.NoError(t, os.WriteFile(filepath.Join(workflows, name), nil, 0o644))
	}
	// The lockfile lists a, a workflow that does not exist, and a nested one.
	lock := "version: 'v0.0.3'\nworkflows:\n  '.github/workflows/a.yml': []\n  '.github/workflows/gone.yml': []\n  '.github/workflows/nested/d.yml': []\n"
	require.NoError(t, os.WriteFile(filepath.Join(workflows, "actions.lock"), []byte(lock), 0o644))

	got, err := (&LockfileAnalyzer{}).Analyze(context.Background(), root, "o/r")
	require.NoError(t, err)

	assert.Equal(t, LockfileResult{
		Repository: "o/r", HasLockfile: true, Version: "v0.0.3", WorkflowsTotal: 2, WorkflowsLocked: 1,
	}, got)
}

func TestCheckLockfileCLIVersion(t *testing.T) {
	tests := []struct {
		output  string
		wantErr bool
	}{
		{output: "actions-lock version v0.4.1\n"},
		{output: "actions-lock version v0.4.2"},
		{output: "actions-lock version v0.10.0"},
		{output: "actions-lock version v1.0.0"},
		{output: "actions-lock version v0.4.0", wantErr: true},
		{output: "actions-lock version v0.3.9", wantErr: true},
		{output: "actions-lock version dev", wantErr: true},
		{output: "actions-lock v0.4.1", wantErr: true},
		{output: "v0.4.1", wantErr: true},
		{output: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.output, func(t *testing.T) {
			err := checkLockfileCLIVersion(test.output)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestNewLockfileAnalyzer runs the real CLI; it is skipped unless gh and the
// actions-lock extension are installed.
func TestNewLockfileAnalyzer(t *testing.T) {
	gh, err := exec.LookPath("gh")
	if err != nil {
		t.Skip("gh not on $PATH")
	}
	if err := exec.Command(gh, "actions-lock", "--version").Run(); err != nil {
		t.Skip("gh actions-lock not installed")
	}

	a, err := NewLockfileAnalyzer()

	require.NoError(t, err)
	assert.NotEmpty(t, a.gh)
}

func TestParseLockfile(t *testing.T) {
	workflows := map[string]bool{".github/workflows/ci.yml": true}

	t.Run("lists a workflow, ignores other sections", func(t *testing.T) {
		data := []byte("version: 'v0.0.3'\nworkflows:\n  '.github/workflows/ci.yml':\n    - 'a/b@v1'\ndependencies:\n  'a/b@v1':\n    ref: v1\n")
		version, locked, err := parseLockfile(data, workflows)
		require.NoError(t, err)
		assert.Equal(t, "v0.0.3", version)
		assert.Equal(t, 1, locked)
	})

	t.Run("workflow with an empty list is still onboarded", func(t *testing.T) {
		_, locked, err := parseLockfile([]byte("version: v0.0.3\nworkflows:\n  '.github/workflows/ci.yml': []\n"), workflows)
		require.NoError(t, err)
		assert.Equal(t, 1, locked)
	})

	for name, data := range map[string]string{
		"empty file":        "",
		"no version":        "workflows: {}\n",
		"invalid yaml":      "version: 'v0.0.3'\nworkflows: [unclosed\n",
		"wrong shape":       "version: v0.0.3\nworkflows:\n  - '.github/workflows/ci.yml'\n",
		"entries not lists": "version: v0.0.3\nworkflows:\n  '.github/workflows/ci.yml': nope\n",
	} {
		t.Run("malformed: "+name, func(t *testing.T) {
			_, _, err := parseLockfile([]byte(data), workflows)
			assert.Error(t, err)
		})
	}
}

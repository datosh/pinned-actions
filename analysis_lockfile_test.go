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

// lockfileAnalyzerOrSkip returns an analyzer backed by the real CLI, or skips
// the test unless gh and the actions-lock extension are installed.
func lockfileAnalyzerOrSkip(t *testing.T) *LockfileAnalyzer {
	t.Helper()
	gh, err := exec.LookPath("gh")
	if err != nil {
		t.Skip("gh not on $PATH")
	}
	if err := exec.Command(gh, "actions-lock", "--version").Run(); err != nil {
		t.Skip("gh actions-lock not installed")
	}
	a, err := NewLockfileAnalyzer()
	require.NoError(t, err)
	return a
}

func TestNewLockfileAnalyzer(t *testing.T) {
	assert.NotEmpty(t, lockfileAnalyzerOrSkip(t).gh)
}

func TestLockfileAnalyzerVerifiesWithCLI(t *testing.T) {
	a := lockfileAnalyzerOrSkip(t)

	tests := []struct {
		repo         string
		wantComplete bool
		wantMalform  bool
	}{
		{repo: "locked", wantComplete: true},
		{repo: "stale"},
		{repo: "not-onboarded"},
		{repo: "malformed", wantMalform: true},
	}
	for _, test := range tests {
		t.Run(test.repo, func(t *testing.T) {
			got, err := a.Analyze(context.Background(), "testdata/lockfile", test.repo)
			require.NoError(t, err)

			result, ok := got.(LockfileResult)
			require.True(t, ok)
			assert.Equal(t, test.wantComplete, result.Complete)
			assert.Equal(t, test.wantMalform, result.Malformed)
			if !test.wantComplete && !test.wantMalform {
				assert.Contains(t, result.Findings, "not-pinned", "a used action is missing from the lockfile")
			}
			if test.wantComplete {
				assert.Empty(t, result.Findings)
			}
		})
	}
}

func TestParseVerifyLocal(t *testing.T) {
	t.Run("real output for a repository without a lockfile", func(t *testing.T) {
		data, err := os.ReadFile("testdata/lockfile/verify-local-no-lockfile.json")
		require.NoError(t, err)

		complete, categories, err := parseVerifyLocal(data)

		require.NoError(t, err)
		assert.False(t, complete)
		assert.Equal(t, []string{"not-pinned", "sha-as-ref"}, categories, "distinct and sorted")
	})

	t.Run("complete", func(t *testing.T) {
		// Shape of the real output for this repository's own lockfile.
		complete, categories, err := parseVerifyLocal([]byte(`{"cli_version":"v0.4.1","findings":[],"lockfile_version":"v0.0.3","valid":true,"workflows":[]}`))

		require.NoError(t, err)
		assert.True(t, complete)
		assert.Empty(t, categories)
	})

	t.Run("findings may be null", func(t *testing.T) {
		complete, categories, err := parseVerifyLocal([]byte(`{"valid":false,"findings":null}`))

		require.NoError(t, err)
		assert.False(t, complete)
		assert.Empty(t, categories)
	})

	t.Run("not JSON", func(t *testing.T) {
		_, _, err := parseVerifyLocal([]byte("error: no workflow files found"))
		assert.ErrorContains(t, err, "parsing")
	})
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

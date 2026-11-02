package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestDownloadRepositoryRefreshesExistingClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	origin := t.TempDir()
	gitIn(t, origin, "init", "-q")
	// Partial clones (--filter) need the server side to allow filtering.
	gitIn(t, origin, "config", "uploadpack.allowFilter", "true")
	commit := func(content string) {
		path := filepath.Join(origin, ".github", "workflows", "ci.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		gitIn(t, origin, "add", "-A")
		gitIn(t, origin, "commit", "-q", "-m", content)
	}

	d := newTestDownloader(t, nil, 1)
	d.config.DownloadDir = t.TempDir()
	repo := &github.Repository{
		FullName: new("owner/repo"),
		CloneURL: new("file://" + origin),
	}
	checkout := filepath.Join(d.config.DownloadDir, "owner", "repo", ".github", "workflows", "ci.yml")
	assertCheckout := func(want string) {
		t.Helper()
		got, err := os.ReadFile(checkout)
		require.NoError(t, err)
		assert.Equal(t, want, string(got))
	}

	commit("v1")
	require.NoError(t, d.downloadRepository(context.Background(), repo))
	assertCheckout("v1")

	commit("v2")
	require.NoError(t, d.downloadRepository(context.Background(), repo))
	assertCheckout("v2")

	// A clone that git cannot use is replaced rather than failing the repo.
	commit("v3")
	head := filepath.Join(d.config.DownloadDir, "owner", "repo", ".git", "HEAD")
	require.NoError(t, os.WriteFile(head, []byte("garbage"), 0o644))
	require.NoError(t, d.downloadRepository(context.Background(), repo))
	assertCheckout("v3")
}

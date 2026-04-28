package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v85/github"
)

// ImmutableResult records whether the latest release of a repository is
// marked as immutable using GitHub's Immutable Releases feature.
type ImmutableResult struct {
	Repository             string `json:"repository"`
	LatestReleaseImmutable bool   `json:"latest_release_immutable"`
}

// ImmutableReleasesAnalyzer checks whether the latest GitHub release for each
// repository is marked immutable. It requires a GitHub API token for
// reasonable rate limits (one API call per repository).
type ImmutableReleasesAnalyzer struct {
	client    *github.Client
	resultDir string
	results   []ImmutableResult
}

func NewImmutableReleasesAnalyzer(client *github.Client, resultDir string) *ImmutableReleasesAnalyzer {
	return &ImmutableReleasesAnalyzer{client: client, resultDir: resultDir}
}

func (im *ImmutableReleasesAnalyzer) Name() string { return "immutable" }

func (im *ImmutableReleasesAnalyzer) Analyze(ctx context.Context, _, repo string) error {
	ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("unexpected repo format %q (want owner/name)", repo)
	}
	owner, name := parts[0], parts[1]

	release, resp, err := im.client.Repositories.GetLatestRelease(ctx, owner, name)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			// No releases published — not using immutable releases.
			im.results = append(im.results, ImmutableResult{Repository: repo})
			return nil
		}
		return fmt.Errorf("fetching latest release for %s: %w", repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	im.results = append(im.results, ImmutableResult{
		Repository:             repo,
		LatestReleaseImmutable: release.GetImmutable(),
	})
	return nil
}

func (im *ImmutableReleasesAnalyzer) Close() error {
	out := filepath.Join(im.resultDir, "immutable.json")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	if err := json.NewEncoder(f).Encode(im.results); err != nil {
		_ = f.Close()
		return fmt.Errorf("encoding %s: %w", out, err)
	}
	return f.Close()
}

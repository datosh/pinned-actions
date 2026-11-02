package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v93/github"
)

// ImmutableResult records whether the latest release of a repository is
// marked as immutable using GitHub's Immutable Releases feature.
type ImmutableResult struct {
	Repository string `json:"repository"`
	// HasRelease is false when the repository has no published release, which
	// is not the same as a mutable one.
	HasRelease             bool `json:"has_release"`
	LatestReleaseImmutable bool `json:"latest_release_immutable"`
}

// ImmutableReleasesAnalyzer checks whether the latest GitHub release for each
// repository is marked immutable. It requires a GitHub API token for
// reasonable rate limits (one API call per repository).
type ImmutableReleasesAnalyzer struct {
	client *github.Client
}

func NewImmutableReleasesAnalyzer(client *github.Client) *ImmutableReleasesAnalyzer {
	return &ImmutableReleasesAnalyzer{client: client}
}

func (im *ImmutableReleasesAnalyzer) Name() string { return "immutable" }

func (im *ImmutableReleasesAnalyzer) Analyze(ctx context.Context, _, repo string) (any, error) {
	ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected repo format %q (want owner/name)", repo)
	}
	owner, name := parts[0], parts[1]

	release, resp, err := im.client.Repositories.GetLatestRelease(ctx, owner, name)
	if err != nil {
		var ghErr *github.ErrorResponse
		if errors.As(err, &ghErr) && ghErr.Response.StatusCode == http.StatusNotFound {
			// No release published at all.
			return ImmutableResult{Repository: repo}, nil
		}
		return nil, fmt.Errorf("fetching latest release for %s: %w", repo, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return ImmutableResult{
		Repository:             repo,
		HasRelease:             true,
		LatestReleaseImmutable: release.GetImmutable(),
	}, nil
}

func (im *ImmutableReleasesAnalyzer) Close() error { return nil }

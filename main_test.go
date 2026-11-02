package main

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeResult is what fakeAnalyzer produces; like real results it has a repository.
type fakeResult struct {
	Repository string `json:"repository"`
}

type fakeAnalyzer struct {
	name     string
	analyzed []string
	closed   int
	closeErr error
}

func (f *fakeAnalyzer) Name() string {
	if f.name != "" {
		return f.name
	}
	return "fake"
}

func (f *fakeAnalyzer) Analyze(_ context.Context, _, repo string) (any, error) {
	f.analyzed = append(f.analyzed, repo)
	return fakeResult{Repository: repo}, nil
}

func (f *fakeAnalyzer) Close() error {
	f.closed++
	return f.closeErr
}

func TestRunPipeline(t *testing.T) {
	downloadFails := errors.New("search failed")
	closeFails := errors.New("disk full")

	tests := []struct {
		name         string
		downloadErr  error
		closeErr     error
		wantErrs     []error
		wantAnalyzed []string
	}{
		{
			name:         "success",
			wantAnalyzed: []string{"a/one", "b/two"},
		},
		{
			name:         "download fails after some repos: analyzers still closed",
			downloadErr:  downloadFails,
			wantErrs:     []error{downloadFails},
			wantAnalyzed: []string{"a/one", "b/two"},
		},
		{
			name:         "close fails",
			closeErr:     closeFails,
			wantErrs:     []error{closeFails},
			wantAnalyzed: []string{"a/one", "b/two"},
		},
		{
			name:         "download and close fail: both reported",
			downloadErr:  downloadFails,
			closeErr:     closeFails,
			wantErrs:     []error{downloadFails, closeFails},
			wantAnalyzed: []string{"a/one", "b/two"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			download := func(_ context.Context, out chan string) error {
				out <- "a/one"
				out <- "b/two"
				return test.downloadErr
			}
			a := &fakeAnalyzer{closeErr: test.closeErr}

			resultDir := t.TempDir()
			store, storeErr := openResults(resultDir, []Analyzer{a}, false)
			require.NoError(t, storeErr)
			err := runPipeline(context.Background(), download, "/unused", []Analyzer{a}, store)

			if len(test.wantErrs) == 0 {
				require.NoError(t, err)
			}
			for _, want := range test.wantErrs {
				assert.ErrorIs(t, err, want)
			}
			assert.Equal(t, test.wantAnalyzed, a.analyzed)
			assert.Equal(t, 1, a.closed)

			// Results were saved one line per repository, even on failure.
			saved, readErr := readJSONL[fakeResult](filepath.Join(resultDir, "fake.jsonl"))
			require.NoError(t, readErr)
			var savedRepos []string
			for _, s := range saved {
				savedRepos = append(savedRepos, s.Repository)
			}
			assert.Equal(t, test.wantAnalyzed, savedRepos)
		})
	}
}

func TestRunPipelineCancelledStillFlushes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	download := func(ctx context.Context, out chan string) error {
		out <- "a/one"
		cancel() // e.g. SIGINT
		out <- "b/two"
		return ctx.Err()
	}
	a := &fakeAnalyzer{}

	store, storeErr := openResults(t.TempDir(), []Analyzer{a}, false)
	require.NoError(t, storeErr)
	err := runPipeline(ctx, download, "/unused", []Analyzer{a}, store)

	assert.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, a.analyzed, "b/two", "repos arriving after cancellation are not analyzed")
	assert.Equal(t, 1, a.closed, "analyzers are closed so results are written")
}

func TestRunGitHonorsContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	assert.Error(t, runGit(ctx, "version"))
	assert.NoError(t, runGit(context.Background(), "version"))
}

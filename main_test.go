package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAnalyzer struct {
	analyzed []string
	closed   int
	closeErr error
}

func (f *fakeAnalyzer) Name() string { return "fake" }

func (f *fakeAnalyzer) Analyze(_ context.Context, _, repo string) error {
	f.analyzed = append(f.analyzed, repo)
	return nil
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

			err := runPipeline(context.Background(), download, "/unused", []Analyzer{a})

			if len(test.wantErrs) == 0 {
				require.NoError(t, err)
			}
			for _, want := range test.wantErrs {
				assert.ErrorIs(t, err, want)
			}
			assert.Equal(t, test.wantAnalyzed, a.analyzed)
			assert.Equal(t, 1, a.closed)
		})
	}
}

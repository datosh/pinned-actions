package main

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestZizmorAnalyzer runs the real zizmor binary; it is skipped when zizmor
// is not installed.
func TestZizmorAnalyzer(t *testing.T) {
	if _, err := exec.LookPath("zizmor"); err != nil {
		t.Skip("zizmor not on $PATH")
	}

	tests := []struct {
		name         string
		repo         string
		wantUsesGHA  bool
		wantFindings int
	}{
		{name: "unpinned workflow", repo: "unpinned", wantUsesGHA: true, wantFindings: 1},
		{name: "only a dependabot config", repo: "no-workflows"},
		{name: "no .github directory", repo: "no-github"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a, err := NewZizmorAnalyzer(t.TempDir())
			require.NoError(t, err)

			require.NoError(t, a.Analyze(context.Background(), "testdata", test.repo))

			require.Len(t, a.results, 1)
			got := a.results[0]
			assert.Equal(t, test.wantUsesGHA, got.UsesGHA)
			assert.Len(t, got.Findings, test.wantFindings)
		})
	}
}

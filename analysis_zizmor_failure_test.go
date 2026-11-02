package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeZizmor returns an analyzer whose "zizmor" is a shell script.
func fakeZizmor(t *testing.T, script string) *ZizmorAnalyzer {
	t.Helper()
	path := filepath.Join(t.TempDir(), "zizmor")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755))
	return &ZizmorAnalyzer{zizmorBin: path}
}

func analyzeUnpinned(t *testing.T, a *ZizmorAnalyzer) ZizmorResult {
	t.Helper()
	got, err := a.Analyze(context.Background(), "testdata", "unpinned")
	require.NoError(t, err)
	result, ok := got.(ZizmorResult)
	require.True(t, ok)
	return result
}

func TestZizmorAnalyzerKeepsRepositoriesZizmorCrashesOn(t *testing.T) {
	// What zizmor prints when its artipacked audit fails (vscode): progress
	// first, the cause last, exit code 1.
	a := fakeZizmor(t, `
echo "INFO zizmor: starting" >&2
echo "fatal: no audit was performed" >&2
echo "" >&2
echo "    1: unexpected node: comment" >&2
exit 1
`)

	result := analyzeUnpinned(t, a)

	assert.Equal(t, ZizmorResult{
		Repository: "unpinned",
		UsesGHA:    true,
		Findings:   []ZizmorFinding{},
		Error:      "zizmor exited with status 1: 1: unexpected node: comment",
	}, result)
}

func TestZizmorAnalyzerFailureWithoutOutput(t *testing.T) {
	result := analyzeUnpinned(t, fakeZizmor(t, "exit 2\n"))

	assert.Equal(t, "zizmor exited with status 2", result.Error)
}

func TestZizmorAnalyzerKeepsRepositoriesWithUnreadableOutput(t *testing.T) {
	result := analyzeUnpinned(t, fakeZizmor(t, "echo 'not json'\n"))

	assert.True(t, strings.HasPrefix(result.Error, "unreadable zizmor output"), result.Error)
	assert.Empty(t, result.Findings)
}

func TestZizmorAnalyzerExitCodeThreeMeansNoActions(t *testing.T) {
	result := analyzeUnpinned(t, fakeZizmor(t, "exit 3\n"))

	assert.Equal(t, ZizmorResult{Repository: "unpinned", Findings: []ZizmorFinding{}}, result)
}

func TestZizmorAnalyzerCancellationIsAnErrorNotAFailedRepository(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := fakeZizmor(t, "exit 1\n").Analyze(ctx, "testdata", "unpinned")

	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, got)
}

func TestZizmorAnalyzerMissingBinaryIsAnError(t *testing.T) {
	a := &ZizmorAnalyzer{zizmorBin: filepath.Join(t.TempDir(), "does-not-exist")}

	got, err := a.Analyze(context.Background(), "testdata", "unpinned")

	assert.ErrorContains(t, err, "running zizmor")
	assert.Nil(t, got)
}

func TestLastLine(t *testing.T) {
	assert.Equal(t, "", lastLine(nil))
	assert.Equal(t, "only", lastLine([]byte("only\n")))
	assert.Equal(t, "last", lastLine([]byte("first\n\nlast\n\n")))
	assert.Equal(t, "indented", lastLine([]byte("first\n    indented  \n")))

	long := strings.Repeat("ä", 300) // multi-byte characters must not be cut in half
	assert.Equal(t, strings.Repeat("ä", 200), lastLine([]byte(long)))
}

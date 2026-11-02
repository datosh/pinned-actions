package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeJSONL(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file starts empty", func(t *testing.T) {
		w, done, err := resumeJSONL(filepath.Join(dir, "new.jsonl"))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		assert.Empty(t, done)
	})

	t.Run("keeps complete lines, drops a cut-off last line, appends", func(t *testing.T) {
		path := filepath.Join(dir, "killed.jsonl")
		// The last record was cut off mid-write: no closing brace, no newline.
		require.NoError(t, os.WriteFile(path, []byte("{\"repository\":\"a/one\"}\n{\"repository\":\"b/two\"}\n{\"repository\":\"c/th"), 0o644))

		w, done, err := resumeJSONL(path)
		require.NoError(t, err)
		assert.Equal(t, map[string]bool{"a/one": true, "b/two": true}, done)

		require.NoError(t, w.Write(fakeResult{Repository: "c/three"}))
		require.NoError(t, w.Close())

		got, err := readJSONL[fakeResult](path)
		require.NoError(t, err)
		assert.Equal(t, []fakeResult{{"a/one"}, {"b/two"}, {"c/three"}}, got)
	})

	t.Run("corrupt complete line is an error", func(t *testing.T) {
		path := filepath.Join(dir, "corrupt.jsonl")
		require.NoError(t, os.WriteFile(path, []byte("not json\n"), 0o644))

		_, _, err := resumeJSONL(path)
		assert.ErrorContains(t, err, "parsing")
	})
}

func TestOpenResultsWithoutResumeStartsOver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fake.jsonl")
	require.NoError(t, os.WriteFile(path, []byte("{\"repository\":\"a/one\"}\n"), 0o644))

	store, err := openResults(dir, []Analyzer{&fakeAnalyzer{}}, false)
	require.NoError(t, err)
	require.NoError(t, store.Close())

	assert.False(t, store.has(0, "a/one"))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Empty(t, data)
}

func TestResultStoreCompleteIsPerAnalyzer(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "first.jsonl"), []byte("{\"repository\":\"a/one\"}\n{\"repository\":\"b/two\"}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "second.jsonl"), []byte("{\"repository\":\"a/one\"}\n"), 0o644))

	store, err := openResults(dir, []Analyzer{&fakeAnalyzer{name: "first"}, &fakeAnalyzer{name: "second"}}, true)
	require.NoError(t, err)
	defer func() { _ = store.Close() }()

	assert.True(t, store.complete("a/one"), "both analyzers have it")
	assert.False(t, store.complete("b/two"), "only the first analyzer has it")
	assert.False(t, store.complete("c/three"))
	assert.True(t, store.has(0, "b/two"))
	assert.False(t, store.has(1, "b/two"))
}

func TestRunPipelineResumeSkipsFinishedWork(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "first.jsonl"), []byte("{\"repository\":\"a/one\"}\n"), 0o644))

	first, second := &fakeAnalyzer{name: "first"}, &fakeAnalyzer{name: "second"}
	analyzers := []Analyzer{first, second}
	store, err := openResults(dir, analyzers, true)
	require.NoError(t, err)

	download := func(_ context.Context, out chan string) error {
		out <- "a/one"
		out <- "b/two"
		return nil
	}
	require.NoError(t, runPipeline(context.Background(), download, "/unused", analyzers, store))

	assert.Equal(t, []string{"b/two"}, first.analyzed, "a/one was already done by first")
	assert.Equal(t, []string{"a/one", "b/two"}, second.analyzed, "second had no results yet")

	got, err := readJSONL[fakeResult](filepath.Join(dir, "first.jsonl"))
	require.NoError(t, err)
	assert.Equal(t, []fakeResult{{"a/one"}, {"b/two"}}, got, "existing results are kept")
}

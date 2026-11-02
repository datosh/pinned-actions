package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.json")

	require.NoError(t, writeJSON(path, map[string]int{"a": 1}))
	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"a":1}`, string(got))

	// An existing file is replaced, not appended to.
	require.NoError(t, writeJSON(path, []string{"x"}))
	got, err = os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `["x"]`, string(got))
}

func TestReadJSON(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.json")
	require.NoError(t, os.WriteFile(valid, []byte(`{"a":1}`), 0o644))
	invalid := filepath.Join(dir, "invalid.json")
	require.NoError(t, os.WriteFile(invalid, []byte(`[1,2]`), 0o644))

	var got map[string]int
	require.NoError(t, readJSON(valid, &got))
	assert.Equal(t, map[string]int{"a": 1}, got)

	assert.ErrorContains(t, readJSON(filepath.Join(dir, "missing.json"), &got), "reading")
	assert.ErrorContains(t, readJSON(invalid, &got), "parsing")
}

func TestJSONLRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.jsonl")

	w, err := newJSONLWriter(path)
	require.NoError(t, err)
	require.NoError(t, w.Write(map[string]int{"a": 1}))

	// Each line is on disk before Close, so a crash keeps earlier results.
	got, err := readJSONL[map[string]int](path)
	require.NoError(t, err)
	assert.Equal(t, []map[string]int{{"a": 1}}, got)

	require.NoError(t, w.Write(map[string]int{"b": 2}))
	require.NoError(t, w.Close())

	got, err = readJSONL[map[string]int](path)
	require.NoError(t, err)
	assert.Equal(t, []map[string]int{{"a": 1}, {"b": 2}}, got)
}

func TestReadJSONLEmptyAndInvalid(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty.jsonl")
	require.NoError(t, os.WriteFile(empty, nil, 0o644))
	broken := filepath.Join(dir, "broken.jsonl")
	require.NoError(t, os.WriteFile(broken, []byte("{\"a\":1}\n{\"a\":"), 0o644))

	got, err := readJSONL[map[string]int](empty)
	require.NoError(t, err)
	assert.NotNil(t, got, "an empty file is an empty list, not nil")
	assert.Empty(t, got)

	_, err = readJSONL[map[string]int](broken)
	assert.ErrorContains(t, err, "parsing")

	_, err = readJSONL[map[string]int](filepath.Join(dir, "missing.jsonl"))
	assert.ErrorContains(t, err, "reading")
}

func TestWriteJSONErrors(t *testing.T) {
	dir := t.TempDir()

	assert.ErrorContains(t, writeJSON(filepath.Join(dir, "missing", "out.json"), 1), "creating")
	assert.ErrorContains(t, writeJSON(filepath.Join(dir, "out.json"), make(chan int)), "encoding")
}

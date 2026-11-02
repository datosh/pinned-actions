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

func TestWriteJSONErrors(t *testing.T) {
	dir := t.TempDir()

	assert.ErrorContains(t, writeJSON(filepath.Join(dir, "missing", "out.json"), 1), "creating")
	assert.ErrorContains(t, writeJSON(filepath.Join(dir, "out.json"), make(chan int)), "encoding")
}

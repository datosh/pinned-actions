package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyzerNames(t *testing.T) {
	assert.Equal(t, []string{"pinned", "zizmor", "immutable"}, analyzerNames())
}

func TestBuildAnalyzers(t *testing.T) {
	config := Config{ResultDir: t.TempDir(), Analyzers: []string{"pinned", "immutable"}}

	analyzers, err := buildAnalyzers(config, nil)
	require.NoError(t, err)

	var names []string
	for _, a := range analyzers {
		names = append(names, a.Name())
	}
	assert.Equal(t, []string{"pinned", "immutable"}, names, "order follows the configuration")
}

func TestBuildAnalyzersUnknown(t *testing.T) {
	config := Config{ResultDir: t.TempDir(), Analyzers: []string{"pinned", "nope"}}

	_, err := buildAnalyzers(config, nil)

	assert.ErrorContains(t, err, `unknown analyzer "nope"`)
	assert.ErrorContains(t, err, "available: pinned, zizmor, immutable")
}

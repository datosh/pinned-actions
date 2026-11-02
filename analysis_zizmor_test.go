package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseZizmorOutput(t *testing.T) {
	// Real output of `zizmor --format=json --offline testdata/unpinned/.github`
	// from zizmor 1.31.1, run from the repository root.
	data, err := os.ReadFile(filepath.Join("testdata", "zizmor", "unpinned.json"))
	require.NoError(t, err)

	findings, err := parseZizmorOutput(data, "testdata", "unpinned")
	require.NoError(t, err)

	require.Len(t, findings, 1)
	assert.Equal(t, ZizmorFinding{
		Rule:       "unpinned-uses",
		Severity:   "High",
		Confidence: "High",
		File:       ".github/workflows/test.yml",
		// The `uses:` line in testdata/unpinned/.github/workflows/test.yml.
		Line: 18,
	}, findings[0])
}

func TestCheckZizmorVersion(t *testing.T) {
	tests := []struct {
		output  string
		wantErr bool
	}{
		{output: "zizmor 1.31.1\n"},
		{output: "zizmor 1.31.2"},
		{output: "zizmor 1.40.0"},
		{output: "zizmor 2.0.0"},
		{output: "zizmor 1.32.0-rc.1"},
		{output: "zizmor 1.31.0", wantErr: true},
		{output: "zizmor 1.30.9", wantErr: true},
		{output: "zizmor 0.99.99", wantErr: true},
		{output: "zizmor", wantErr: true},
		{output: "zizmor one.two.three", wantErr: true},
		{output: "", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.output, func(t *testing.T) {
			err := checkZizmorVersion(test.output)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseZizmorOutputNoFindings(t *testing.T) {
	findings, err := parseZizmorOutput([]byte(`[]`), "/dl", "o/r")
	require.NoError(t, err)

	assert.NotNil(t, findings)
	assert.Empty(t, findings)

	// The result file must say [], not null.
	encoded, err := json.Marshal(ZizmorResult{Repository: "o/r", UsesGHA: true, Findings: findings})
	require.NoError(t, err)
	assert.JSONEq(t, `{"repository":"o/r","uses_gha":true,"findings":[]}`, string(encoded))
}

func TestParseZizmorOutputNoLocalLocation(t *testing.T) {
	data := []byte(`[{"ident":"x","determinations":{"severity":"Low","confidence":"Low"},"locations":[]}]`)

	findings, err := parseZizmorOutput(data, "/dl", "o/r")
	require.NoError(t, err)

	require.Len(t, findings, 1)
	assert.Empty(t, findings[0].File)
	assert.Zero(t, findings[0].Line)
}

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHex(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "valid lowercase hex", input: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", want: true},
		{name: "uppercase hex rejected", input: "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2", want: false},
		{name: "mixed case rejected", input: "a1B2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", want: false},
		{name: "non-hex character", input: "g1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", want: false},
		{name: "empty string", input: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, isHex(test.input))
		})
	}
}

func TestPinnedAnalyzer(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	testdata := filepath.Join(wd, "testdata")

	tests := []struct {
		name              string
		repo              string
		wantPinned        int
		wantTotal         int
		wantHasRenovate   bool
		wantHasDependabot bool
	}{
		{
			name:       "Single pinned action",
			repo:       "pinned",
			wantPinned: 1,
			wantTotal:  1,
		},
		{
			name:       "Single unpinned action",
			repo:       "unpinned",
			wantPinned: 0,
			wantTotal:  1,
		},
		{
			name:       "OCI pinned",
			repo:       "oci-pinned",
			wantPinned: 1,
			wantTotal:  1,
		},
		{
			name:       "OCI unpinned",
			repo:       "oci-unpinned",
			wantPinned: 0,
			wantTotal:  1,
		},
		{
			name:       "Local action",
			repo:       "local",
			wantPinned: 0,
			wantTotal:  0,
		},
		{
			name:              "No workflows directory",
			repo:              "no-workflows",
			wantPinned:        0,
			wantTotal:         0,
			wantHasDependabot: true,
		},
		{
			name:       "Composite action only",
			repo:       "composite",
			wantPinned: 1,
			wantTotal:  2,
		},
		{
			name:       "Same reference in workflow and composite action counts once",
			repo:       "composite-dedup",
			wantPinned: 1,
			wantTotal:  1,
		},
		{
			name:       "No .github directory",
			repo:       "no-github",
			wantPinned: 0,
			wantTotal:  0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := NewPinnedAnalyzer(t.TempDir())
			err := a.Analyze(context.Background(), testdata, test.repo)
			assert.NoError(t, err)

			require.Len(t, a.results, 1)
			got := a.results[0]
			assert.Equal(t, test.wantPinned, got.ActionsPinned)
			assert.Equal(t, test.wantTotal, got.ActionsTotal)
			assert.Equal(t, test.wantHasRenovate, got.HasRenovate)
			assert.Equal(t, test.wantHasDependabot, got.HasDependabot)
		})
	}
}

func TestAnalyseRepositoryConfigDetection(t *testing.T) {
	tests := []struct {
		file           string
		wantRenovate   bool
		wantDependabot bool
	}{
		{file: "renovate.json", wantRenovate: true},
		{file: "renovate.json5", wantRenovate: true},
		{file: ".github/renovate.json", wantRenovate: true},
		{file: ".github/renovate.json5", wantRenovate: true},
		{file: ".gitlab/renovate.json", wantRenovate: true},
		{file: ".gitlab/renovate.json5", wantRenovate: true},
		{file: ".renovaterc", wantRenovate: true},
		{file: ".renovaterc.json", wantRenovate: true},
		{file: ".renovaterc.json5", wantRenovate: true},
		{file: ".github/dependabot.yml", wantDependabot: true},
		{file: ".github/dependabot.yaml", wantDependabot: true},
		{file: "README.md"},
	}

	for _, test := range tests {
		t.Run(test.file, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "owner", "repo", test.file)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			require.NoError(t, os.WriteFile(path, nil, 0o644))

			got, err := analyseRepository(dir, "owner/repo")
			require.NoError(t, err)
			assert.Equal(t, test.wantRenovate, got.HasRenovate)
			assert.Equal(t, test.wantDependabot, got.HasDependabot)
		})
	}

	t.Run("no config", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "owner", "repo"), 0o755))

		got, err := analyseRepository(dir, "owner/repo")
		require.NoError(t, err)
		assert.False(t, got.HasRenovate)
		assert.False(t, got.HasDependabot)
	})
}

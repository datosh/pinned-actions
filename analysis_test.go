package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyseRepository(t *testing.T) {
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis, err := AnalyseRepository(testdata, test.repo)
			assert.NoError(t, err)

			assert.Equal(t, test.wantPinned, analysis.ActionsPinned)
			assert.Equal(t, test.wantTotal, analysis.ActionsTotal)
			assert.Equal(t, test.wantHasRenovate, analysis.HasRenovate)
			assert.Equal(t, test.wantHasDependabot, analysis.HasDependabot)
		})
	}
}

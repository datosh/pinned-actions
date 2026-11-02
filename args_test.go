package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScanArgsDefaults(t *testing.T) {
	got := ParseScanArgs(nil)

	assert.Equal(t, *NewConfig(), got)
}

func TestParseScanArgsOverrides(t *testing.T) {
	got := ParseScanArgs([]string{
		"-download-dir=/dl",
		"-result-dir=out",
		"-max-pages=7",
		"-per-page=50",
		"-analyzer=pinned,zizmor",
	})

	want := *NewConfig()
	want.DownloadDir = "/dl"
	want.ResultDir = "out"
	want.MaxPages = 7
	want.PerPage = 50
	want.Analyzers = []string{"pinned", "zizmor"}
	assert.Equal(t, want, got)
}

package main

import (
	"flag"
	"os"
	"strings"
)

func ParseScanArgs(args []string) Config {
	config := NewConfig()
	fs := flag.NewFlagSet("scan", flag.ExitOnError)

	fs.StringVar(&config.DownloadDir, "download-dir", config.DownloadDir, "path to folder where repositories will be downloaded")
	fs.StringVar(&config.ResultDir, "result-dir", config.ResultDir, "path to folder where analysis results will be written")
	fs.IntVar(&config.MaxPages, "max-pages", config.MaxPages, "maximum number of pages to download")
	fs.IntVar(&config.PerPage, "per-page", config.PerPage, "number of repositories to download per page")

	analyzerFlag := fs.String("analyzer", strings.Join(config.Analyzers, ","), "comma-separated list of analyzers to run (available: "+strings.Join(analyzerNames(), ", ")+")")

	if err := fs.Parse(args); err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}

	config.Analyzers = strings.Split(*analyzerFlag, ",")

	return *config
}

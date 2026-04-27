package main

import (
	"flag"
	"os"
)

func ParseArgs() Config {
	config := NewConfig()
	fs := flag.NewFlagSet("GH Pinned Actions", flag.ExitOnError)

	fs.StringVar(&config.DownloadDir, "download-dir", "/tmp/pinned", "path to folder where repositories will be downloaded")
	fs.StringVar(&config.ResultDir, "result-dir", "results", "path to folder where analysis results will be written")
	fs.IntVar(&config.MaxPages, "max-pages", 1, "maximum number of pages to download")
	fs.IntVar(&config.PerPage, "per-page", 100, "number of repositories to download per page")

	err := fs.Parse(os.Args[1:])
	if err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}

	return *config
}

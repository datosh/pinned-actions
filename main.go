package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v86/github"
)

const usage = `Usage: pinned-actions <command> [flags]

Commands:
  scan           Download repositories and run analyzers
  export-pinned  Export pinned.json to the format expected by the frontend
  export-zizmor  Export zizmor.json to a slimmed per-rule-count format for the frontend

Run 'pinned-actions <command> -h' for command-specific flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		runScan(os.Args[2:])
	case "export-pinned":
		runExportPinned(os.Args[2:])
	case "export-zizmor":
		runExportZizmor(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func runScan(args []string) {
	config := ParseScanArgs(args)
	log.Printf("Configuration:\n%s", config)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Printf("[WARN] GITHUB_TOKEN not set; unauthenticated search requests are heavily rate-limited and may return incomplete results")
	}
	client := github.NewClient(nil).WithAuthToken(token)

	ctx := context.Background()

	if err := os.MkdirAll(config.ResultDir, 0o755); err != nil {
		log.Fatalf("creating result directory: %v", err)
	}

	analyzers, err := buildAnalyzers(config, client)
	if err != nil {
		log.Fatalf("configuring analyzers: %v", err)
	}

	downloaded := make(chan string)
	downloader := NewRepositoryDownloader(client, config)

	go func() {
		err := downloader.Download(ctx, downloaded)
		if err != nil {
			log.Fatalf("downloading: %v", err)
		}
		close(downloaded)
	}()

	for repo := range downloaded {
		for _, a := range analyzers {
			if err := a.Analyze(ctx, config.DownloadDir, repo); err != nil {
				log.Printf("[ERROR] %s: analysing %s: %v", a.Name(), repo, err)
			}
		}
	}

	for _, a := range analyzers {
		if err := a.Close(); err != nil {
			log.Fatalf("closing analyzer %s: %v", a.Name(), err)
		}
	}
}

func buildAnalyzers(config Config, client *github.Client) ([]Analyzer, error) {
	var analyzers []Analyzer
	for _, name := range config.Analyzers {
		switch name {
		case "pinned":
			analyzers = append(analyzers, NewPinnedAnalyzer(config.ResultDir))
		case "zizmor":
			a, err := NewZizmorAnalyzer(config.ResultDir)
			if err != nil {
				return nil, err
			}
			analyzers = append(analyzers, a)
		case "immutable":
			analyzers = append(analyzers, NewImmutableReleasesAnalyzer(client, config.ResultDir))
		default:
			return nil, fmt.Errorf("unknown analyzer %q (available: pinned, zizmor, immutable)", name)
		}
	}
	return analyzers, nil
}

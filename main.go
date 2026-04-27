package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/go-github/v85/github"
)

func main() {
	config := ParseArgs()
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
		default:
			return nil, fmt.Errorf("unknown analyzer %q (available: pinned, zizmor)", name)
		}
	}
	return analyzers, nil
}

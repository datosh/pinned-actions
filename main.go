package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-github/v93/github"
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
	var opts []github.ClientOptionsFunc
	if token != "" {
		// WithAuthToken rejects an empty token, so only add it when set.
		opts = append(opts, github.WithAuthToken(token))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		log.Fatalf("creating GitHub client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		log.Printf("Received %v; flushing results (signal again to abort)", sig)
		signal.Stop(signals) // a second signal now kills the process
		cancel()
	}()

	if err := os.MkdirAll(config.ResultDir, 0o755); err != nil {
		log.Fatalf("creating result directory: %v", err)
	}

	analyzers, err := buildAnalyzers(config, client)
	if err != nil {
		log.Fatalf("configuring analyzers: %v", err)
	}

	downloader := NewRepositoryDownloader(client, config)
	if err := runPipeline(ctx, downloader.Download, config.DownloadDir, analyzers); err != nil {
		log.Fatal(err)
	}
}

// runPipeline feeds every downloaded repository to all analyzers. Analyzers
// are always closed, so results gathered so far are flushed even if the
// download fails part way; all errors are returned together afterwards.
func runPipeline(
	ctx context.Context,
	download func(context.Context, chan string) error,
	downloadDir string,
	analyzers []Analyzer,
) error {
	downloaded := make(chan string)
	downloadErr := make(chan error, 1)

	go func() {
		defer close(downloaded)
		if err := download(ctx, downloaded); err != nil {
			downloadErr <- fmt.Errorf("downloading: %w", err)
		}
		close(downloadErr)
	}()

	for repo := range downloaded {
		if ctx.Err() != nil {
			continue // keep draining so the downloader can finish
		}
		for _, a := range analyzers {
			if err := a.Analyze(ctx, downloadDir, repo); err != nil {
				log.Printf("[ERROR] %s: analysing %s: %v", a.Name(), repo, err)
			}
		}
	}

	errs := []error{<-downloadErr}
	for _, a := range analyzers {
		if err := a.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing analyzer %s: %w", a.Name(), err))
		}
	}
	return errors.Join(errs...)
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

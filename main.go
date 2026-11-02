package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/go-github/v93/github"
)

const usage = `Usage: pinned-actions <command> [flags]

Commands:
  scan           Download repositories and run analyzers
  export-pinned  Export pinned.jsonl to the format expected by the frontend
  export-zizmor  Export zizmor.jsonl to a slimmed per-rule-count format for the frontend

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

// fatal logs an error and exits with a non-zero status.
func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func runScan(args []string) {
	config := ParseScanArgs(args)
	slog.Info("scan configuration", "config", config)

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		slog.Warn("GITHUB_TOKEN not set; unauthenticated search requests are heavily rate-limited and may return incomplete results")
	}
	var opts []github.ClientOptionsFunc
	if token != "" {
		// WithAuthToken rejects an empty token, so only add it when set.
		opts = append(opts, github.WithAuthToken(token))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		fatal("creating GitHub client", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signals
		slog.Info("received signal; flushing results (signal again to abort)", "signal", sig)
		signal.Stop(signals) // a second signal now kills the process
		cancel()
	}()

	if err := os.MkdirAll(config.ResultDir, 0o755); err != nil {
		fatal("creating result directory", "path", config.ResultDir, "error", err)
	}

	analyzers, err := buildAnalyzers(config, client)
	if err != nil {
		fatal("configuring analyzers", "error", err)
	}

	store, err := openResults(config.ResultDir, analyzers, config.Resume)
	if err != nil {
		fatal("opening results", "error", err)
	}

	downloader := NewRepositoryDownloader(client, config)
	downloader.skip = store.complete // no need to fetch repos nobody needs
	if err := runPipeline(ctx, downloader.Download, config.DownloadDir, analyzers, store); err != nil {
		fatal("scan failed", "error", err)
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
	store *resultStore,
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
		for i, a := range analyzers {
			if store.has(i, repo) {
				continue // resumed run: this analyzer already has its result
			}
			result, err := a.Analyze(ctx, downloadDir, repo)
			if err != nil {
				slog.Error("analysis failed", "analyzer", a.Name(), "repo", repo, "error", err)
				continue
			}
			if result == nil {
				continue
			}
			if err := store.write(i, result); err != nil {
				slog.Error("saving result failed", "analyzer", a.Name(), "repo", repo, "error", err)
			}
		}
	}

	errs := []error{<-downloadErr, store.Close()}
	for _, a := range analyzers {
		if err := a.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing analyzer %s: %w", a.Name(), err))
		}
	}
	return errors.Join(errs...)
}

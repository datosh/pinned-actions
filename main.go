package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee/pkg/ghactions"
)

func main() {
	config := NewConfig()
	client := github.NewClient(nil)

	ctx := context.Background()

	downloaded := make(chan string)
	downloader := NewRepositoryDownloader(client, config)

	go func() {
		log.Println("Waiting for downloads")
		for {
			select {
			case repo := <-downloaded:
				analysis, err := AnalyseRepository(*config, repo)
				if err != nil {
					log.Fatalf("analysing repository: %v", err)
				}
				log.Printf("%v", analysis)
			}
		}
	}()

	err := downloader.Download(ctx, downloaded)
	if err != nil {
		log.Fatalf("downloading: %v", err)
	}

	// wait for both be done
}

func AnalyseRepository(config Config, repo string) (Analysis, error) {
	analysis := NewAnalysis(repo)

	repoPath := filepath.Join(config.DownloadDir, repo)
	actions, err := ghactions.ListActionsInDirectory(repoPath)
	if err != nil {
		return analysis, fmt.Errorf("listing actions: %w", err)
	}

	for _, action := range actions {
		if len(action.Ref) == 40 && isHex(action.Ref) {
			analysis.CountPinned()
		} else {
			analysis.CountUnpinned()
		}
	}

	return analysis, nil
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

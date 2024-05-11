package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/go-github/v60/github"
	"github.com/stacklok/frizbee/pkg/ghactions"
)

func main() {
	config := NewConfig()
	client := github.NewClient(nil)

	ctx := context.Background()

	done := make(chan bool)
	downloaded := make(chan string)
	downloader := NewRepositoryDownloader(client, config)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		var analyzed []Analysis

		for {
			select {

			case repo := <-downloaded:
				analysis, err := AnalyseRepository(*config, repo)
				if err != nil {
					log.Fatalf("analysing repository: %v", err)
				}
				analyzed = append(analyzed, analysis)

			case <-done:
				resultFile, err := os.Create(config.ResultFile)
				if err != nil {
					log.Fatalf("creating output file: %v", err)
				}

				json.NewEncoder(resultFile).Encode(analyzed)

				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		err := downloader.Download(ctx, downloaded)
		if err != nil {
			log.Fatalf("downloading: %v", err)
		}

		done <- true
	}()

	wg.Wait()
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

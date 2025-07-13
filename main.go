package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/stacklok/frizbee/pkg/replacer"

	"github.com/google/go-github/v62/github"
	fzconfig "github.com/stacklok/frizbee/pkg/utils/config"
)

func main() {
	config := ParseArgs()
	log.Printf("Configuration:\n%s", config)

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
				analysis, err := AnalyseRepository(config, repo)
				if err != nil {
					log.Printf("[ERROR] analysing repository: %v", err)
					// TODO: https://github.com/stacklok/frizbee/issues/77
					continue
				}
				analyzed = append(analyzed, analysis)

			case <-done:
				resultFile, err := os.Create(config.ResultFile)
				if err != nil {
					log.Fatalf("creating output file: %v", err)
				}

				err = json.NewEncoder(resultFile).Encode(analyzed)
				if err != nil {
					log.Fatalf("encoding output file: %v", err)
				}

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
	workflowsPath := filepath.Join(repoPath, ".github", "workflows")

	// Create a new Frizbee instance
	r := replacer.NewGitHubActionsReplacer(&fzconfig.Config{})

	actions, err := r.ListPath(workflowsPath)
	if err != nil {
		return analysis, fmt.Errorf("listing actions: %w", err)
	}

	for _, action := range actions.Entities {
		if len(action.Ref) == 40 && isHex(action.Ref) {
			analysis.CountPinned()
		} else {
			analysis.CountUnpinned()
		}
	}

	// https://docs.renovatebot.com/configuration-options/
	renovatePaths := []string{
		filepath.Join(repoPath, "renovate.json"),
		filepath.Join(repoPath, "renovate.json5"),
		filepath.Join(repoPath, ".github", "renovate.json"),
		filepath.Join(repoPath, ".github", "renovate.json5"),
		filepath.Join(repoPath, ".gitlab", "renovate.json"),
		filepath.Join(repoPath, ".gitlab", "renovate.json5"),
		filepath.Join(repoPath, ".renovaterc"),
		filepath.Join(repoPath, ".renovaterc.json"),
		filepath.Join(repoPath, ".renovaterc.json5"),
	}

	renovate := slices.ContainsFunc(renovatePaths, func(path string) bool {
		return exists(path)
	})
	analysis.HasRenovate = renovate

	dependabotPath := filepath.Join(repoPath, ".github", "dependabot.yml")
	if exists(dependabotPath) {
		analysis.HasDependabot = true
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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/go-github/v62/github"
)

type RepositoryDownloader struct {
	client     *github.Client
	config     Config
	downloaded chan string //nolint:unused
}

func NewRepositoryDownloader(client *github.Client, config Config) *RepositoryDownloader {
	return &RepositoryDownloader{
		client: client,
		config: config,
	}
}

func (r *RepositoryDownloader) Download(ctx context.Context, downloaded chan string) error {
	ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: r.config.PerPage,
		},
	}

	maxStars := 500000
	minStars := 1000

	for i := 0; i < r.config.MaxPages; i++ {
		query := buildQuery(minStars, maxStars)
		log.Printf("Searching repos with: %s\n", query)
		searchResult, resp, err := r.client.Search.Repositories(ctx, query, opts)
		if err != nil {
			return fmt.Errorf("searching repositories: %w", err)
		}

		if searchResult == nil {
			return fmt.Errorf("No search results")
		}

		for _, repository := range searchResult.Repositories {
			maxStars = repository.GetStargazersCount() - 1
			if err := r.downloadRepository(repository); err != nil {
				log.Printf("[ERROR] downloading repository: %v", err)
				continue
			}
			downloaded <- repository.GetFullName()
		}

		if resp.NextPage == 0 {
			log.Printf("Handled all pages after %d requests.\n", i+1)
			break
		}
	}

	return nil
}

func (r *RepositoryDownloader) downloadRepository(repository *github.Repository) error {
	cloneURL := repository.GetCloneURL()
	log.Printf("Clone from: %s\n", cloneURL)

	cloneInto := filepath.Join(r.config.DownloadDir, *repository.FullName)
	log.Printf("Clone into: %s\n", cloneInto)

	if exists(cloneInto) {
		log.Printf("Repository already exists: %s\n", cloneInto)
		// TODO: Fetch latest changes
		return nil
	}

	// TODO: Prevent that username & password are queried. Disable TTY?
	err := exec.Command("git", "clone", "--depth", "1", cloneURL, cloneInto).Run()
	if err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	return nil
}

// exists returns whether the given file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	panic(err)
}

func buildQuery(minStars, maxStars int) string {
	return fmt.Sprintf("stars:%d..%d", minStars, maxStars)
}

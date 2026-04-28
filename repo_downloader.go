package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/google/go-github/v85/github"
)

var blockedRepos = []string{
	"cdnjs/cdnjs",
	"go-xorm/xorm",
}

type RepositoryDownloader struct {
	client *github.Client
	config Config
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

	stars := make(map[string]int)

	for i := 0; i < r.config.MaxPages; i++ {
		query := buildQuery(minStars, maxStars)
		log.Printf("Searching repos with: %s\n", query)
		searchResult, _, err := r.client.Search.Repositories(ctx, query, opts)
		if err != nil {
			return fmt.Errorf("searching repositories: %w", err)
		}

		if searchResult == nil {
			return fmt.Errorf("no search results")
		}

		if searchResult.GetIncompleteResults() {
			log.Printf("[WARN] GitHub returned incomplete results for query %q — retrying\n", query)
			i--
			continue
		}

		log.Printf("Found %d repositories\n", len(searchResult.Repositories))

		for _, repository := range searchResult.Repositories {
			maxStars = repository.GetStargazersCount() - 1
			stars[repository.GetFullName()] = repository.GetStargazersCount()
			if err := r.downloadRepository(repository); err != nil {
				log.Printf("[ERROR] downloading repository: %v", err)
				continue
			}
			downloaded <- repository.GetFullName()
		}

		if len(searchResult.Repositories) < r.config.PerPage {
			log.Printf("Handled all pages after %d requests.\n", i+1)
			break
		}
	}

	if err := r.writeStars(stars); err != nil {
		return fmt.Errorf("writing stars: %w", err)
	}

	return nil
}

func (r *RepositoryDownloader) writeStars(stars map[string]int) error {
	out := filepath.Join(r.config.ResultDir, "stars.json")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	if err := json.NewEncoder(f).Encode(stars); err != nil {
		_ = f.Close()
		return fmt.Errorf("encoding %s: %w", out, err)
	}
	return f.Close()
}

func (r *RepositoryDownloader) downloadRepository(repository *github.Repository) error {
	if slices.Contains(blockedRepos, repository.GetFullName()) {
		return fmt.Errorf("repository %s is blocked", repository.GetFullName())
	}

	cloneURL := repository.GetCloneURL()
	log.Printf("Clone from: %s\n", cloneURL)

	cloneInto := filepath.Join(r.config.DownloadDir, *repository.FullName)
	log.Printf("Clone into: %s\n", cloneInto)

	if exists(cloneInto) {
		log.Printf("Repository already exists: %s\n", cloneInto)
		// TODO: Fetch latest changes
		return nil
	}

	err := exec.Command("git", "clone", "--depth", "1", "--filter=blob:none", "--sparse", cloneURL, cloneInto).Run()
	if err != nil {
		return fmt.Errorf("cloning repository %s into %s: %w", cloneURL, cloneInto, err)
	}

	err = exec.Command("git", "-C", cloneInto, "sparse-checkout", "set", ".github").Run()
	if err != nil {
		return fmt.Errorf("setting sparse checkout for %s: %w", cloneInto, err)
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

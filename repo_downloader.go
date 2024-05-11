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
	downloaded chan string
}

func NewRepositoryDownloader(client *github.Client, config Config) *RepositoryDownloader {
	return &RepositoryDownloader{
		client: client,
		config: config,
	}
}

func (r *RepositoryDownloader) Download(ctx context.Context, downloaded chan string) error {
	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: r.config.PerPage,
		},
	}

	ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	for i := 0; i < r.config.MaxPages; i++ {
		searchResult, resp, err := r.client.Search.Repositories(ctx, r.config.Query, opts)
		if err != nil {
			return fmt.Errorf("searching repositories: %w", err)
		}

		if searchResult == nil {
			return fmt.Errorf("No search results")
		}

		for _, repository := range searchResult.Repositories {
			if *repository.Fork {
				continue
			}

			cloneURL := repository.GetCloneURL()
			log.Printf("Clone from: %s\n", cloneURL)

			cloneInto := filepath.Join(r.config.DownloadDir, *repository.FullName)
			log.Printf("Clone into: %s\n", cloneInto)

			if !exists(cloneInto) {
				err := exec.Command("git", "clone", "--depth", "1", cloneURL, cloneInto).Run()
				if err != nil {
					return fmt.Errorf("cloning repository: %w", err)
				}
			} else {
				log.Printf("Repository already exists: %s\n", cloneInto)
			}

			downloaded <- *repository.FullName
		}

		opts.Page = resp.NextPage
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

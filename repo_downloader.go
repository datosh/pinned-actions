package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
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
			log.Printf("Cloning into: %s\n", cloneInto)

			_, err := git.PlainClone(cloneInto, false, &git.CloneOptions{
				URL:   cloneURL,
				Depth: 1,
			})
			// TODO: if repo exists update instead
			// https://github.com/go-git/go-git/blob/master/_examples/pull/main.go
			if err != nil && err != git.ErrRepositoryAlreadyExists {
				return fmt.Errorf("cloning repository: %w", err)
			}

			downloaded <- *repository.FullName
		}

		opts.Page = resp.NextPage
	}

	return nil
}

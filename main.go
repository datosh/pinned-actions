package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/google/go-github/v60/github"
)

func main() {
	// TODO: Caching for development https://github.com/google/go-github?tab=readme-ov-file#conditional-requests
	client := github.NewClient(nil)
	// TODO: Actual pagination https://github.com/google/go-github?tab=readme-ov-file#pagination
	searchResult, resp, err := client.Search.Repositories(
		context.Background(),
		"stars:>1000",
		&github.SearchOptions{
			Sort:  "stars",
			Order: "desc",
			ListOptions: github.ListOptions{
				PerPage: 5,
			},
		})
	if err != nil {
		log.Fatalf("fetching repositories: %v", err)
	}

	log.Printf("Retrieved page: %d\n", resp.FirstPage)
	if searchResult != nil {
		for _, repository := range searchResult.Repositories {
			if *repository.Fork {
				log.Println("Skipping fork")
				continue
			}
			cloneURL := repository.GetCloneURL()
			log.Printf("Clone at: %s\n", cloneURL)
			FetchWorkflows(cloneURL, *repository.Name)
		}
	}
}

func FetchWorkflows(cloneURL, repoName string) {
	cloneInto := filepath.Join("/tmp/pinned", repoName)
	_, err := git.PlainClone(cloneInto, false, &git.CloneOptions{
		URL:      cloneURL,
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		log.Fatalf("cloning '%s': %v", cloneURL, err)
	}

	workflowsFolder := filepath.Join(cloneInto, ".github/workflows")
	ListFiles(workflowsFolder)
	// TODO: eventually remove tmp folder
}

func ListFiles(directory string) error {
	d, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("opening directory: %w", err)
	}
	defer d.Close()

	files, err := d.Readdir(-1)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}

	log.Println("Files in directory:")
	for _, file := range files {
		log.Println(file.Name())
	}
	return nil
}

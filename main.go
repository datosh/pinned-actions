package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/google/go-github/v62/github"
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
				analysis, err := AnalyseRepository(config.DownloadDir, repo)
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

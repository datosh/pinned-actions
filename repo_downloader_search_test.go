package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-github/v93/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	Name  string
	Stars int
}

// newSearchServer serves GitHub's repository search for the given repos. The
// first incomplete requests answer with incomplete_results=true; a negative
// value makes every request incomplete. requests counts all search calls.
func newSearchServer(t *testing.T, repos []fakeRepo, incomplete int32, requests *atomic.Int32) *github.Client {
	t.Helper()

	sorted := append([]fakeRepo(nil), repos...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Stars != sorted[j].Stars {
			return sorted[i].Stars > sorted[j].Stars
		}
		return sorted[i].Name < sorted[j].Name
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !strings.HasSuffix(req.URL.Path, "/search/repositories") {
			http.NotFound(w, req)
			return
		}
		n := requests.Add(1)

		var minStars, maxStars int
		_, err := fmt.Sscanf(req.URL.Query().Get("q"), "stars:%d..%d", &minStars, &maxStars)
		assert.NoError(t, err)
		perPage := 30
		if v := req.URL.Query().Get("per_page"); v != "" {
			perPage, err = strconv.Atoi(v)
			assert.NoError(t, err)
		}

		items := []map[string]any{}
		for _, repo := range sorted {
			if repo.Stars >= minStars && repo.Stars <= maxStars && len(items) < perPage {
				items = append(items, map[string]any{"full_name": repo.Name, "stargazers_count": repo.Stars})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{
			"total_count":        len(items),
			"incomplete_results": incomplete < 0 || n <= incomplete,
			"items":              items,
		}))
	}))
	t.Cleanup(srv.Close)

	base := srv.URL + "/"
	client, err := github.NewClient(github.WithURLs(&base, &base))
	require.NoError(t, err)
	return client
}

func newTestDownloader(t *testing.T, client *github.Client, perPage int) *RepositoryDownloader {
	t.Helper()
	config := NewConfig()
	config.PerPage = perPage
	config.MaxPages = 10
	config.ResultDir = t.TempDir()

	d := NewRepositoryDownloader(client, *config)
	d.clone = func(context.Context, *github.Repository) error { return nil }
	d.retryDelay = 0
	return d
}

func readStars(t *testing.T, d *RepositoryDownloader) map[string]int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(d.config.ResultDir, "stars.json"))
	require.NoError(t, err)
	var stars map[string]int
	require.NoError(t, json.Unmarshal(data, &stars))
	return stars
}

func TestDownloadTiedStarsAtPageBoundary(t *testing.T) {
	// b and c are tied; with two repos per page, c falls off the first page.
	repos := []fakeRepo{
		{"o/a", 5000}, {"o/b", 4000}, {"o/c", 4000},
		{"o/d", 3000}, {"o/e", 3000}, {"o/f", 2000},
	}
	var requests atomic.Int32
	d := newTestDownloader(t, newSearchServer(t, repos, 0, &requests), 2)

	downloaded := make(chan string, len(repos))
	require.NoError(t, d.Download(context.Background(), downloaded))
	close(downloaded)

	var got []string
	for name := range downloaded {
		got = append(got, name)
	}
	assert.Equal(t, []string{"o/a", "o/b", "o/c", "o/d", "o/e", "o/f"}, got)
	assert.Len(t, readStars(t, d), len(repos))
}

func TestDownloadRetriesIncompleteResults(t *testing.T) {
	repos := []fakeRepo{{"o/a", 5000}}
	var requests atomic.Int32
	d := newTestDownloader(t, newSearchServer(t, repos, 2, &requests), 10)

	downloaded := make(chan string, 10)
	require.NoError(t, d.Download(context.Background(), downloaded))

	assert.Equal(t, int32(3), requests.Load(), "two incomplete answers, then the real one")
	assert.Equal(t, map[string]int{"o/a": 5000}, readStars(t, d))
}

func TestDownloadGivesUpOnPersistentlyIncompleteResults(t *testing.T) {
	var requests atomic.Int32
	d := newTestDownloader(t, newSearchServer(t, []fakeRepo{{"o/a", 5000}}, -1, &requests), 10)

	err := d.Download(context.Background(), make(chan string, 10))

	assert.ErrorContains(t, err, "incomplete results")
	assert.Equal(t, int32(maxIncompleteRetries+1), requests.Load())
}

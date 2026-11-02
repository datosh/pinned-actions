package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/google/go-github/v93/github"
)

var blockedRepos = []string{
	"cdnjs/cdnjs",
	"go-xorm/xorm",
}

// maxIncompleteRetries bounds how often one search is repeated when GitHub
// reports incomplete results.
const maxIncompleteRetries = 5

type RepositoryDownloader struct {
	client *github.Client
	config Config
	// clone fetches one repository; replaced in tests.
	clone func(context.Context, *github.Repository) error
	// retryDelay is the first wait before retrying incomplete results; it
	// doubles with every further retry.
	retryDelay time.Duration
}

func NewRepositoryDownloader(client *github.Client, config Config) *RepositoryDownloader {
	r := &RepositoryDownloader{
		client:     client,
		config:     config,
		retryDelay: 2 * time.Second,
	}
	r.clone = r.downloadRepository
	return r
}

func (r *RepositoryDownloader) Download(ctx context.Context, downloaded chan string) (err error) {
	ctx = context.WithValue(ctx, github.SleepUntilPrimaryRateLimitResetWhenRateLimited, true)

	opts := &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			PerPage: r.config.PerPage,
		},
	}

	maxStars := r.config.MaxStars
	minStars := r.config.MinStars

	stars := make(map[string]int)
	// Persist the stars of everything seen so far, even if the search fails.
	defer func() {
		if werr := r.writeStars(stars); werr != nil {
			err = errors.Join(err, fmt.Errorf("writing stars: %w", werr))
		}
	}()

	incompleteRetries := 0
	for i := 0; i < r.config.MaxPages; i++ {
		query := buildQuery(minStars, maxStars)
		slog.Info("searching repositories", "query", query)
		searchResult, _, err := r.client.Search.Repositories(ctx, query, opts)
		if err != nil {
			return fmt.Errorf("searching repositories: %w", err)
		}

		if searchResult == nil {
			return fmt.Errorf("no search results")
		}

		if searchResult.GetIncompleteResults() {
			if incompleteRetries >= maxIncompleteRetries {
				return fmt.Errorf("incomplete results for query %q after %d retries", query, incompleteRetries)
			}
			delay := r.retryDelay << incompleteRetries
			incompleteRetries++
			slog.Warn("GitHub returned incomplete results; retrying", "query", query, "delay", delay)
			if err := sleepContext(ctx, delay); err != nil {
				return err
			}
			i-- // a retry does not use up a page
			continue
		}
		incompleteRetries = 0

		slog.Info("found repositories", "count", len(searchResult.Repositories))

		newRepos := 0
		lastStars := maxStars
		for _, repository := range searchResult.Repositories {
			if err := ctx.Err(); err != nil {
				return err
			}
			name := repository.GetFullName()
			lastStars = repository.GetStargazersCount()
			if _, seen := stars[name]; seen {
				continue
			}
			newRepos++
			stars[name] = lastStars
			if err := r.clone(ctx, repository); err != nil {
				slog.Error("downloading repository failed", "repo", name, "error", err)
				continue
			}
			downloaded <- name
		}

		if len(searchResult.Repositories) < r.config.PerPage {
			slog.Info("handled all pages", "requests", i+1)
			break
		}

		// Repos tied with the last one on this page may not have fit on it, so
		// the next query includes lastStars again; repos already seen are skipped.
		// If a full page brought nothing new, the tie is larger than a page and
		// we have to move on, losing the remainder of it.
		maxStars = lastStars
		if newRepos == 0 {
			slog.Warn("more repositories tied on stars than fit on a page; some are skipped", "per_page", r.config.PerPage, "stars", lastStars)
			maxStars--
		}
	}

	return nil
}

// sleepContext waits for d or until ctx is cancelled.
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (r *RepositoryDownloader) writeStars(stars map[string]int) error {
	return writeJSON(filepath.Join(r.config.ResultDir, "stars.json"), stars)
}

func (r *RepositoryDownloader) downloadRepository(ctx context.Context, repository *github.Repository) error {
	if slices.Contains(blockedRepos, repository.GetFullName()) {
		return fmt.Errorf("repository %s is blocked", repository.GetFullName())
	}

	cloneURL := repository.GetCloneURL()
	cloneInto := filepath.Join(r.config.DownloadDir, *repository.FullName)

	cloned, err := exists(cloneInto)
	if err != nil {
		return fmt.Errorf("checking %s: %w", cloneInto, err)
	}
	if cloned {
		err := refreshClone(ctx, cloneInto)
		if err == nil {
			return nil
		}
		// A broken clone (e.g. from an interrupted earlier run) is replaced.
		slog.Warn("refreshing clone failed, cloning again", "path", cloneInto, "error", err)
		if err := os.RemoveAll(cloneInto); err != nil {
			return fmt.Errorf("removing broken clone %s: %w", cloneInto, err)
		}
	}

	slog.Info("cloning repository", "from", cloneURL, "into", cloneInto)
	err = runGit(ctx, "clone", "--depth", "1", "--filter=blob:none", "--sparse", cloneURL, cloneInto)
	if err != nil {
		return fmt.Errorf("cloning repository %s into %s: %w", cloneURL, cloneInto, err)
	}

	err = runGit(ctx, "-C", cloneInto, "sparse-checkout", "set", ".github")
	if err != nil {
		return fmt.Errorf("setting sparse checkout for %s: %w", cloneInto, err)
	}

	return nil
}

// refreshClone moves an existing shallow clone to the current remote HEAD, so
// repeated scans do not analyze stale checkouts.
func refreshClone(ctx context.Context, dir string) error {
	slog.Info("refreshing existing clone", "path", dir)
	if err := runGit(ctx, "-C", dir, "fetch", "--depth", "1", "origin", "HEAD"); err != nil {
		return err
	}
	return runGit(ctx, "-C", dir, "reset", "--hard", "FETCH_HEAD")
}

// runGit runs git and includes its output in the returned error.
func runGit(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, bytes.TrimSpace(out))
	}
	return nil
}

// exists returns whether the given file or directory exists. A path below a
// regular file (ENOTDIR) does not exist; any other stat failure, such as a
// permission error, is returned to the caller.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return false, nil
	}
	return false, err
}

func buildQuery(minStars, maxStars int) string {
	return fmt.Sprintf("stars:%d..%d", minStars, maxStars)
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// lockfileName is the GitHub Actions workflow lockfile, kept next to the
// workflows it locks.
const lockfileName = "actions.lock"

// LockfileResult records whether a repository uses GitHub Actions workflow
// lockfiles, and for how many of its workflows.
type LockfileResult struct {
	Repository  string `json:"repository"`
	HasLockfile bool   `json:"has_lockfile"`
	// Version is the lockfile's schema version, e.g. v0.0.3.
	Version        string `json:"version,omitempty"`
	WorkflowsTotal int    `json:"workflows_total"`
	// WorkflowsLocked counts the workflows the lockfile lists.
	WorkflowsLocked int `json:"workflows_locked"`
	// Malformed is set when the lockfile exists but cannot be read as one.
	Malformed bool `json:"malformed,omitempty"`
}

// LockfileAnalyzer inspects .github/workflows/actions.lock of each repository.
type LockfileAnalyzer struct{}

func NewLockfileAnalyzer() *LockfileAnalyzer { return &LockfileAnalyzer{} }

func (l *LockfileAnalyzer) Name() string { return "lockfile" }

func (l *LockfileAnalyzer) Analyze(_ context.Context, repoPath, repo string) (any, error) {
	result := LockfileResult{Repository: repo}

	workflowsDir := filepath.Join(repoPath, repo, ".github", "workflows")
	hasDir, err := exists(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("checking %s: %w", workflowsDir, err)
	}
	if !hasDir {
		return result, nil
	}

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", workflowsDir, err)
	}
	workflows := make(map[string]bool) // repo-relative paths, as the lockfile writes them
	for _, entry := range entries {
		// GitHub only runs workflow files directly inside the directory.
		if ext := filepath.Ext(entry.Name()); !entry.IsDir() && (ext == ".yml" || ext == ".yaml") {
			workflows[".github/workflows/"+entry.Name()] = true
		}
	}
	result.WorkflowsTotal = len(workflows)

	data, err := os.ReadFile(filepath.Join(workflowsDir, lockfileName))
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading lockfile in %s: %w", workflowsDir, err)
	}
	result.HasLockfile = true

	result.Version, result.WorkflowsLocked, err = parseLockfile(data, workflows)
	if err != nil {
		slog.Warn("malformed lockfile", "repo", repo, "error", err)
		result.Malformed = true
	}
	return result, nil
}

func (l *LockfileAnalyzer) Close() error { return nil }

// parseLockfile returns the schema version of a lockfile and how many of the
// given workflow files (repo-relative paths) it lists.
func parseLockfile(data []byte, workflows map[string]bool) (version string, locked int, err error) {
	var doc struct {
		Version   string              `yaml:"version"`
		Workflows map[string][]string `yaml:"workflows"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return "", 0, err
	}
	if doc.Version == "" {
		return "", 0, errors.New("no version")
	}
	for path := range doc.Workflows {
		if workflows[path] {
			locked++
		}
	}
	return doc.Version, locked, nil
}

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// lockfileName is the GitHub Actions workflow lockfile, kept next to the
// workflows it locks.
const lockfileName = "actions.lock"

// LockfileResult records whether a repository uses GitHub Actions workflow
// lockfiles, and for how many of its workflows.
type LockfileResult struct {
	Repository     string `json:"repository"`
	HasLockfile    bool   `json:"has_lockfile"`
	WorkflowsTotal int    `json:"workflows_total"`
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

	result.HasLockfile, err = exists(filepath.Join(workflowsDir, lockfileName))
	if err != nil {
		return nil, fmt.Errorf("checking lockfile in %s: %w", workflowsDir, err)
	}

	entries, err := os.ReadDir(workflowsDir)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", workflowsDir, err)
	}
	for _, entry := range entries {
		// GitHub only runs workflow files directly inside the directory.
		if ext := filepath.Ext(entry.Name()); !entry.IsDir() && (ext == ".yml" || ext == ".yaml") {
			result.WorkflowsTotal++
		}
	}
	return result, nil
}

func (l *LockfileAnalyzer) Close() error { return nil }

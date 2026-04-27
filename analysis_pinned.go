package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/stacklok/frizbee/pkg/replacer"
	fzconfig "github.com/stacklok/frizbee/pkg/utils/config"
)

type Analysis struct {
	Repository    string `json:"repository"`
	ActionsPinned int    `json:"actions_pinned"`
	ActionsTotal  int    `json:"actions_total"`
	HasRenovate   bool   `json:"has_renovate"`
	HasDependabot bool   `json:"has_dependabot"`
}

// PinnedAnalyzer checks what fraction of GitHub Actions in each repository
// are pinned to a full-length commit SHA or OCI digest.
type PinnedAnalyzer struct {
	resultDir string
	results   []Analysis
}

func NewPinnedAnalyzer(resultDir string) *PinnedAnalyzer {
	return &PinnedAnalyzer{resultDir: resultDir}
}

func (p *PinnedAnalyzer) Name() string { return "pinned" }

func (p *PinnedAnalyzer) Analyze(_ context.Context, repoPath, repo string) error {
	analysis, err := analyseRepository(repoPath, repo)
	if err != nil {
		return err
	}
	p.results = append(p.results, analysis)
	return nil
}

func (p *PinnedAnalyzer) Close() error {
	out := filepath.Join(p.resultDir, "pinned.json")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	if err := json.NewEncoder(f).Encode(p.results); err != nil {
		_ = f.Close()
		return fmt.Errorf("encoding %s: %w", out, err)
	}
	return f.Close()
}

func analyseRepository(dir string, repo string) (Analysis, error) {
	analysis := Analysis{Repository: repo}

	repoPath := filepath.Join(dir, repo)
	workflowsPath := filepath.Join(repoPath, ".github", "workflows")

	r := replacer.NewGitHubActionsReplacer(&fzconfig.Config{})

	actions, err := r.ListPath(workflowsPath)
	if err != nil {
		return analysis, fmt.Errorf("listing actions: %w", err)
	}

	for _, action := range actions.Entities {
		switch action.Type {
		case "container":
			if len(action.Ref) == 71 && strings.HasPrefix(action.Ref, "sha256:") {
				analysis.ActionsPinned++
				analysis.ActionsTotal++
			} else {
				analysis.ActionsTotal++
			}
		case "action":
			if len(action.Ref) == 40 && isHex(action.Ref) {
				analysis.ActionsPinned++
				analysis.ActionsTotal++
			} else {
				analysis.ActionsTotal++
			}
		default:
			log.Printf("[WARN] unknown action type: %s", action.Type)
		}
	}

	// https://docs.renovatebot.com/configuration-options/
	renovatePaths := []string{
		filepath.Join(repoPath, "renovate.json"),
		filepath.Join(repoPath, "renovate.json5"),
		filepath.Join(repoPath, ".github", "renovate.json"),
		filepath.Join(repoPath, ".github", "renovate.json5"),
		filepath.Join(repoPath, ".gitlab", "renovate.json"),
		filepath.Join(repoPath, ".gitlab", "renovate.json5"),
		filepath.Join(repoPath, ".renovaterc"),
		filepath.Join(repoPath, ".renovaterc.json"),
		filepath.Join(repoPath, ".renovaterc.json5"),
	}
	analysis.HasRenovate = slices.ContainsFunc(renovatePaths, func(path string) bool {
		return exists(path)
	})

	dependabotPath := filepath.Join(repoPath, ".github", "dependabot.yml")
	if exists(dependabotPath) {
		analysis.HasDependabot = true
	}

	return analysis, nil
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

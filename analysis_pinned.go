package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/stacklok/frizbee/pkg/interfaces"
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

	// Workflows and composite actions both reference other actions via `uses:`.
	err := countActions([]string{
		filepath.Join(repoPath, ".github", "workflows"),
		filepath.Join(repoPath, ".github", "actions"),
	}, &analysis)
	if err != nil {
		return analysis, err
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
	for _, path := range renovatePaths {
		found, err := exists(path)
		if err != nil {
			return analysis, fmt.Errorf("checking renovate config: %w", err)
		}
		if found {
			analysis.HasRenovate = true
			break
		}
	}

	dependabotPath := filepath.Join(repoPath, ".github", "dependabot.yml")
	analysis.HasDependabot, err = exists(dependabotPath)
	if err != nil {
		return analysis, fmt.Errorf("checking dependabot config: %w", err)
	}

	return analysis, nil
}

// countActions adds the distinct actions referenced below dirs to analysis.
// Missing directories are skipped: a repository without them uses no actions.
func countActions(dirs []string, analysis *Analysis) error {
	r := replacer.NewGitHubActionsReplacer(&fzconfig.Config{})

	// frizbee de-duplicates references per call; do the same across dirs so a
	// reference used in both a workflow and a composite action counts once.
	seen := make(map[string]bool)
	var entities []interfaces.EntityRef
	for _, dir := range dirs {
		found, err := exists(dir)
		if err != nil {
			return fmt.Errorf("checking %s: %w", dir, err)
		}
		if !found {
			continue
		}
		listed, err := r.ListPath(dir)
		if err != nil {
			return fmt.Errorf("listing actions in %s: %w", dir, err)
		}
		for _, e := range listed.Entities {
			key := e.Type + "|" + e.Name + "@" + e.Ref
			if !seen[key] {
				seen[key] = true
				entities = append(entities, e)
			}
		}
	}

	for _, action := range entities {
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
	return nil
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

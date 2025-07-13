package main

import (
	"fmt"
	"log"
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

func NewAnalysis(repository string) Analysis {
	return Analysis{
		Repository:    repository,
		ActionsPinned: 0,
		ActionsTotal:  0,
		HasRenovate:   false,
		HasDependabot: false,
	}
}

func (a *Analysis) CountPinned() {
	a.ActionsPinned++
	a.ActionsTotal++
}

func (a *Analysis) CountUnpinned() {
	a.ActionsTotal++
}

func (a Analysis) String() string {
	updater := "None"
	if a.HasRenovate {
		updater = "Renovate"
	}
	if a.HasDependabot {
		updater = "Dependabot"
	}
	return fmt.Sprintf("%s: %d/%d (%s)", a.Repository, a.ActionsPinned, a.ActionsTotal, updater)
}

func AnalyseRepository(dir string, repo string) (Analysis, error) {
	analysis := NewAnalysis(repo)

	repoPath := filepath.Join(dir, repo)
	workflowsPath := filepath.Join(repoPath, ".github", "workflows")

	// Create a new Frizbee instance
	r := replacer.NewGitHubActionsReplacer(&fzconfig.Config{})

	actions, err := r.ListPath(workflowsPath)
	if err != nil {
		return analysis, fmt.Errorf("listing actions: %w", err)
	}

	for _, action := range actions.Entities {
		switch action.Type {
		case "container":
			if len(action.Ref) == 71 && strings.HasPrefix(action.Ref, "sha256:") {
				analysis.CountPinned()
			} else {
				analysis.CountUnpinned()
			}
		case "action":
			if len(action.Ref) == 40 && isHex(action.Ref) {
				analysis.CountPinned()
			} else {
				analysis.CountUnpinned()
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

	renovate := slices.ContainsFunc(renovatePaths, func(path string) bool {
		return exists(path)
	})
	analysis.HasRenovate = renovate

	dependabotPath := filepath.Join(repoPath, ".github", "dependabot.yml")
	if exists(dependabotPath) {
		analysis.HasDependabot = true
	}

	return analysis, nil
}

func isHex(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

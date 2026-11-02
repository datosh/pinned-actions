package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v93/github"
)

// Analyzer is implemented by every analysis pass that runs against a
// downloaded repository. Offline analyzers inspect the local checkout;
// online analyzers may additionally call external APIs.
//
// Analyze is called once per repository. Close is called exactly once after
// all repositories have been processed; implementations should flush any
// accumulated results to disk there.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, repoPath, repo string) error
	Close() error
}

// analyzerRegistry lists every analyzer selectable with --analyzer. To add
// one, append an entry here.
var analyzerRegistry = []struct {
	name  string
	build func(config Config, client *github.Client) (Analyzer, error)
}{
	{"pinned", func(config Config, _ *github.Client) (Analyzer, error) {
		return NewPinnedAnalyzer(config.ResultDir), nil
	}},
	{"zizmor", func(config Config, _ *github.Client) (Analyzer, error) {
		a, err := NewZizmorAnalyzer(config.ResultDir)
		if err != nil {
			return nil, err
		}
		return a, nil
	}},
	{"immutable", func(config Config, client *github.Client) (Analyzer, error) {
		return NewImmutableReleasesAnalyzer(client, config.ResultDir), nil
	}},
}

// analyzerNames returns the names of all registered analyzers.
func analyzerNames() []string {
	names := make([]string, 0, len(analyzerRegistry))
	for _, entry := range analyzerRegistry {
		names = append(names, entry.name)
	}
	return names
}

// buildAnalyzers creates the analyzers selected in config.Analyzers.
func buildAnalyzers(config Config, client *github.Client) ([]Analyzer, error) {
	var analyzers []Analyzer
	for _, name := range config.Analyzers {
		found := false
		for _, entry := range analyzerRegistry {
			if entry.name != name {
				continue
			}
			a, err := entry.build(config, client)
			if err != nil {
				return nil, err
			}
			analyzers = append(analyzers, a)
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("unknown analyzer %q (available: %s)", name, strings.Join(analyzerNames(), ", "))
		}
	}
	return analyzers, nil
}

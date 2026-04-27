package main

import "context"

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

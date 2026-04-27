package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ZizmorFinding is one security finding emitted by zizmor for a single
// workflow file.
type ZizmorFinding struct {
	Rule       string `json:"rule"`
	Severity   string `json:"severity"`
	Confidence string `json:"confidence"`
	File       string `json:"file"`
	Line       int    `json:"line"`
}

// ZizmorResult collects all findings for one repository.
type ZizmorResult struct {
	Repository string          `json:"repository"`
	Findings   []ZizmorFinding `json:"findings"`
}

// ZizmorAnalyzer runs zizmor against the .github/ directory of each
// repository. It requires the zizmor binary to be present on $PATH.
type ZizmorAnalyzer struct {
	zizmorBin string
	resultDir string
	results   []ZizmorResult
}

func NewZizmorAnalyzer(resultDir string) (*ZizmorAnalyzer, error) {
	bin, err := exec.LookPath("zizmor")
	if err != nil {
		return nil, fmt.Errorf("zizmor not found on $PATH: %w", err)
	}
	return &ZizmorAnalyzer{zizmorBin: bin, resultDir: resultDir}, nil
}

func (z *ZizmorAnalyzer) Name() string { return "zizmor" }

func (z *ZizmorAnalyzer) Analyze(_ context.Context, repoPath, repo string) error {
	githubDir := filepath.Join(repoPath, repo, ".github")
	if !exists(githubDir) {
		z.results = append(z.results, ZizmorResult{Repository: repo, Findings: []ZizmorFinding{}})
		return nil
	}

	var stdout bytes.Buffer
	cmd := exec.Command(z.zizmorBin,
		"--format=json",
		"--offline",
		"--no-exit-codes",
		githubDir,
	)
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Exit code 3 means zizmor found no auditable inputs (e.g. .github/
		// exists but contains only issue templates, not workflows or actions).
		// Treat this the same as an empty findings list.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 3 {
			z.results = append(z.results, ZizmorResult{Repository: repo, Findings: []ZizmorFinding{}})
			return nil
		}
		return fmt.Errorf("running zizmor: %w", err)
	}

	findings, err := parseZizmorOutput(stdout.Bytes(), repoPath, repo)
	if err != nil {
		return fmt.Errorf("parsing zizmor output: %w", err)
	}

	z.results = append(z.results, ZizmorResult{
		Repository: repo,
		Findings:   findings,
	})
	return nil
}

func (z *ZizmorAnalyzer) Close() error {
	out := filepath.Join(z.resultDir, "zizmor.json")
	f, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("creating %s: %w", out, err)
	}
	if err := json.NewEncoder(f).Encode(z.results); err != nil {
		_ = f.Close()
		return fmt.Errorf("encoding %s: %w", out, err)
	}
	return f.Close()
}

// zizmorRawFinding mirrors the relevant subset of zizmor's JSON output.
type zizmorRawFinding struct {
	Ident          string `json:"ident"`
	Determinations struct {
		Severity   string `json:"severity"`
		Confidence string `json:"confidence"`
	} `json:"determinations"`
	Locations []struct {
		Symbolic struct {
			Key struct {
				Local *struct {
					GivenPath string `json:"given_path"`
				} `json:"Local"`
			} `json:"key"`
		} `json:"symbolic"`
		Concrete struct {
			Location struct {
				StartPoint struct {
					Row int `json:"row"`
				} `json:"start_point"`
			} `json:"location"`
		} `json:"concrete"`
	} `json:"locations"`
}

func parseZizmorOutput(data []byte, repoPath, repo string) ([]ZizmorFinding, error) {
	var raw []zizmorRawFinding
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	absRepoPath := filepath.Join(repoPath, repo)
	var findings []ZizmorFinding
	for _, r := range raw {
		f := ZizmorFinding{
			Rule:       r.Ident,
			Severity:   r.Determinations.Severity,
			Confidence: r.Determinations.Confidence,
		}
		// Use the first location that has a concrete file path.
		for _, loc := range r.Locations {
			if loc.Symbolic.Key.Local != nil {
				f.File = strings.TrimPrefix(loc.Symbolic.Key.Local.GivenPath, absRepoPath+"/")
				f.Line = loc.Concrete.Location.StartPoint.Row
				break
			}
		}
		findings = append(findings, f)
	}
	return findings, nil
}

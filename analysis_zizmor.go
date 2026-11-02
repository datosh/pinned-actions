package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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
	UsesGHA    bool            `json:"uses_gha"`
	Findings   []ZizmorFinding `json:"findings"`
}

// ZizmorAnalyzer runs zizmor against the .github/ directory of each
// repository. It requires the zizmor binary to be present on $PATH.
type ZizmorAnalyzer struct {
	zizmorBin string
	resultDir string
	results   []ZizmorResult
}

// minZizmorVersion is the oldest zizmor whose JSON output parseZizmorOutput
// understands (see testdata/zizmor).
var minZizmorVersion = [3]int{1, 31, 1}

func NewZizmorAnalyzer(resultDir string) (*ZizmorAnalyzer, error) {
	bin, err := exec.LookPath("zizmor")
	if err != nil {
		return nil, fmt.Errorf("zizmor not found on $PATH: %w", err)
	}
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		return nil, fmt.Errorf("running zizmor --version: %w", err)
	}
	if err := checkZizmorVersion(string(out)); err != nil {
		return nil, err
	}
	return &ZizmorAnalyzer{zizmorBin: bin, resultDir: resultDir}, nil
}

// checkZizmorVersion verifies the output of `zizmor --version`
// (e.g. "zizmor 1.31.1") is at least minZizmorVersion.
func checkZizmorVersion(output string) error {
	fields := strings.Fields(output)
	if len(fields) != 2 {
		return fmt.Errorf("unexpected zizmor --version output %q", output)
	}
	// Drop any pre-release or build suffix, e.g. 1.32.0-rc.1.
	core, _, _ := strings.Cut(fields[1], "-")
	core, _, _ = strings.Cut(core, "+")

	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return fmt.Errorf("unexpected zizmor version %q", fields[1])
	}
	var version [3]int
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("unexpected zizmor version %q: %w", fields[1], err)
		}
		version[i] = n
	}

	if slices.Compare(version[:], minZizmorVersion[:]) < 0 {
		return fmt.Errorf("zizmor %s is too old; need at least %d.%d.%d",
			fields[1], minZizmorVersion[0], minZizmorVersion[1], minZizmorVersion[2])
	}
	return nil
}

func (z *ZizmorAnalyzer) Name() string { return "zizmor" }

// Analyze collects results itself and writes them in Close (not yet ported to
// per-repository results).
func (z *ZizmorAnalyzer) Analyze(ctx context.Context, repoPath, repo string) (any, error) {
	return nil, z.analyze(ctx, repoPath, repo)
}

func (z *ZizmorAnalyzer) analyze(ctx context.Context, repoPath, repo string) error {
	githubDir := filepath.Join(repoPath, repo, ".github")
	hasGithubDir, err := exists(githubDir)
	if err != nil {
		return fmt.Errorf("checking %s: %w", githubDir, err)
	}
	if !hasGithubDir {
		z.results = append(z.results, ZizmorResult{Repository: repo, UsesGHA: false, Findings: []ZizmorFinding{}})
		return nil
	}

	var stdout bytes.Buffer
	cmd := exec.CommandContext(ctx, z.zizmorBin,
		"--format=json",
		"--offline",
		"--no-exit-codes",
		// Only workflows and actions: a .github/ with just a dependabot.yml
		// must not count as using GitHub Actions.
		"--collect=workflows,actions",
		githubDir,
	)
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Exit code 3 means zizmor found no auditable inputs (e.g. .github/
		// exists but contains only issue templates or a dependabot.yml, not
		// workflows or actions). The repo does not use GHA in a meaningful way.
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 3 {
			z.results = append(z.results, ZizmorResult{Repository: repo, UsesGHA: false, Findings: []ZizmorFinding{}})
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
		UsesGHA:    true,
		Findings:   findings,
	})
	return nil
}

func (z *ZizmorAnalyzer) Close() error {
	return writeJSON(filepath.Join(z.resultDir, "zizmor.json"), z.results)
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
					VerbatimPath string `json:"verbatim_path"`
				} `json:"Local"`
			} `json:"key"`
			// Primary for the offending location, else e.g. Hidden or Related.
			Kind string `json:"kind"`
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
		// Prefer the primary location; the first one is often just context,
		// such as the whole step. Fall back to any local file location.
		best := -1
		for i, loc := range r.Locations {
			if loc.Symbolic.Key.Local == nil {
				continue
			}
			if loc.Symbolic.Kind == "Primary" {
				best = i
				break
			}
			if best == -1 {
				best = i
			}
		}
		if best >= 0 {
			loc := r.Locations[best]
			f.File = strings.TrimPrefix(loc.Symbolic.Key.Local.VerbatimPath, absRepoPath+"/")
			// zizmor rows are 0-based; editors and GitHub count from 1.
			f.Line = loc.Concrete.Location.StartPoint.Row + 1
		}
		findings = append(findings, f)
	}
	return findings, nil
}

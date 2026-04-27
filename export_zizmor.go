package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// ZizmorWebRule is one entry in the slimmed per-rule-count format written by
// export-zizmor. File and line information is intentionally omitted to keep
// the output small enough for a static website.
type ZizmorWebRule struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// ZizmorWebResult is one repository entry in the export-zizmor output.
type ZizmorWebResult struct {
	Repository string          `json:"repository"`
	UsesGHA    bool            `json:"uses_gha"`
	Rules      []ZizmorWebRule `json:"rules"`
}

// runExportZizmor implements the "export-zizmor" subcommand. It reads
// zizmor.json from the result directory, aggregates findings into per-rule
// counts (dropping file and line), and writes the slimmed output.
//
// Usage:
//
//	pinned-actions export-zizmor --result-dir=results/ --out=zizmor-web.json
func runExportZizmor(args []string) {
	fs := flag.NewFlagSet("export-zizmor", flag.ExitOnError)
	resultDir := fs.String("result-dir", "results", "directory containing analysis results")
	out := fs.String("out", "zizmor-web.json", "output file for the frontend")

	if err := fs.Parse(args); err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}

	in := filepath.Join(*resultDir, "zizmor.json")
	data, err := os.ReadFile(in)
	if err != nil {
		log.Fatalf("reading %s: %v", in, err)
	}

	var raw []ZizmorResult
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Fatalf("parsing %s: %v", in, err)
	}

	web := make([]ZizmorWebResult, 0, len(raw))
	for _, r := range raw {
		web = append(web, ZizmorWebResult{
			Repository: r.Repository,
			UsesGHA:    r.UsesGHA,
			Rules:      aggregateRules(r.Findings),
		})
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("creating %s: %v", *out, err)
	}
	if err := json.NewEncoder(f).Encode(web); err != nil {
		_ = f.Close()
		log.Fatalf("writing %s: %v", *out, err)
	}
	if err := f.Close(); err != nil {
		log.Fatalf("closing %s: %v", *out, err)
	}

	total := 0
	for _, r := range web {
		total += len(r.Rules)
	}
	fmt.Printf("Exported %d repositories (%d unique rule entries) to %s\n", len(web), total, *out)
}

// aggregateRules collapses a flat list of findings into per-(rule,severity)
// counts, preserving the rule's severity from the source data.
func aggregateRules(findings []ZizmorFinding) []ZizmorWebRule {
	type key struct{ rule, severity string }
	counts := make(map[key]int)
	// Preserve insertion order for deterministic output.
	var order []key
	for _, f := range findings {
		k := key{f.Rule, f.Severity}
		if counts[k] == 0 {
			order = append(order, k)
		}
		counts[k]++
	}
	rules := make([]ZizmorWebRule, 0, len(order))
	for _, k := range order {
		rules = append(rules, ZizmorWebRule{
			Rule:     k.rule,
			Severity: k.severity,
			Count:    counts[k],
		})
	}
	return rules
}

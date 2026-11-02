package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	Stars      int             `json:"stars"`
	Rules      []ZizmorWebRule `json:"rules"`
}

// runExportZizmor implements the "export-zizmor" subcommand. It reads
// zizmor.jsonl from the result directory, aggregates findings into per-rule
// counts (dropping file and line), and writes the slimmed output.
//
// Usage:
//
//	pinned-actions export-zizmor --result-dir=results/ --out=zizmor-web.js
func runExportZizmor(args []string) {
	a := parseExportArgs("export-zizmor", "zizmor-web.js", args)

	var stars map[string]int
	if err := readJSON(filepath.Join(a.ResultDir, "stars.json"), &stars); err != nil {
		fatal("export failed", "error", err)
	}

	raw, err := readJSONL[ZizmorResult](filepath.Join(a.ResultDir, "zizmor.jsonl"))
	if err != nil {
		fatal("export failed", "error", err)
	}

	web, failed := webResults(raw, stars)
	if failed > 0 {
		slog.Warn("skipped repositories zizmor could not analyze", "count", failed)
	}

	jsonData, err := json.Marshal(web)
	if err != nil {
		fatal("marshalling data", "error", err)
	}

	// The frontend loads this as a script, so it is JS rather than plain JSON.
	script := fmt.Sprintf("const ZIZMOR_DATA = %s;\n", jsonData)
	if err := os.WriteFile(a.Out, []byte(script), 0o644); err != nil {
		fatal("writing output", "path", a.Out, "error", err)
	}

	total := 0
	for _, r := range web {
		total += len(r.Rules)
	}
	fmt.Printf("Exported %d repositories (%d unique rule entries) to %s\n", len(web), total, a.Out)
}

// webResults converts zizmor results into the website format. Repositories
// zizmor failed on are left out, not shown as clean, and counted.
func webResults(raw []ZizmorResult, stars map[string]int) (web []ZizmorWebResult, failed int) {
	web = make([]ZizmorWebResult, 0, len(raw))
	for _, r := range raw {
		if r.Error != "" {
			failed++
			continue
		}
		web = append(web, ZizmorWebResult{
			Repository: r.Repository,
			UsesGHA:    r.UsesGHA,
			Stars:      stars[r.Repository],
			Rules:      aggregateRules(r.Findings),
		})
	}
	return web, failed
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

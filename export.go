package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// exportArgs are the flags shared by all export subcommands.
type exportArgs struct {
	ResultDir string
	Out       string
}

// parseExportArgs parses the --result-dir and --out flags of an export
// subcommand; defaultOut is the default output file. Subcommands with further
// flags register them in extra.
func parseExportArgs(name, defaultOut string, args []string, extra ...func(*flag.FlagSet)) exportArgs {
	var a exportArgs
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.StringVar(&a.ResultDir, "result-dir", "results", "directory containing analysis results")
	fs.StringVar(&a.Out, "out", defaultOut, "output file for the frontend")
	for _, register := range extra {
		register(fs)
	}

	if err := fs.Parse(args); err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}
	return a
}

// PinnedBuckets counts the repositories that use actions by how many of those
// actions are pinned. Repositories without any action are not counted.
type PinnedBuckets struct {
	Pinned          int `json:"pinned"`
	PartiallyPinned int `json:"partially-pinned"`
	Unpinned        int `json:"unpinned"`
}

// bucketPinned sorts repositories into the three disjoint buckets.
func bucketPinned(results []Analysis) PinnedBuckets {
	var b PinnedBuckets
	for _, r := range results {
		switch {
		case r.ActionsTotal == 0:
		case r.ActionsPinned == r.ActionsTotal:
			b.Pinned++
		case r.ActionsPinned > 0:
			b.PartiallyPinned++
		default:
			b.Unpinned++
		}
	}
	return b
}

// setYear records buckets under year in the JSON file at path (year -> counts),
// creating the file if needed and keeping the other years.
func setYear(path, year string, buckets PinnedBuckets) error {
	years := map[string]PinnedBuckets{}
	if err := readJSON(path, &years); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	years[year] = buckets

	data, err := json.MarshalIndent(years, "", "    ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// runExportPinned implements the "export-pinned" subcommand. It counts how
// many repositories in pinned.jsonl are fully, partially or not pinned and
// stores the counts under --year in the frontend's pinned.json, keeping the
// other years.
//
// Usage:
//
//	pinned-actions export-pinned --result-dir=results/ --year=2026
func runExportPinned(args []string) {
	var year, raw string
	a := parseExportArgs("export-pinned", "frontend/pinned.json", args, func(flags *flag.FlagSet) {
		flags.StringVar(&year, "year", "", "year to create or replace in the output file (required)")
		flags.StringVar(&raw, "raw", "", "also write all per-repository results to this file as a JSON array")
	})
	if _, err := strconv.Atoi(year); err != nil || len(year) != 4 {
		fatal("-year must be a four-digit year", "year", year)
	}

	results, err := readJSONL[Analysis](filepath.Join(a.ResultDir, "pinned.jsonl"))
	if err != nil {
		fatal("export failed", "error", err)
	}

	buckets := bucketPinned(results)
	if err := setYear(a.Out, year, buckets); err != nil {
		fatal("export failed", "error", err)
	}
	if raw != "" {
		if err := writeJSON(raw, results); err != nil {
			fatal("export failed", "error", err)
		}
	}

	fmt.Printf("Scanned %d repositories; %+v recorded for %s in %s\n", len(results), buckets, year, a.Out)
}

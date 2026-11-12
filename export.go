package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
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
// actions are pinned, plus the ones that use none. Fields added later are
// missing for scans recorded before they existed.
type PinnedBuckets struct {
	Pinned          int `json:"pinned"`
	PartiallyPinned int `json:"partially-pinned"`
	Unpinned        int `json:"unpinned"`
	// Locked counts repositories that are not fully SHA-pinned, but whose
	// lockfile covers all their actions.
	Locked    int `json:"locked,omitempty"`
	NoActions int `json:"no-actions,omitempty"`
	// WithLockfile counts all repositories that have an actions.lock. It is
	// only set when lockfile data was available, so that 0 differs from unknown.
	WithLockfile *int `json:"with-lockfile,omitempty"`
	// LockfileComplete counts the repositories whose lockfile covers every
	// workflow, whether or not their actions are also SHA-pinned. Set together
	// with WithLockfile.
	LockfileComplete *int   `json:"lockfile-complete,omitempty"`
	Scanned          string `json:"scanned,omitempty"` // YYYY-MM-DD
}

// String is the one-line summary printed by export-pinned. (%+v would print the
// address of the lockfile counts.)
func (b PinnedBuckets) String() string {
	count := func(n *int) string {
		if n == nil {
			return "unknown"
		}
		return strconv.Itoa(*n)
	}
	return fmt.Sprintf("pinned=%d locked=%d partially-pinned=%d unpinned=%d no-actions=%d with-lockfile=%s lockfile-complete=%s scanned=%s",
		b.Pinned, b.Locked, b.PartiallyPinned, b.Unpinned, b.NoActions, count(b.WithLockfile), count(b.LockfileComplete), b.Scanned)
}

// bucketPinned sorts repositories into the disjoint buckets. A repository
// whose actions are all SHA-pinned counts as pinned, else one with a complete
// lockfile as locked, else it is partially pinned or unpinned. lockfiles is
// keyed by repository; nil means there is no lockfile data, and a repository
// missing from it has no lockfile.
func bucketPinned(results []Analysis, lockfiles map[string]LockfileResult) PinnedBuckets {
	var b PinnedBuckets
	withLockfile, lockfileComplete := 0, 0
	for _, r := range results {
		lockfile := lockfiles[r.Repository]
		if lockfile.HasLockfile {
			withLockfile++
		}
		if lockfile.Complete {
			lockfileComplete++
		}
		switch {
		case r.ActionsTotal == 0:
			b.NoActions++
		case r.ActionsPinned == r.ActionsTotal:
			b.Pinned++
		case lockfile.Complete:
			b.Locked++
		case r.ActionsPinned > 0:
			b.PartiallyPinned++
		default:
			b.Unpinned++
		}
	}
	if lockfiles != nil {
		b.WithLockfile = &withLockfile
		b.LockfileComplete = &lockfileComplete
	}
	return b
}

// readLockfileResults reads lockfile.jsonl from dir by repository. The file
// is optional: scans without the lockfile analyzer give nil.
func readLockfileResults(dir string) (map[string]LockfileResult, error) {
	records, err := readJSONL[LockfileResult](filepath.Join(dir, "lockfile.jsonl"))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	byRepo := make(map[string]LockfileResult, len(records))
	for _, r := range records {
		byRepo[r.Repository] = r
	}
	return byRepo, nil
}

// ScanEntry is one scan in the frontend's scans.json: the counts of the scan,
// labeled with its month, and the archive with the raw results.
type ScanEntry struct {
	Label string `json:"label"` // YYYY-MM
	PinnedBuckets
	// Data is the file name of the downloadable archive, next to scans.json.
	Data string `json:"data,omitempty"`
}

// setScan records entry in the JSON file at path (a list of scans, oldest
// first), creating the file if needed. An existing scan with the same label is
// replaced, the other scans are kept.
func setScan(path string, entry ScanEntry) error {
	var scans []ScanEntry
	if err := readJSON(path, &scans); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	scans = slices.DeleteFunc(scans, func(s ScanEntry) bool { return s.Label == entry.Label })
	scans = append(scans, entry)
	slices.SortFunc(scans, func(a, b ScanEntry) int { return strings.Compare(a.Label, b.Label) })

	data, err := json.MarshalIndent(scans, "", "    ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// runExportPinned implements the "export-pinned" subcommand. It counts how
// many repositories in pinned.jsonl are fully, partially or not pinned and
// stores the counts under --label in the frontend's scans.json, keeping the
// other scans.
//
// Usage:
//
//	pinned-actions export-pinned --result-dir=results/ --label=2026-12 --data=results-2026-12.json.tar.gz
func runExportPinned(args []string) {
	var label, data, raw, scanned string
	a := parseExportArgs("export-pinned", "frontend/scans.json", args, func(flags *flag.FlagSet) {
		flags.StringVar(&label, "label", "", "month of the scan, YYYY-MM, to create or replace in the output file (default: month of --scanned)")
		flags.StringVar(&data, "data", "", "file name of the downloadable archive with the raw results, next to the output file")
		flags.StringVar(&raw, "raw", "", "also write all per-repository results to this file as a JSON array")
		flags.StringVar(&scanned, "scanned", "", "date of the scan, YYYY-MM-DD (default: last modification of pinned.jsonl)")
	})

	in := filepath.Join(a.ResultDir, "pinned.jsonl")
	if scanned == "" {
		info, err := os.Stat(in)
		if err != nil {
			fatal("export failed", "error", err)
		}
		scanned = info.ModTime().UTC().Format(time.DateOnly)
	} else if _, err := time.Parse(time.DateOnly, scanned); err != nil {
		fatal("-scanned must be a date like 2026-04-28", "scanned", scanned)
	}
	if label == "" {
		label = scanned[:len("2006-01")]
	} else if _, err := time.Parse("2006-01", label); err != nil {
		fatal("-label must be a month like 2026-04", "label", label)
	}

	results, err := readJSONL[Analysis](in)
	if err != nil {
		fatal("export failed", "error", err)
	}

	lockfiles, err := readLockfileResults(a.ResultDir)
	if err != nil {
		fatal("export failed", "error", err)
	}
	if lockfiles == nil {
		slog.Info("no lockfile.jsonl found; exporting without lockfile data")
	}

	buckets := bucketPinned(results, lockfiles)
	buckets.Scanned = scanned
	if err := setScan(a.Out, ScanEntry{Label: label, PinnedBuckets: buckets, Data: data}); err != nil {
		fatal("export failed", "error", err)
	}
	if raw != "" {
		if err := writeJSON(raw, results); err != nil {
			fatal("export failed", "error", err)
		}
	}

	fmt.Printf("Scanned %d repositories; %s recorded for %s in %s\n", len(results), buckets, label, a.Out)
}

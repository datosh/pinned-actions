package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// exportArgs are the flags shared by all export subcommands.
type exportArgs struct {
	ResultDir string
	Out       string
}

// parseExportArgs parses the --result-dir and --out flags of an export
// subcommand; defaultOut is the default output file.
func parseExportArgs(name, defaultOut string, args []string) exportArgs {
	var a exportArgs
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.StringVar(&a.ResultDir, "result-dir", "results", "directory containing analysis results")
	fs.StringVar(&a.Out, "out", defaultOut, "output file for the frontend")

	if err := fs.Parse(args); err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}
	return a
}

// runExportPinned implements the "export-pinned" subcommand. It reads
// pinned.jsonl from the result directory and writes it to a specified output
// file in the format expected by the frontend.
//
// Usage:
//
//	pinned-actions export-pinned --result-dir=results/ --out=web.json
func runExportPinned(args []string) {
	a := parseExportArgs("export-pinned", "web.json", args)

	// Reading into []Analysis also validates the file's shape.
	results, err := readJSONL[Analysis](filepath.Join(a.ResultDir, "pinned.jsonl"))
	if err != nil {
		fatal("export failed", "error", err)
	}

	if err := writeJSON(a.Out, results); err != nil {
		fatal("export failed", "error", err)
	}

	fmt.Printf("Exported %d repositories to %s\n", len(results), a.Out)
}

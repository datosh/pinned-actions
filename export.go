package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// runExport implements the "export" subcommand. It reads pinned.json from the
// result directory and writes it to a specified output file in the format
// expected by the frontend.
//
// Usage:
//
//	pinned-actions export --result-dir=results/ --out=web.json
func runExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	resultDir := fs.String("result-dir", "results", "directory containing analysis results")
	out := fs.String("out", "web.json", "output file for the frontend")

	if err := fs.Parse(args); err != nil {
		fs.PrintDefaults()
		os.Exit(1)
	}

	in := filepath.Join(*resultDir, "pinned.json")
	data, err := os.ReadFile(in)
	if err != nil {
		log.Fatalf("reading %s: %v", in, err)
	}

	// Validate: ensure the file is a JSON array of Analysis objects.
	var results []Analysis
	if err := json.Unmarshal(data, &results); err != nil {
		log.Fatalf("parsing %s: %v", in, err)
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatalf("creating %s: %v", *out, err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(results); err != nil {
		log.Fatalf("writing %s: %v", *out, err)
	}

	fmt.Printf("Exported %d repositories to %s\n", len(results), *out)
}

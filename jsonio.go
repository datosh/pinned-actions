package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// readJSON decodes the JSON file at path into v.
func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	return nil
}

// writeJSON encodes v as JSON into the file at path, replacing any existing file.
func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	if err := json.NewEncoder(f).Encode(v); err != nil {
		_ = f.Close()
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return f.Close()
}

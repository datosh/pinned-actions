package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

// jsonlWriter appends one JSON document per line to a file. Every Write goes
// straight to the file, so a crash loses at most the line in flight.
type jsonlWriter struct {
	f   *os.File
	enc *json.Encoder
}

// newJSONLWriter creates (or truncates) the file at path.
func newJSONLWriter(path string) (*jsonlWriter, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating %s: %w", path, err)
	}
	return &jsonlWriter{f: f, enc: json.NewEncoder(f)}, nil
}

func (w *jsonlWriter) Write(v any) error {
	if err := w.enc.Encode(v); err != nil {
		return fmt.Errorf("writing %s: %w", w.f.Name(), err)
	}
	return nil
}

func (w *jsonlWriter) Close() error { return w.f.Close() }

// readJSONL decodes a file with one JSON document per line into a slice.
func readJSONL[T any](path string) ([]T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	items := []T{}
	dec := json.NewDecoder(f)
	for {
		var item T
		err := dec.Decode(&item)
		if errors.Is(err, io.EOF) {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		items = append(items, item)
	}
}

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

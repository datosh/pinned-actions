package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

// resumeJSONL opens the file at path for appending and returns the
// repositories (the "repository" field of each line) it already contains. A
// missing file is created. A last line without its newline was cut off by a
// killed run; it is dropped so new lines start cleanly.
func resumeJSONL(path string) (*jsonlWriter, map[string]bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		w, err := newJSONLWriter(path)
		return w, map[string]bool{}, err
	}
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", path, err)
	}

	data = data[:bytes.LastIndexByte(data, '\n')+1]
	done := make(map[string]bool)
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var record struct {
			Repository string `json:"repository"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			return nil, nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		done[record.Repository] = true
	}

	if err := os.Truncate(path, int64(len(data))); err != nil {
		return nil, nil, fmt.Errorf("truncating %s: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("opening %s: %w", path, err)
	}
	return &jsonlWriter{f: f, enc: json.NewEncoder(f)}, done, nil
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

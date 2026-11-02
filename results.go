package main

import (
	"errors"
	"path/filepath"
)

// resultStore holds the result file of every analyzer of one scan, and which
// repositories each analyzer already has a result for.
type resultStore struct {
	writers []*jsonlWriter
	done    []map[string]bool // per analyzer, by repository
}

// openResults opens <resultDir>/<analyzer>.jsonl for every analyzer. Without
// resume the files are started from scratch; with resume existing results are
// kept and new ones appended.
func openResults(resultDir string, analyzers []Analyzer, resume bool) (*resultStore, error) {
	s := &resultStore{
		writers: make([]*jsonlWriter, len(analyzers)),
		done:    make([]map[string]bool, len(analyzers)),
	}
	for i, a := range analyzers {
		path := filepath.Join(resultDir, a.Name()+".jsonl")
		var err error
		if resume {
			s.writers[i], s.done[i], err = resumeJSONL(path)
		} else {
			s.writers[i], err = newJSONLWriter(path)
		}
		if err != nil {
			_ = s.Close()
			return nil, err
		}
	}
	return s, nil
}

// has reports whether analyzer i already has a result for repo.
func (s *resultStore) has(i int, repo string) bool { return s.done[i][repo] }

// complete reports whether every analyzer already has a result for repo.
func (s *resultStore) complete(repo string) bool {
	for i := range s.done {
		if !s.has(i, repo) {
			return false
		}
	}
	return len(s.done) > 0
}

// write appends result to the file of analyzer i.
func (s *resultStore) write(i int, result any) error { return s.writers[i].Write(result) }

func (s *resultStore) Close() error {
	var errs []error
	for _, w := range s.writers {
		if w != nil {
			errs = append(errs, w.Close())
		}
	}
	return errors.Join(errs...)
}

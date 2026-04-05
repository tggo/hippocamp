package rdfstore

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tggo/goRDFlib/trig"
)

// Save serializes the store to a TriG file and clears the dirty flag.
// The parent directory is created if it does not exist.
func Save(s *Store, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("save: create directory: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("save: create file: %w", err)
	}
	defer f.Close()

	if err := trig.SerializeDataset(s.Dataset(), f); err != nil {
		return fmt.Errorf("save: serialize: %w", err)
	}

	s.ClearDirty()
	return nil
}

// Load parses a TriG file into the store, merging with existing data.
func Load(s *Store, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("load: open file: %w", err)
	}
	defer f.Close()

	if err := trig.ParseDataset(s.Dataset(), f); err != nil {
		return fmt.Errorf("load: parse: %w", err)
	}

	return nil
}

// AutoLoad loads the file at path into the store if the file exists.
// Returns (true, nil) if the file was loaded, (false, nil) if not found.
func AutoLoad(s *Store, path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}
	if err := Load(s, path); err != nil {
		return false, err
	}
	return true, nil
}

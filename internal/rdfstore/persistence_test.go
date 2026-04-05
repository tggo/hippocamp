package rdfstore_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestSaveLoad_RoundTrip(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "en", "")
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/knows", "http://example.org/Bob", "uri", "", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "test.trig")

	if err := rdfstore.Save(s, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// dirty should be cleared after save
	if s.IsDirty() {
		t.Error("store should not be dirty after Save")
	}

	// load into a fresh store
	s2 := rdfstore.NewStore()
	if err := rdfstore.Load(s2, path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	triples, err := s2.ListTriples("", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(triples) != 2 {
		t.Errorf("expected 2 triples after load, got %d", len(triples))
	}
}

func TestSaveLoad_NamedGraphs(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.CreateGraph("http://example.org/g1")
	_ = s.CreateGraph("http://example.org/g2")
	_ = s.AddTriple("http://example.org/g1", "http://a.org/s", "http://a.org/p", "http://a.org/o1", "uri", "", "")
	_ = s.AddTriple("http://example.org/g2", "http://a.org/s", "http://a.org/p", "http://a.org/o2", "uri", "", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "named.trig")

	if err := rdfstore.Save(s, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	s2 := rdfstore.NewStore()
	if err := rdfstore.Load(s2, path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	g1triples, _ := s2.ListTriples("http://example.org/g1", "", "", "")
	g2triples, _ := s2.ListTriples("http://example.org/g2", "", "", "")

	if len(g1triples) != 1 {
		t.Errorf("g1 expected 1 triple, got %d", len(g1triples))
	}
	if len(g2triples) != 1 {
		t.Errorf("g2 expected 1 triple, got %d", len(g2triples))
	}
}

func TestLoad_MissingFile(t *testing.T) {
	s := rdfstore.NewStore()
	err := rdfstore.Load(s, "/nonexistent/file.trig")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAutoLoad(t *testing.T) {
	// Prepare a file
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/x", "http://example.org/y", "http://example.org/z", "uri", "", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "auto.trig")
	if err := rdfstore.Save(s, path); err != nil {
		t.Fatal(err)
	}

	// AutoLoad should load the file if it exists
	s2 := rdfstore.NewStore()
	loaded, err := rdfstore.AutoLoad(s2, path)
	if err != nil {
		t.Fatalf("AutoLoad: %v", err)
	}
	if !loaded {
		t.Error("expected AutoLoad to return true for existing file")
	}
	triples, _ := s2.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple after AutoLoad, got %d", len(triples))
	}

	// AutoLoad on non-existent file should return false, no error
	s3 := rdfstore.NewStore()
	loaded, err = rdfstore.AutoLoad(s3, "/nonexistent/path.trig")
	if err != nil {
		t.Fatalf("AutoLoad on missing file: %v", err)
	}
	if loaded {
		t.Error("expected AutoLoad to return false for missing file")
	}
}

func TestSave_CreatesDirectory(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "file.trig")

	if err := rdfstore.Save(s, path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

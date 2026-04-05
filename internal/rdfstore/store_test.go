package rdfstore_test

import (
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestNewStore(t *testing.T) {
	s := rdfstore.NewStore()
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	// default graph must exist
	graphs := s.ListGraphs()
	if len(graphs) == 0 {
		t.Error("expected at least one (default) graph")
	}
}

func TestAddAndListTriples(t *testing.T) {
	s := rdfstore.NewStore()

	err := s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")
	if err != nil {
		t.Fatalf("AddTriple: %v", err)
	}

	triples, err := s.ListTriples("", "", "", "")
	if err != nil {
		t.Fatalf("ListTriples: %v", err)
	}
	if len(triples) != 1 {
		t.Errorf("expected 1 triple, got %d", len(triples))
	}
	if triples[0].Subject != "http://example.org/Alice" {
		t.Errorf("unexpected subject: %q", triples[0].Subject)
	}
}

func TestRemoveTriple(t *testing.T) {
	s := rdfstore.NewStore()

	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")
	err := s.RemoveTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o")
	if err != nil {
		t.Fatalf("RemoveTriple: %v", err)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("expected 0 triples, got %d", len(triples))
	}
}

func TestListTriples_Wildcard(t *testing.T) {
	s := rdfstore.NewStore()

	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p1", "http://a.org/o1", "uri", "", "")
	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p2", "http://a.org/o2", "uri", "", "")

	// filter by subject only
	triples, err := s.ListTriples("", "http://a.org/s", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(triples) != 2 {
		t.Errorf("expected 2 triples, got %d", len(triples))
	}

	// filter by predicate
	triples, _ = s.ListTriples("", "", "http://a.org/p1", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple, got %d", len(triples))
	}
}

func TestCreateAndDeleteGraph(t *testing.T) {
	s := rdfstore.NewStore()

	if err := s.CreateGraph("http://example.org/g1"); err != nil {
		t.Fatalf("CreateGraph: %v", err)
	}

	graphs := s.ListGraphs()
	found := false
	for _, g := range graphs {
		if g == "http://example.org/g1" {
			found = true
		}
	}
	if !found {
		t.Error("created graph not in list")
	}

	if err := s.DeleteGraph("http://example.org/g1"); err != nil {
		t.Fatalf("DeleteGraph: %v", err)
	}

	graphs = s.ListGraphs()
	for _, g := range graphs {
		if g == "http://example.org/g1" {
			t.Error("deleted graph still in list")
		}
	}
}

func TestGraphIsolation(t *testing.T) {
	s := rdfstore.NewStore()

	_ = s.CreateGraph("http://example.org/g1")
	_ = s.AddTriple("http://example.org/g1", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")

	// default graph should be empty
	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("default graph should be empty, got %d triples", len(triples))
	}

	// named graph should have the triple
	triples, _ = s.ListTriples("http://example.org/g1", "", "", "")
	if len(triples) != 1 {
		t.Errorf("g1 should have 1 triple, got %d", len(triples))
	}
}

func TestClearGraph(t *testing.T) {
	s := rdfstore.NewStore()

	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")
	if err := s.ClearGraph(""); err != nil {
		t.Fatal(err)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("expected empty graph after clear, got %d triples", len(triples))
	}
}

func TestStats(t *testing.T) {
	s := rdfstore.NewStore()

	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")

	stats := s.Stats("")
	if stats["triples"] != 1 {
		t.Errorf("expected 1 triple in stats, got %d", stats["triples"])
	}
}

func TestDirtyTracking(t *testing.T) {
	s := rdfstore.NewStore()

	if s.IsDirty() {
		t.Error("new store should not be dirty")
	}

	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")
	if !s.IsDirty() {
		t.Error("store should be dirty after mutation")
	}

	s.ClearDirty()
	if s.IsDirty() {
		t.Error("store should not be dirty after ClearDirty")
	}
}

func TestBindAndListPrefixes(t *testing.T) {
	s := rdfstore.NewStore()

	s.BindPrefix("ex", "http://example.org/")
	prefixes := s.ListPrefixes()

	if prefixes["ex"] != "http://example.org/" {
		t.Errorf("unexpected prefix ex: %q", prefixes["ex"])
	}
}

func TestRemovePrefix(t *testing.T) {
	s := rdfstore.NewStore()

	s.BindPrefix("ex", "http://example.org/")
	s.RemovePrefix("ex")

	prefixes := s.ListPrefixes()
	if _, ok := prefixes["ex"]; ok {
		t.Error("prefix ex should have been removed")
	}
}

func TestLiteralWithLang(t *testing.T) {
	s := rdfstore.NewStore()

	err := s.AddTriple("", "http://a.org/s", "http://a.org/label", "Hello", "literal", "en", "")
	if err != nil {
		t.Fatalf("AddTriple with lang: %v", err)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Fatalf("expected 1 triple, got %d", len(triples))
	}
	if triples[0].Lang != "en" {
		t.Errorf("expected lang en, got %q", triples[0].Lang)
	}
}

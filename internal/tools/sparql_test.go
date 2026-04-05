package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

func TestSPARQLSelect(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")
	_ = s.AddTriple("", "http://example.org/Bob", "http://example.org/name", "Bob", "literal", "", "")

	result := callTool(t, s, "sparql", map[string]any{
		"query": `SELECT ?s ?name WHERE { ?s <http://example.org/name> ?name }`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatalf("expected JSON: %q\nerr: %v", raw, err)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestSPARQLAsk(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")

	result := callTool(t, s, "sparql", map[string]any{
		"query": `ASK { <http://example.org/Alice> <http://example.org/name> ?name }`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	if raw != "true" {
		t.Errorf("expected true, got %q", raw)
	}
}

func TestSPARQLUpdate_Insert(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "sparql", map[string]any{
		"query": `INSERT DATA { <http://example.org/Alice> <http://example.org/name> "Alice" }`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple after INSERT DATA, got %d", len(triples))
	}
}

func TestSPARQLInvalidQuery(t *testing.T) {
	s := rdfstore.NewStore()
	result := callTool(t, s, "sparql", map[string]any{
		"query": `THIS IS NOT SPARQL`,
	})
	if !result.IsError {
		t.Error("expected error for invalid SPARQL")
	}
}

func TestSPARQLMissingQuery(t *testing.T) {
	s := rdfstore.NewStore()
	result := callTool(t, s, "sparql", map[string]any{})
	if !result.IsError {
		t.Error("expected error for missing query parameter")
	}
}

func TestSPARQLNamedGraph(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.CreateGraph("http://example.org/g1")
	_ = s.AddTriple("http://example.org/g1", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")

	result := callTool(t, s, "sparql", map[string]any{
		"query": `SELECT ?name WHERE { <http://example.org/Alice> <http://example.org/name> ?name }`,
		"graph": "http://example.org/g1",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row, got %d", len(rows))
	}
}

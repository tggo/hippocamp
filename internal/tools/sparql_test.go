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

func TestSPARQL_PrefixInQuery(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")

	result := callTool(t, s, "sparql", map[string]any{
		"query": `PREFIX ex: <http://example.org/>
SELECT ?name WHERE { ex:Alice ex:name ?name }`,
	})
	if result.IsError {
		t.Fatalf("PREFIX in query should work: %v", tools.ResultText(result))
	}
	raw := tools.ResultText(result)
	var rows []map[string]any
	json.Unmarshal([]byte(raw), &rows)
	if len(rows) != 1 {
		t.Errorf("expected 1 row with PREFIX, got %d", len(rows))
	}
}

func TestSPARQL_InsertWhere(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")

	// INSERT...WHERE should be detected as update and work.
	result := callTool(t, s, "sparql", map[string]any{
		"query": `INSERT { <http://example.org/Alice> <http://example.org/hasData> "yes" }
WHERE { <http://example.org/Alice> <http://example.org/name> ?name }`,
	})
	if result.IsError {
		t.Fatalf("INSERT...WHERE failed: %s", tools.ResultText(result))
	}

	triples, _ := s.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) != 2 {
		t.Errorf("expected 2 triples after INSERT...WHERE, got %d", len(triples))
	}
}

func TestSPARQL_PrefixedUpdate(t *testing.T) {
	s := rdfstore.NewStore()

	// PREFIX + INSERT DATA
	result := callTool(t, s, "sparql", map[string]any{
		"query": `PREFIX ex: <http://example.org/>
INSERT DATA { ex:Alice ex:name "Alice" }`,
	})
	if result.IsError {
		t.Fatalf("PREFIX + INSERT DATA failed: %s", tools.ResultText(result))
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple, got %d", len(triples))
	}
}

func TestSPARQL_DeleteInsertWhere(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/type", "http://example.org/OldType", "uri", "", "")

	// DELETE...INSERT...WHERE pattern
	result := callTool(t, s, "sparql", map[string]any{
		"query": `DELETE { <http://example.org/Alice> <http://example.org/type> <http://example.org/OldType> }
INSERT { <http://example.org/Alice> <http://example.org/type> <http://example.org/NewType> }
WHERE { <http://example.org/Alice> <http://example.org/type> <http://example.org/OldType> }`,
	})
	if result.IsError {
		t.Fatalf("DELETE...INSERT...WHERE failed: %s", tools.ResultText(result))
	}

	triples, _ := s.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple after swap, got %d", len(triples))
	}
	if triples[0].Object != "http://example.org/NewType" {
		t.Errorf("expected NewType, got %s", triples[0].Object)
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

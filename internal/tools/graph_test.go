package tools_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

func TestGraphList(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.CreateGraph("http://example.org/g1")

	result := callTool(t, s, "graph", map[string]any{
		"action": "list",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var graphs []string
	if err := json.Unmarshal([]byte(raw), &graphs); err != nil {
		t.Fatalf("expected JSON array of strings: %v", err)
	}
	found := false
	for _, g := range graphs {
		if g == "http://example.org/g1" {
			found = true
		}
	}
	if !found {
		t.Errorf("g1 not in list: %v", graphs)
	}
}

func TestGraphCreate(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "graph", map[string]any{
		"action": "create",
		"name":   "http://example.org/new",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	graphs := s.ListGraphs()
	found := false
	for _, g := range graphs {
		if g == "http://example.org/new" {
			found = true
		}
	}
	if !found {
		t.Error("created graph not found")
	}
}

func TestGraphDelete(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.CreateGraph("http://example.org/del")

	result := callTool(t, s, "graph", map[string]any{
		"action": "delete",
		"name":   "http://example.org/del",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	for _, g := range s.ListGraphs() {
		if g == "http://example.org/del" {
			t.Error("deleted graph still present")
		}
	}
}

func TestGraphStats(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")

	result := callTool(t, s, "graph", map[string]any{
		"action": "stats",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var stats map[string]any
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if stats["triples"].(float64) != 1 {
		t.Errorf("expected 1 triple in stats, got %v", stats["triples"])
	}
}

func TestGraphClear(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://a.org/s", "http://a.org/p", "http://a.org/o", "uri", "", "")

	result := callTool(t, s, "graph", map[string]any{
		"action": "clear",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("expected empty after clear, got %d", len(triples))
	}
}

func TestGraphDumpLoad(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "test.trig")

	result := callTool(t, s, "graph", map[string]any{
		"action": "dump",
		"file":   path,
	})
	if result.IsError {
		t.Fatalf("dump error: %v", result.Content)
	}

	// clear and load back
	_ = s.ClearGraph("")
	result = callTool(t, s, "graph", map[string]any{
		"action": "load",
		"file":   path,
	})
	if result.IsError {
		t.Fatalf("load error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple after load, got %d", len(triples))
	}
}

func TestGraphPrefixAdd(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "graph", map[string]any{
		"action": "prefix_add",
		"prefix": "ex",
		"uri":    "http://example.org/",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	prefixes := s.ListPrefixes()
	if prefixes["ex"] != "http://example.org/" {
		t.Errorf("prefix not registered: %v", prefixes)
	}
}

func TestGraphPrefixList(t *testing.T) {
	s := rdfstore.NewStore()
	s.BindPrefix("ex", "http://example.org/")

	result := callTool(t, s, "graph", map[string]any{
		"action": "prefix_list",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var prefixes map[string]string
	if err := json.Unmarshal([]byte(raw), &prefixes); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if prefixes["ex"] != "http://example.org/" {
		t.Errorf("unexpected prefixes: %v", prefixes)
	}
}

func TestGraphInvalidAction(t *testing.T) {
	s := rdfstore.NewStore()
	result := callTool(t, s, "graph", map[string]any{
		"action": "unknown_action",
	})
	if !result.IsError {
		t.Error("expected error for unknown action")
	}
}

func TestGraphImport_Basic(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	trig := `@prefix ex: <http://example.org/> .
	ex:Alice ex:name "Alice" .
	ex:Alice ex:knows ex:Bob .
	ex:Bob ex:name "Bob" .`

	result := callTool(t, s, "graph", map[string]any{"action": "import", "data": trig})
	if result.IsError {
		t.Fatalf("import failed: %v", tools.ResultText(result))
	}

	// Verify triples exist for Alice
	triples, _ := s.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) != 2 {
		t.Errorf("expected 2 triples for Alice, got %d", len(triples))
	}
}

func TestGraphImport_NamedGraph(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	trig := `@prefix ex: <http://example.org/> .
	GRAPH <http://example.org/g1> {
		ex:Alice ex:name "Alice" .
		ex:Alice ex:knows ex:Bob .
	}`

	result := callTool(t, s, "graph", map[string]any{"action": "import", "data": trig})
	if result.IsError {
		t.Fatalf("import failed: %v", tools.ResultText(result))
	}

	// Verify triples are in the named graph
	triples, _ := s.ListTriples("http://example.org/g1", "", "", "")
	if len(triples) != 2 {
		t.Errorf("expected 2 triples in named graph g1, got %d", len(triples))
	}

	// Default graph should not have these triples
	defaultTriples, _ := s.ListTriples("", "http://example.org/Alice", "", "")
	if len(defaultTriples) != 0 {
		t.Errorf("expected 0 triples in default graph for Alice, got %d", len(defaultTriples))
	}
}

func TestGraphImport_Malformed(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	// Seed some valid data first
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")
	triples, _ := s.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) != 1 {
		t.Fatalf("seed failed: expected 1 triple, got %d", len(triples))
	}

	// Try importing garbage TriG
	result := callTool(t, s, "graph", map[string]any{"action": "import", "data": "this is not valid TriG {{{"})
	if !result.IsError {
		t.Error("expected error for malformed TriG")
	}

	// Verify existing data is still intact
	triples, _ = s.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) == 0 {
		t.Error("existing data was corrupted by bad import")
	}
}

func TestGraphImport_Empty(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	result := callTool(t, s, "graph", map[string]any{"action": "import", "data": ""})
	if !result.IsError {
		t.Error("expected error for empty data string")
	}
}

func TestGraphImport_Large(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	// Generate 500 triples in TriG format
	var b strings.Builder
	b.WriteString("@prefix ex: <http://example.org/> .\n")
	for i := 0; i < 500; i++ {
		b.WriteString(fmt.Sprintf("<http://example.org/entity/%d> ex:label \"Entity %d\" .\n", i, i))
	}

	result := callTool(t, s, "graph", map[string]any{"action": "import", "data": b.String()})
	if result.IsError {
		t.Fatalf("import failed: %v", tools.ResultText(result))
	}

	// Verify all 500 are present
	stats := s.Stats("")
	if stats["triples"] < 500 {
		t.Errorf("expected >= 500 triples, got %d", stats["triples"])
	}

	// Spot-check a specific entity
	triples, _ := s.ListTriples("", "http://example.org/entity/42", "", "")
	if len(triples) == 0 {
		t.Error("entity/42 not found after large import")
	}

	triples, _ = s.ListTriples("", "http://example.org/entity/499", "", "")
	if len(triples) == 0 {
		t.Error("entity/499 not found after large import")
	}
}

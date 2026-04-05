package tools_test

import (
	"encoding/json"
	"path/filepath"
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

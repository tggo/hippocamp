package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

// callTool is a test helper that invokes a registered tool handler.
func callTool(t *testing.T, s *rdfstore.Store, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	handler := tools.HandlerFor(s, name)
	if handler == nil {
		t.Fatalf("no handler registered for tool %q", name)
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool %q returned error: %v", name, err)
	}
	return result
}

func TestTripleAdd(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alice",
		"object_type": "literal",
	})
	if result.IsError {
		t.Fatalf("expected success, got error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple, got %d", len(triples))
	}
}

func TestTripleRemove(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "http://example.org/Bob", "uri", "", "")

	result := callTool(t, s, "triple", map[string]any{
		"action":    "remove",
		"subject":   "http://example.org/Alice",
		"predicate": "http://example.org/name",
		"object":    "http://example.org/Bob",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("expected 0 triples after remove, got %d", len(triples))
	}
}

func TestTripleList(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/name", "Alice", "literal", "", "")
	_ = s.AddTriple("", "http://example.org/Alice", "http://example.org/age", "30", "literal", "", "")

	result := callTool(t, s, "triple", map[string]any{
		"action":  "list",
		"subject": "http://example.org/Alice",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	// Result should be JSON array
	raw := tools.ResultText(result)
	var rows []map[string]any
	if err := json.Unmarshal([]byte(raw), &rows); err != nil {
		t.Fatalf("expected JSON array, got: %q", raw)
	}
	if len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d", len(rows))
	}
}

func TestTripleAdd_NamedGraph(t *testing.T) {
	s := rdfstore.NewStore()
	_ = s.CreateGraph("http://example.org/g1")

	result := callTool(t, s, "triple", map[string]any{
		"action":    "add",
		"graph":     "http://example.org/g1",
		"subject":   "http://example.org/s",
		"predicate": "http://example.org/p",
		"object":    "http://example.org/o",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	triples, _ := s.ListTriples("http://example.org/g1", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple in g1, got %d", len(triples))
	}
	// default graph should be empty
	triples, _ = s.ListTriples("", "", "", "")
	if len(triples) != 0 {
		t.Errorf("default graph should be empty, got %d triples", len(triples))
	}
}

func TestTripleAdd_MissingRequired(t *testing.T) {
	s := rdfstore.NewStore()
	result := callTool(t, s, "triple", map[string]any{
		"action": "add",
		// missing subject/predicate/object
	})
	if !result.IsError {
		t.Error("expected error for missing required fields")
	}
}

func TestTripleAdd_LiteralWithLang(t *testing.T) {
	s := rdfstore.NewStore()
	result := callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alice",
		"object_type": "literal",
		"lang":        "en",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 || triples[0].Lang != "en" {
		t.Errorf("expected literal with lang=en, got %+v", triples)
	}
}

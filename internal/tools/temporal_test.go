package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

func TestTripleInvalidate(t *testing.T) {
	s := rdfstore.NewStore()

	// Add a fact about Alice working at OldCo.
	callTool(t, s, "triple", map[string]any{
		"action":    "add",
		"subject":   "http://example.org/Alice",
		"predicate": "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
		"object":    "https://hippocamp.dev/ontology#Entity",
	})
	callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://www.w3.org/2000/01/rdf-schema#label",
		"object":      "Alice",
		"object_type": "literal",
	})
	callTool(t, s, "triple", map[string]any{
		"action":    "add",
		"subject":   "http://example.org/Alice",
		"predicate": "http://example.org/worksAt",
		"object":    "http://example.org/OldCo",
	})

	// Invalidate Alice.
	result := callTool(t, s, "triple", map[string]any{
		"action":  "invalidate",
		"subject": "http://example.org/Alice",
	})

	raw := tools.ResultText(result)
	var resp map[string]string
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("expected JSON response: %v (%s)", err, raw)
	}
	if resp["status"] != "invalidated" {
		t.Errorf("expected status=invalidated, got %q", resp["status"])
	}
	if resp["validTo"] == "" {
		t.Error("expected validTo timestamp")
	}

	// Verify hippo:validTo triple was added.
	triples, _ := s.ListTriples("", "http://example.org/Alice", "https://hippocamp.dev/ontology#validTo", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 validTo triple, got %d", len(triples))
	}

	// Verify hippo:status was set.
	triples, _ = s.ListTriples("", "http://example.org/Alice", "https://hippocamp.dev/ontology#status", "")
	if len(triples) != 1 || triples[0].Object != "invalidated" {
		t.Errorf("expected status=invalidated triple, got %+v", triples)
	}

	// Original worksAt triple should still exist (historical).
	triples, _ = s.ListTriples("", "http://example.org/Alice", "http://example.org/worksAt", "")
	if len(triples) != 1 {
		t.Error("original triple should be preserved for history")
	}
}

func TestTripleInvalidate_NotFound(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "triple", map[string]any{
		"action":  "invalidate",
		"subject": "http://example.org/NonExistent",
	})

	if !result.IsError {
		t.Error("expected error for non-existent subject")
	}
}

func TestTripleInvalidate_MissingSubject(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "triple", map[string]any{
		"action": "invalidate",
	})

	if !result.IsError {
		t.Error("expected error for missing subject")
	}
}

func TestTripleAdd_DuplicateDetection(t *testing.T) {
	s := rdfstore.NewStore()

	// Add a triple.
	result := callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alice",
		"object_type": "literal",
	})
	if tools.ResultText(result) != "ok" {
		t.Fatalf("first add should succeed, got: %s", tools.ResultText(result))
	}

	// Try to add the same triple again.
	result = callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alice",
		"object_type": "literal",
	})
	raw := tools.ResultText(result)
	if !strings.Contains(raw, "duplicate") {
		t.Errorf("expected duplicate warning, got: %s", raw)
	}

	// Verify only 1 triple exists.
	triples, _ := s.ListTriples("", "", "", "")
	if len(triples) != 1 {
		t.Errorf("expected 1 triple (no duplicate), got %d", len(triples))
	}
}

func TestTripleAdd_DifferentObjectNotDuplicate(t *testing.T) {
	s := rdfstore.NewStore()

	callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alice",
		"object_type": "literal",
	})
	result := callTool(t, s, "triple", map[string]any{
		"action":      "add",
		"subject":     "http://example.org/Alice",
		"predicate":   "http://example.org/name",
		"object":      "Alicia",
		"object_type": "literal",
	})
	if tools.ResultText(result) != "ok" {
		t.Errorf("different object should not be a duplicate, got: %s", tools.ResultText(result))
	}
}

func TestGraphSummary(t *testing.T) {
	s := rdfstore.NewStore()

	// Seed some data.
	addTriple := func(subj, pred, obj, objType string) {
		_ = s.AddTriple("", subj, pred, obj, objType, "", "")
	}

	hippo := "https://hippocamp.dev/ontology#"
	rdfT := "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfsL := "http://www.w3.org/2000/01/rdf-schema#label"

	// Topics.
	addTriple("http://ex.org/topic/auth", rdfT, hippo+"Topic", "uri")
	addTriple("http://ex.org/topic/auth", rdfsL, "Authentication", "literal")
	addTriple("http://ex.org/topic/db", rdfT, hippo+"Topic", "uri")
	addTriple("http://ex.org/topic/db", rdfsL, "Database", "literal")

	// Entities.
	addTriple("http://ex.org/alice", rdfT, hippo+"Entity", "uri")
	addTriple("http://ex.org/alice", rdfsL, "Alice", "literal")
	addTriple("http://ex.org/alice", hippo+"hasTopic", "http://ex.org/topic/auth", "uri")

	// Decision.
	addTriple("http://ex.org/dec1", rdfT, hippo+"Decision", "uri")
	addTriple("http://ex.org/dec1", rdfsL, "Use JWT tokens", "literal")
	addTriple("http://ex.org/dec1", hippo+"rationale", "Stateless auth", "literal")

	result := callTool(t, s, "graph", map[string]any{
		"action": "summary",
	})
	if result.IsError {
		t.Fatalf("unexpected error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var summary map[string]any
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatalf("expected JSON, got: %s", raw)
	}

	// Check type counts.
	tc, ok := summary["type_counts"].(map[string]any)
	if !ok {
		t.Fatal("missing type_counts")
	}
	if tc["Topic"] != float64(2) {
		t.Errorf("expected 2 Topics, got %v", tc["Topic"])
	}
	if tc["Entity"] != float64(1) {
		t.Errorf("expected 1 Entity, got %v", tc["Entity"])
	}
	if tc["Decision"] != float64(1) {
		t.Errorf("expected 1 Decision, got %v", tc["Decision"])
	}

	// Check topics list.
	topics, ok := summary["topics"].([]any)
	if !ok || len(topics) != 2 {
		t.Errorf("expected 2 topics, got %v", summary["topics"])
	}

	// Check decisions.
	decs, ok := summary["decisions"].([]any)
	if !ok || len(decs) != 1 {
		t.Errorf("expected 1 decision, got %v", summary["decisions"])
	}

	// Check total_triples > 0.
	if summary["total_triples"].(float64) == 0 {
		t.Error("expected non-zero total_triples")
	}
}

func TestGraphSummary_Empty(t *testing.T) {
	s := rdfstore.NewStore()

	result := callTool(t, s, "graph", map[string]any{
		"action": "summary",
	})
	if result.IsError {
		t.Fatalf("summary on empty graph should not error: %v", result.Content)
	}

	raw := tools.ResultText(result)
	var summary map[string]any
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatalf("expected JSON: %v", err)
	}
	if summary["total_triples"].(float64) != 0 {
		t.Error("expected 0 total_triples for empty graph")
	}
}

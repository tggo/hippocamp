package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func seedSearchData(t *testing.T, store *rdfstore.Store) {
	t.Helper()
	triples := []struct {
		graph, subj, pred, obj, objType string
	}{
		// A function
		{"", "https://hippocamp.dev/project/test#AddTriple", rdfType, "https://hippocamp.dev/ontology#Function", "uri"},
		{"", "https://hippocamp.dev/project/test#AddTriple", rdfsLabel, "AddTriple", "literal"},
		{"", "https://hippocamp.dev/project/test#AddTriple", hippoSummary, "Adds a triple to the named graph", "literal"},
		{"", "https://hippocamp.dev/project/test#AddTriple", hippoSignature, "func (s *Store) AddTriple(graphName, subject, predicate, object, objectType, lang, datatype string) error", "literal"},
		// A file
		{"", "https://hippocamp.dev/project/test/store.go", rdfType, "https://hippocamp.dev/ontology#File", "uri"},
		{"", "https://hippocamp.dev/project/test/store.go", rdfsLabel, "store.go", "literal"},
		{"", "https://hippocamp.dev/project/test/store.go", hippoFilePath, "internal/rdfstore/store.go", "literal"},
		{"", "https://hippocamp.dev/project/test/store.go", hippoSummary, "RDF store wrapper with BadgerDB backend", "literal"},
		// A struct
		{"", "https://hippocamp.dev/project/test#Store", rdfType, "https://hippocamp.dev/ontology#Struct", "uri"},
		{"", "https://hippocamp.dev/project/test#Store", rdfsLabel, "Store", "literal"},
		{"", "https://hippocamp.dev/project/test#Store", hippoSummary, "Wraps a context-aware goRDFlib Dataset backed by BadgerDB", "literal"},
		// A concept
		{"", "https://hippocamp.dev/project/test#Persistence", rdfType, "https://hippocamp.dev/ontology#Concept", "uri"},
		{"", "https://hippocamp.dev/project/test#Persistence", rdfsLabel, "Persistence", "literal"},
		{"", "https://hippocamp.dev/project/test#Persistence", hippoSummary, "TriG-based persistence with auto-load and auto-save", "literal"},
	}

	for _, tr := range triples {
		if err := store.AddTriple(tr.graph, tr.subj, tr.pred, tr.obj, tr.objType, "", ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func callSearch(t *testing.T, handler handlerFunc, args map[string]any) []SearchResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)
	if res.IsError {
		t.Fatalf("search returned error: %s", text)
	}

	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		t.Fatalf("unmarshal: %v (text: %s)", err, text)
	}
	return results
}

func TestSearch_BasicKeyword(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{"query": "triple"})
	if len(results) == 0 {
		t.Fatal("expected results for 'triple'")
	}
	if results[0].URI != "https://hippocamp.dev/project/test#AddTriple" {
		t.Errorf("expected AddTriple as top result, got %s", results[0].URI)
	}
}

func TestSearch_TypeFilter(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{
		"query": "store",
		"type":  "https://hippocamp.dev/ontology#Struct",
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Label != "Store" {
		t.Errorf("expected label 'Store', got %q", results[0].Label)
	}
}

func TestSearch_CaseInsensitive(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{"query": "BADGERDB"})
	if len(results) == 0 {
		t.Fatal("expected case-insensitive match for 'BADGERDB'")
	}
}

func TestSearch_Limit(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{"query": "store triple persistence", "limit": 2.0})
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}

func TestSearch_NoResults(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{"query": "zzzznonexistentzzzz"})
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearch_EmptyQuery(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": ""}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := ResultText(res)
	if !res.IsError {
		t.Errorf("expected error for empty query, got: %s", text)
	}
}

func TestSearch_NamedGraphScope(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Add to specific named graph.
	store.CreateGraph("http://test.org/proj")
	store.AddTriple("http://test.org/proj", "http://test.org/Foo", rdfsLabel, "Foo", "literal", "", "")
	store.AddTriple("http://test.org/proj", "http://test.org/Foo", rdfType, "https://hippocamp.dev/ontology#Function", "uri", "", "")

	// Add to default graph — should NOT appear in scoped search.
	store.AddTriple("", "http://test.org/Bar", rdfsLabel, "Foo also", "literal", "", "")

	handler := HandlerFor(store, "search")

	results := callSearch(t, handler, map[string]any{
		"query": "foo",
		"scope": "http://test.org/proj",
	})
	if len(results) != 1 {
		t.Fatalf("expected 1 scoped result, got %d", len(results))
	}
	if results[0].URI != "http://test.org/Foo" {
		t.Errorf("expected Foo, got %s", results[0].URI)
	}
}

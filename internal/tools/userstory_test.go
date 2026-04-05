package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// buildTriG converts a slice of testTriple to a TriG string.
func buildTriG(triples []testTriple) string {
	var b strings.Builder
	b.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n")
	b.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	b.WriteString("@prefix hippo: <https://hippocamp.dev/ontology#> .\n\n")
	for _, t := range triples {
		b.WriteString("<" + t.Subject + "> ")
		b.WriteString("<" + t.Predicate + "> ")
		if t.ObjType == "literal" {
			escaped := strings.ReplaceAll(t.Object, `"`, `\"`)
			b.WriteString(`"` + escaped + `"`)
		} else {
			b.WriteString("<" + t.Object + ">")
		}
		b.WriteString(" .\n")
	}
	return b.String()
}

// importTriG imports TriG data into a store via the graph tool handler.
func importTriG(t *testing.T, store *rdfstore.Store, data string) {
	t.Helper()
	handler := HandlerFor(store, "graph")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "import", "data": data}
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}
	if res.IsError {
		t.Fatalf("import failed: %s", ResultText(res))
	}
	t.Logf("Import result: %s", ResultText(res))
}

func TestUserStory_FirstAnalysis(t *testing.T) {
	// US1: Empty graph -> import house project triples via TriG -> search finds entities
	store := rdfstore.NewStore()
	defer store.Close()

	proj := houseProject()
	trig := buildTriG(proj.Triples)

	// Import via graph tool (the real way)
	graphHandler := HandlerFor(store, "graph")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "import", "data": trig}
	res, err := graphHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}
	if res.IsError {
		t.Fatalf("import failed: %s", ResultText(res))
	}

	// Verify search works - find Jim Patterson (contractor)
	searchHandler := HandlerFor(store, "search")
	req = mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "Jim Patterson"}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if res.IsError {
		t.Fatalf("search failed: %s", ResultText(res))
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal search results: %v", err)
	}
	if len(results) == 0 {
		t.Error("search for 'Jim Patterson' returned 0 results after import")
	}

	// Search for "metal roof" -> must find decision
	req.Params.Arguments = map[string]any{"query": "metal roof"}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var roofResults []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &roofResults); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(roofResults) == 0 {
		t.Error("search for 'metal roof' returned 0 results after import")
	}

	// Validate the graph
	validateHandler := HandlerFor(store, "validate")
	req = mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{}
	res, err = validateHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	if res.IsError {
		t.Fatalf("validate failed: %s", ResultText(res))
	}
	var valOut map[string]any
	if err := json.Unmarshal([]byte(ResultText(res)), &valOut); err != nil {
		t.Fatalf("unmarshal validate: %v", err)
	}
	if valOut["valid"] != true {
		t.Errorf("expected valid=true, got %v; warnings: %v", valOut["valid"], valOut["warnings"])
	}
}

func TestUserStory_SearchAfterImport(t *testing.T) {
	// US2: Import garden project via TriG -> test all query types
	store := rdfstore.NewStore()
	defer store.Close()

	proj := gardenProject()
	trig := buildTriG(proj.Triples)
	importTriG(t, store, trig)

	searchHandler := HandlerFor(store, "search")

	// Keyword: "tomato"
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "tomato"}
	res, err := searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) == 0 {
		t.Error("search for 'tomato' returned 0 results")
	}

	// Type filter: query="soil", type=hippo:Note
	req.Params.Arguments = map[string]any{
		"query": "soil",
		"type":  "https://hippocamp.dev/ontology#Note",
	}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var filteredResults []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &filteredResults); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(filteredResults) == 0 {
		t.Error("search for 'soil' with type Note returned 0 results")
	}
	for _, r := range filteredResults {
		if r.Type != "https://hippocamp.dev/ontology#Note" {
			t.Errorf("expected type Note, got %s", r.Type)
		}
	}

	// Related: query="varieties" with related=true
	req.Params.Arguments = map[string]any{"query": "varieties", "related": true}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var relatedResults []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &relatedResults); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(relatedResults) == 0 {
		t.Error("search for 'varieties' with related=true returned 0 results")
	}

	// Negative: "electrical" -> 0 results (not in garden project)
	req.Params.Arguments = map[string]any{"query": "electrical"}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var negResults []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &negResults); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(negResults) != 0 {
		t.Errorf("search for 'electrical' in garden project should return 0 results, got %d", len(negResults))
	}
}

func TestUserStory_PersistenceRoundTrip(t *testing.T) {
	// US3: Import -> dump to file -> create new store -> load -> search still works
	store1 := rdfstore.NewStore()
	defer store1.Close()

	proj := salesProject()
	trig := buildTriG(proj.Triples)
	importTriG(t, store1, trig)

	// Dump to file
	tmpFile := filepath.Join(t.TempDir(), "test.trig")
	graphHandler := HandlerFor(store1, "graph")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "dump", "file": tmpFile}
	res, err := graphHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("dump error: %v", err)
	}
	if res.IsError {
		t.Fatalf("dump failed: %s", ResultText(res))
	}

	// New store, load from file
	store2 := rdfstore.NewStore()
	defer store2.Close()
	graphHandler2 := HandlerFor(store2, "graph")
	req = mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "load", "file": tmpFile}
	res, err = graphHandler2(context.Background(), req)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}
	if res.IsError {
		t.Fatalf("load failed: %s", ResultText(res))
	}

	// Search in store2 -> must find "Acme Corp"
	searchHandler := HandlerFor(store2, "search")
	req = mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "Acme Corp"}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) == 0 {
		t.Error("search for 'Acme Corp' in loaded store returned 0 results")
	}
}

func TestUserStory_MalformedImport(t *testing.T) {
	// US4: Existing data + bad import -> error, existing data intact
	store := rdfstore.NewStore()
	defer store.Close()

	// Add good data first
	good := `@prefix ex: <http://example.org/> . ex:Alice ex:name "Alice" .`
	importTriG(t, store, good)

	// Try bad import
	graphHandler := HandlerFor(store, "graph")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"action": "import", "data": "this is not valid TriG {{{"}
	res, _ := graphHandler(context.Background(), req)
	if !res.IsError {
		t.Error("expected error for malformed TriG")
	}

	// Verify Alice still exists
	triples, _ := store.ListTriples("", "http://example.org/Alice", "", "")
	if len(triples) == 0 {
		t.Error("existing data corrupted by bad import")
	}
}

func TestUserStory_LargeImport(t *testing.T) {
	// US5: 500+ triples in one import
	store := rdfstore.NewStore()
	defer store.Close()

	var b strings.Builder
	b.WriteString("@prefix ex: <http://example.org/> .\n")
	b.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	b.WriteString("@prefix hippo: <https://hippocamp.dev/ontology#> .\n\n")
	for i := 0; i < 500; i++ {
		uri := fmt.Sprintf("http://example.org/entity/%d", i)
		b.WriteString(fmt.Sprintf("<%s> a hippo:Entity ; rdfs:label \"Entity %d\" ; hippo:summary \"Summary for entity number %d\" .\n", uri, i, i))
	}

	importTriG(t, store, b.String())

	// Verify count
	stats := store.Stats("")
	if stats["triples"] < 1500 { // 3 triples per entity * 500
		t.Errorf("expected >= 1500 triples, got %d", stats["triples"])
	}

	// Search works
	searchHandler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "Entity 42"}
	res, err := searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results) == 0 {
		t.Error("search for 'Entity 42' returned 0 results after large import")
	}

	// Search for Entity 499
	req.Params.Arguments = map[string]any{"query": "Entity 499"}
	res, err = searchHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	var results499 []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results499); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(results499) == 0 {
		t.Error("search for 'Entity 499' returned 0 results after large import")
	}
}

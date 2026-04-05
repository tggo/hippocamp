package analytics

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestCollectorRecordsTriples(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	// Simulate a search tool call.
	reqID := "req-1"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "search"
	req.Params.Arguments = map[string]any{"query": "authentication"}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: `[{"subject":"http://ex.org/Auth","label":"Auth Service","score":4.0}]`},
		},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	// Check that triples were written to the analytics graph.
	triples, err := store.ListTriples(GraphURI, "", "", "")
	if err != nil {
		t.Fatalf("ListTriples: %v", err)
	}
	if len(triples) == 0 {
		t.Fatal("expected analytics triples, got none")
	}

	// Verify key predicates exist.
	preds := make(map[string]string)
	for _, tr := range triples {
		preds[tr.Predicate] = tr.Object
	}

	if preds[ns+"tool"] != "search" {
		t.Errorf("expected tool=search, got %q", preds[ns+"tool"])
	}
	if preds[ns+"input"] != "authentication" {
		t.Errorf("expected input=authentication, got %q", preds[ns+"input"])
	}
	if preds[ns+"resultCount"] != "1" {
		t.Errorf("expected resultCount=1, got %q", preds[ns+"resultCount"])
	}
	if _, ok := preds[ns+"durationMs"]; !ok {
		t.Error("expected durationMs predicate")
	}
	if _, ok := preds[ns+"timestamp"]; !ok {
		t.Error("expected timestamp predicate")
	}
}

func TestCollectorRecordsSparqlCall(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-2"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "sparql"
	req.Params.Arguments = map[string]any{"query": "SELECT ?s WHERE { ?s a <http://ex.org/Person> }"}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: `[{"s":"http://ex.org/Alice"},{"s":"http://ex.org/Bob"}]`},
		},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	triples, err := store.ListTriples(GraphURI, "", ns+"resultCount", "")
	if err != nil {
		t.Fatalf("ListTriples: %v", err)
	}
	found := false
	for _, tr := range triples {
		if tr.Object == "2" {
			found = true
		}
	}
	if !found {
		t.Error("expected resultCount=2 for SPARQL query with 2 rows")
	}
}

func TestCollectorRecordsErrorCall(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-3"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "sparql"
	req.Params.Arguments = map[string]any{"query": "INVALID SPARQL"}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := mcp.NewToolResultError("SPARQL parse error: unexpected token")
	c.AfterCallTool(context.Background(), reqID, req, result)

	triples, err := store.ListTriples(GraphURI, "", ns+"error", "")
	if err != nil {
		t.Fatalf("ListTriples: %v", err)
	}
	if len(triples) == 0 {
		t.Error("expected error triple for failed SPARQL call")
	}
}

func TestCollectorRecordsTripleAddCall(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-4"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "triple"
	req.Params.Arguments = map[string]any{
		"action":    "add",
		"subject":   "http://ex.org/Alice",
		"predicate": "http://ex.org/name",
		"object":    "Alice",
	}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: "ok"},
		},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	preds := make(map[string]string)
	triples, _ := store.ListTriples(GraphURI, "", "", "")
	for _, tr := range triples {
		preds[tr.Predicate] = tr.Object
	}

	if preds[ns+"tool"] != "triple" {
		t.Errorf("expected tool=triple, got %q", preds[ns+"tool"])
	}
	if preds[ns+"input"] != "add" {
		t.Errorf("expected input=add, got %q", preds[ns+"input"])
	}
}

func TestCollectorSequentialIDs(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	// Two calls should produce distinct subjects.
	for i := 0; i < 2; i++ {
		reqID := i
		req := &mcp.CallToolRequest{}
		req.Params.Name = "search"
		req.Params.Arguments = map[string]any{"query": "test"}
		c.BeforeCallTool(context.Background(), reqID, req)
		result := &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Text: "[]"}},
		}
		c.AfterCallTool(context.Background(), reqID, req, result)
	}

	triples, _ := store.ListTriples(GraphURI, "", ns+"tool", "")
	if len(triples) != 2 {
		t.Errorf("expected 2 tool triples, got %d", len(triples))
	}
	if triples[0].Subject == triples[1].Subject {
		t.Error("expected distinct subjects for different calls")
	}
}

func TestIsAnalyticsQueryByGraph(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-graph"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "triple"
	req.Params.Arguments = map[string]any{
		"action": "list",
		"graph":  GraphURI,
	}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Text: "[]"}},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	// Should NOT have recorded any triples because the call targets the analytics graph.
	triples, _ := store.ListTriples(GraphURI, "", ns+"tool", "")
	if len(triples) != 0 {
		t.Errorf("expected 0 tool triples for analytics-graph call, got %d", len(triples))
	}
}

func TestIsAnalyticsQueryByScope(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-scope"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "search"
	req.Params.Arguments = map[string]any{
		"query": "something",
		"scope": GraphURI,
	}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Text: "[]"}},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	triples, _ := store.ListTriples(GraphURI, "", ns+"tool", "")
	if len(triples) != 0 {
		t.Errorf("expected 0 tool triples for scope=analytics call, got %d", len(triples))
	}
}

func TestIsAnalyticsQueryBySparql(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-sparql"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "sparql"
	req.Params.Arguments = map[string]any{
		"query": "SELECT ?s WHERE { GRAPH <" + GraphURI + "> { ?s ?p ?o } }",
	}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Text: "[]"}},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	triples, _ := store.ListTriples(GraphURI, "", ns+"tool", "")
	if len(triples) != 0 {
		t.Errorf("expected 0 tool triples for SPARQL targeting analytics graph, got %d", len(triples))
	}
}

func TestCollectorRecordsGraphAction(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-graph-create"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "graph"
	req.Params.Arguments = map[string]any{
		"action": "create",
		"graph":  "http://example.org/my-graph",
	}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Text: "ok"}},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	preds := make(map[string]string)
	triples, _ := store.ListTriples(GraphURI, "", "", "")
	for _, tr := range triples {
		preds[tr.Predicate] = tr.Object
	}

	if preds[ns+"tool"] != "graph" {
		t.Errorf("expected tool=graph, got %q", preds[ns+"tool"])
	}
	if preds[ns+"input"] != "create" {
		t.Errorf("expected input=create, got %q", preds[ns+"input"])
	}
	if preds[ns+"graph"] != "http://example.org/my-graph" {
		t.Errorf("expected graph=http://example.org/my-graph, got %q", preds[ns+"graph"])
	}
}

func TestCollectorRecordsValidateCall(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store)

	reqID := "req-validate"
	req := &mcp.CallToolRequest{}
	req.Params.Name = "validate"
	req.Params.Arguments = map[string]any{}

	c.BeforeCallTool(context.Background(), reqID, req)

	result := &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Text: `{"valid":true,"errors":[],"warnings":[],"stats":{"resources":0}}`},
		},
	}
	c.AfterCallTool(context.Background(), reqID, req, result)

	preds := make(map[string]string)
	triples, _ := store.ListTriples(GraphURI, "", "", "")
	for _, tr := range triples {
		preds[tr.Predicate] = tr.Object
	}

	if preds[ns+"tool"] != "validate" {
		t.Errorf("expected tool=validate, got %q", preds[ns+"tool"])
	}
	if preds[ns+"input"] != "validate" {
		t.Errorf("expected input=validate, got %q", preds[ns+"input"])
	}
}

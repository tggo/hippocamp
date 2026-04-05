package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestStress_RapidTripleAdds(t *testing.T) {
	// 200 sequential triple add calls — tests what crashed production
	store := rdfstore.NewStore()
	defer store.Close()
	handler := HandlerFor(store, "triple")

	for i := 0; i < 200; i++ {
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{
			"action":      "add",
			"subject":     fmt.Sprintf("http://example.org/%d", i),
			"predicate":   "http://www.w3.org/2000/01/rdf-schema#label",
			"object":      fmt.Sprintf("Item %d", i),
			"object_type": "literal",
		}
		res, err := handler(context.Background(), req)
		if err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
		if res.IsError {
			t.Fatalf("call %d: %s", i, ResultText(res))
		}
	}

	stats := store.Stats("")
	if stats["triples"] != 200 {
		t.Errorf("expected 200, got %d", stats["triples"])
	}
}

func TestStress_LargeImport(t *testing.T) {
	// 1000 triples in one import
	store := rdfstore.NewStore()
	defer store.Close()

	var b strings.Builder
	b.WriteString("@prefix ex: <http://example.org/> .\n")
	for i := 0; i < 1000; i++ {
		b.WriteString(fmt.Sprintf("<http://example.org/e%d> ex:val \"%d\" .\n", i, i))
	}
	importTriG(t, store, b.String())

	stats := store.Stats("")
	if stats["triples"] != 1000 {
		t.Errorf("expected 1000, got %d", stats["triples"])
	}
}

func TestStress_ConcurrentSearches(t *testing.T) {
	// Import data, then run 50 concurrent searches
	store := rdfstore.NewStore()
	defer store.Close()

	proj := houseProject()
	trig := buildTriG(proj.Triples)
	importTriG(t, store, trig)

	handler := HandlerFor(store, "search")

	var wg sync.WaitGroup
	errors := make(chan string, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			queries := []string{"budget", "permit", "lumber", "electrical", "HVAC"}
			q := queries[n%len(queries)]
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": q}
			res, err := handler(context.Background(), req)
			if err != nil {
				errors <- fmt.Sprintf("search %d: %v", n, err)
				return
			}
			if res.IsError {
				errors <- fmt.Sprintf("search %d: %s", n, ResultText(res))
				return
			}
			// Verify we can unmarshal the results
			var results []SearchResult
			if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
				errors <- fmt.Sprintf("search %d unmarshal: %v", n, err)
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for e := range errors {
		t.Error(e)
	}
}

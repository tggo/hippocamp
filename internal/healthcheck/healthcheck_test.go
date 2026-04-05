package healthcheck

import (
	"testing"
	"time"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestDanglingRefs(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A references non-existent entity B.
	store.AddTriple("", "http://ex.org/A", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "Entity A", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", hippoNS+"references", "http://ex.org/B", "uri", "", "")

	c := New(store, time.Hour) // long interval, we'll check initial scan
	// Wait for initial scan.
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if r.Stats.DanglingRefs != 1 {
		t.Errorf("expected 1 dangling ref, got %d", r.Stats.DanglingRefs)
	}
	if len(r.DanglingRefs) != 1 {
		t.Fatalf("expected 1 dangling ref detail, got %d", len(r.DanglingRefs))
	}
	if r.DanglingRefs[0].Object != "http://ex.org/B" {
		t.Errorf("expected dangling ref to B, got %s", r.DanglingRefs[0].Object)
	}
}

func TestNoDanglingWhenTargetExists(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.AddTriple("", "http://ex.org/A", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "A", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", hippoNS+"references", "http://ex.org/B", "uri", "", "")
	store.AddTriple("", "http://ex.org/B", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/B", rdfsLabel, "B", "literal", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if r.Stats.DanglingRefs != 0 {
		t.Errorf("expected 0 dangling refs, got %d", r.Stats.DanglingRefs)
	}
}

func TestOrphanResources(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity with no relationships at all — orphan.
	store.AddTriple("", "http://ex.org/orphan", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/orphan", rdfsLabel, "Orphan", "literal", "", "")

	// Connected entity — not orphan.
	store.AddTriple("", "http://ex.org/topic", rdfType, hippoNS+"Topic", "uri", "", "")
	store.AddTriple("", "http://ex.org/topic", rdfsLabel, "Topic", "literal", "", "")
	store.AddTriple("", "http://ex.org/linked", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/linked", rdfsLabel, "Linked", "literal", "", "")
	store.AddTriple("", "http://ex.org/linked", hippoNS+"hasTopic", "http://ex.org/topic", "uri", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if r.Stats.Orphans != 1 {
		t.Errorf("expected 1 orphan, got %d", r.Stats.Orphans)
	}
}

func TestMarkDirtyTriggersScan(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	c := New(store, 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	r1 := c.Report()
	if r1 == nil {
		t.Fatal("expected initial report")
	}
	if r1.Stats.Resources != 0 {
		t.Errorf("expected 0 resources initially, got %d", r1.Stats.Resources)
	}

	// Add data and mark dirty.
	store.AddTriple("", "http://ex.org/X", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/X", rdfsLabel, "X", "literal", "", "")
	c.MarkDirty()

	// Wait for next tick.
	time.Sleep(150 * time.Millisecond)

	r2 := c.Report()
	if r2 == nil {
		t.Fatal("expected report after dirty")
	}
	if r2.Stats.Resources != 1 {
		t.Errorf("expected 1 resource after add, got %d", r2.Stats.Resources)
	}
}

func TestAnalyzeZeroResults(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.CreateGraph(analyticsGraph)

	// Call 1: search with 0 results.
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", rdfType, analyticsNS+"ToolCall", "uri", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"tool", "search", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"input", "nonexistent thing", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"resultCount", "0", "literal", "", "")

	// Call 2: search with results (should not appear).
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", rdfType, analyticsNS+"ToolCall", "uri", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"tool", "search", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"input", "found something", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"resultCount", "3", "literal", "", "")

	// Call 3: another search with 0 results.
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:3", rdfType, analyticsNS+"ToolCall", "uri", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:3", analyticsNS+"tool", "search", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:3", analyticsNS+"input", "another missing", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:3", analyticsNS+"resultCount", "0", "literal", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if len(r.MissingAliases) != 2 {
		t.Fatalf("expected 2 alias suggestions, got %d", len(r.MissingAliases))
	}
	queries := map[string]bool{}
	for _, a := range r.MissingAliases {
		queries[a.Query] = true
	}
	if !queries["nonexistent thing"] {
		t.Error("expected suggestion for 'nonexistent thing'")
	}
	if !queries["another missing"] {
		t.Error("expected suggestion for 'another missing'")
	}
	if queries["found something"] {
		t.Error("should not suggest for query with results")
	}
}

func TestAnalyzeZeroResultsDedup(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.CreateGraph(analyticsGraph)

	// Two separate calls with the same zero-result query.
	for i, subj := range []string{"urn:hippocamp:analytics:call:1", "urn:hippocamp:analytics:call:2"} {
		_ = i
		store.AddTriple(analyticsGraph, subj, rdfType, analyticsNS+"ToolCall", "uri", "", "")
		store.AddTriple(analyticsGraph, subj, analyticsNS+"tool", "search", "literal", "", "")
		store.AddTriple(analyticsGraph, subj, analyticsNS+"input", "duplicate query", "literal", "", "")
		store.AddTriple(analyticsGraph, subj, analyticsNS+"resultCount", "0", "literal", "", "")
	}

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if len(r.MissingAliases) != 1 {
		t.Fatalf("expected 1 deduplicated suggestion, got %d", len(r.MissingAliases))
	}
	if r.MissingAliases[0].Query != "duplicate query" {
		t.Errorf("expected query 'duplicate query', got %q", r.MissingAliases[0].Query)
	}
}

func TestAnalyzeZeroResultsIgnoresNonSearch(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.CreateGraph(analyticsGraph)

	// Zero-result call for "sparql" tool — should be ignored.
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", rdfType, analyticsNS+"ToolCall", "uri", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"tool", "sparql", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"input", "SELECT * WHERE { ?s ?p ?o }", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:1", analyticsNS+"resultCount", "0", "literal", "", "")

	// Zero-result call for "triple" tool — should be ignored.
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", rdfType, analyticsNS+"ToolCall", "uri", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"tool", "triple", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"input", "some input", "literal", "", "")
	store.AddTriple(analyticsGraph, "urn:hippocamp:analytics:call:2", analyticsNS+"resultCount", "0", "literal", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if len(r.MissingAliases) != 0 {
		t.Errorf("expected 0 alias suggestions for non-search tools, got %d", len(r.MissingAliases))
	}
}

func TestDanglingRefsWithHasTag(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A uses hasTag to reference a non-existent tag.
	store.AddTriple("", "http://ex.org/A", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "Entity A", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", hippoNS+"hasTag", "http://ex.org/missing-tag", "uri", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if r.Stats.DanglingRefs != 1 {
		t.Errorf("expected 1 dangling ref via hasTag, got %d", r.Stats.DanglingRefs)
	}
	if len(r.DanglingRefs) != 1 {
		t.Fatalf("expected 1 dangling ref detail, got %d", len(r.DanglingRefs))
	}
	if r.DanglingRefs[0].Predicate != hippoNS+"hasTag" {
		t.Errorf("expected predicate hasTag, got %s", r.DanglingRefs[0].Predicate)
	}
	if r.DanglingRefs[0].Object != "http://ex.org/missing-tag" {
		t.Errorf("expected dangling ref to missing-tag, got %s", r.DanglingRefs[0].Object)
	}
}

func TestNonUriObjectsSkipped(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A has a references triple where the object is a literal, not a URI.
	// This should NOT be flagged as a dangling ref.
	store.AddTriple("", "http://ex.org/A", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "Entity A", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", hippoNS+"references", "just a string", "literal", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	if r.Stats.DanglingRefs != 0 {
		t.Errorf("expected 0 dangling refs (literal object should be skipped), got %d", r.Stats.DanglingRefs)
	}
}

func TestSkipsAnalyticsGraph(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Add data in both analytics and default graph.
	store.CreateGraph(analyticsGraph)
	store.AddTriple(analyticsGraph, "http://analytics/call1", "http://purl.org/hippocamp/analytics#tool", "search", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "A", "literal", "", "")

	c := New(store, time.Hour)
	time.Sleep(100 * time.Millisecond)

	r := c.Report()
	if r == nil {
		t.Fatal("expected report")
	}
	// Should count only the real resource, not analytics triples.
	if r.Stats.Resources != 1 {
		t.Errorf("expected 1 resource (skipping analytics), got %d", r.Stats.Resources)
	}
}

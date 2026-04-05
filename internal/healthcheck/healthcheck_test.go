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

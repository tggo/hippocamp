package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// searchAndParse runs a search query and returns parsed results.
func searchAndParse(t *testing.T, store *rdfstore.Store, args map[string]any) []SearchResult {
	t.Helper()
	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)
	if res.IsError {
		t.Fatalf("search tool error: %s", text)
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return results
}

func TestSearch_PrefixMatch(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// "Building Permit" should be found by query "build" (prefix match).
	store.AddTriple("", "http://test.org/permit", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/permit", rdfsLabel, "Building Permit", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "build"})
	if len(results) == 0 {
		t.Fatal("expected prefix match: 'build' should match 'Building'")
	}
}

func TestSearch_PrefixMatch_Contractor(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.AddTriple("", "http://test.org/gc", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/gc", rdfsLabel, "General Contractor", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "contract"})
	if len(results) == 0 {
		t.Fatal("expected prefix match: 'contract' should match 'Contractor'")
	}
}

func TestSearch_WordBoundaryRanking(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// "rebuild" has "build" mid-word; "build plan" has "build" at word start.
	store.AddTriple("", "http://test.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/A", rdfsLabel, "rebuild project", "literal", "", "")

	store.AddTriple("", "http://test.org/B", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/B", rdfsLabel, "build plan", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "build"})
	if len(results) < 2 {
		t.Fatalf("expected both results, got %d", len(results))
	}
	// B should rank higher (word boundary match).
	if results[0].URI != "http://test.org/B" {
		t.Errorf("expected 'build plan' (word boundary) to rank first, got %s", results[0].URI)
	}
}

func TestSearch_FieldBoosting(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A has "budget" in label (high boost).
	store.AddTriple("", "http://test.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/A", rdfsLabel, "Budget Overview", "literal", "", "")

	// Entity B has "budget" only in content (low boost).
	store.AddTriple("", "http://test.org/B", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple("", "http://test.org/B", rdfsLabel, "Financial Notes", "literal", "", "")
	store.AddTriple("", "http://test.org/B", hippoContent, "This covers the budget for Q2", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "budget"})
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// A should rank higher (label match > content match).
	if results[0].URI != "http://test.org/A" {
		t.Errorf("expected label match to rank first, got %s", results[0].URI)
	}
}

func TestSearch_FollowHasTopic(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Topic "Budget" with label "Budget".
	store.AddTriple("", "http://test.org/topic/budget", rdfType, testHippoTopic, "uri", "", "")
	store.AddTriple("", "http://test.org/topic/budget", rdfsLabel, "Budget", "literal", "", "")

	// Decision "Metal Roof" linked to Budget topic via hasTopic.
	// "Metal Roof" does NOT contain "budget" in any searchable field.
	store.AddTriple("", "http://test.org/decision/metal-roof", rdfType, testHippoDecision, "uri", "", "")
	store.AddTriple("", "http://test.org/decision/metal-roof", rdfsLabel, "Metal Roof", "literal", "", "")
	store.AddTriple("", "http://test.org/decision/metal-roof", testHippoRationale, "Hail resistance and longevity", "literal", "", "")
	store.AddTriple("", "http://test.org/decision/metal-roof", testHippoHasTopic, "http://test.org/topic/budget", "uri", "", "")

	// Search "budget" with related=true — should find both topic AND decision.
	results := searchAndParse(t, store, map[string]any{"query": "budget", "related": true})

	foundTopic := false
	foundDecision := false
	for _, r := range results {
		if r.URI == "http://test.org/topic/budget" {
			foundTopic = true
		}
		if r.URI == "http://test.org/decision/metal-roof" {
			foundDecision = true
		}
	}
	if !foundTopic {
		t.Error("expected to find Budget topic (direct match)")
	}
	if !foundDecision {
		t.Error("expected to find Metal Roof decision (via hasTopic relationship)")
	}
}

func TestSearch_RelatedPartOf(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity "Ferguson Supply"
	store.AddTriple("", "http://test.org/entity/ferguson", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/entity/ferguson", rdfsLabel, "Ferguson Supply", "literal", "", "")

	// Note referencing Ferguson — the note has NO "ferguson" in any field.
	store.AddTriple("", "http://test.org/note/fixtures", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple("", "http://test.org/note/fixtures", rdfsLabel, "Plumbing Fixtures Order", "literal", "", "")
	store.AddTriple("", "http://test.org/note/fixtures", testHippoReferences, "http://test.org/entity/ferguson", "uri", "", "")

	// Without related: only Ferguson.
	results := searchAndParse(t, store, map[string]any{"query": "Ferguson"})
	if len(results) != 1 {
		t.Errorf("without related: expected 1 result, got %d", len(results))
	}

	// With related: Ferguson + the referencing note.
	results = searchAndParse(t, store, map[string]any{"query": "Ferguson", "related": true})
	if len(results) < 2 {
		t.Errorf("with related: expected 2 results, got %d", len(results))
	}
}

func TestSearch_RelatedDisabledByDefault(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.AddTriple("", "http://test.org/topic/x", rdfType, testHippoTopic, "uri", "", "")
	store.AddTriple("", "http://test.org/topic/x", rdfsLabel, "UniqueTopicXYZ", "literal", "", "")

	store.AddTriple("", "http://test.org/note/y", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple("", "http://test.org/note/y", rdfsLabel, "Some Note", "literal", "", "")
	store.AddTriple("", "http://test.org/note/y", testHippoHasTopic, "http://test.org/topic/x", "uri", "", "")

	// Without related flag: should NOT find the note.
	results := searchAndParse(t, store, map[string]any{"query": "UniqueTopicXYZ"})
	for _, r := range results {
		if r.URI == "http://test.org/note/y" {
			t.Error("without related=true, should NOT find related resources")
		}
	}
}

func TestSearch_AccumulateScores(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A has "metal" in label AND summary.
	store.AddTriple("", "http://test.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/A", rdfsLabel, "Metal Roof", "literal", "", "")
	store.AddTriple("", "http://test.org/A", hippoSummary, "Standing seam metal roofing", "literal", "", "")

	// Entity B has "metal" only in label.
	store.AddTriple("", "http://test.org/B", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/B", rdfsLabel, "Metal Detector", "literal", "", "")
	store.AddTriple("", "http://test.org/B", hippoSummary, "Equipment for site survey", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "metal"})
	if len(results) < 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// A should rank higher (metal in 2 fields vs 1).
	if results[0].URI != "http://test.org/A" {
		t.Errorf("expected multi-field match to rank first, got %s", results[0].URI)
	}
}

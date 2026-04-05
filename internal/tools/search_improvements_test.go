package tools

import (
	"context"
	"encoding/json"
	"strings"
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

	// Try array first (normal results).
	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err == nil {
		return results
	}

	// Try hint object (zero-result response).
	var hint struct {
		Results []SearchResult `json:"results"`
		Hint    string         `json:"hint"`
	}
	if err := json.Unmarshal([]byte(text), &hint); err != nil {
		t.Fatalf("unmarshal: %v (text: %s)", err, text)
	}
	return hint.Results
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

func TestSearch_PrefixMatch_StemVariation(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// "електропостачання" in label — keyword "електрика" is NOT a substring,
	// but they share the prefix "електр" (6 chars > minPrefixLen of 4).
	store.AddTriple("", "http://test.org/power", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/power", rdfsLabel, "електропостачання системи", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "електрика"})
	if len(results) == 0 {
		t.Fatal("expected prefix match: 'електрика' shares prefix 'електр' with 'електропостачання'")
	}
}

func TestSearch_PrefixMatch_TooShort(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// "abc" is only 3 chars — below minPrefixLen, should NOT trigger prefix match.
	store.AddTriple("", "http://test.org/x", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/x", rdfsLabel, "abcdefgh", "literal", "", "")

	results := searchAndParse(t, store, map[string]any{"query": "abcz"})
	if len(results) > 0 {
		// "abcz" shares only 3 chars with "abcdefgh" — too short for prefix match.
		// But "abcz" is 4 chars and shares prefix "abc" (3 chars) — below threshold.
		t.Fatal("prefix match should not trigger for shared prefix < 4 chars")
	}
}

func TestSearch_AliasMatch(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	testHippoAlias := testHippoNS + "alias"

	// Entity with English label but Ukrainian alias.
	store.AddTriple("", "http://test.org/electrical", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/electrical", rdfsLabel, "Electrical Wiring", "literal", "", "")
	store.AddTriple("", "http://test.org/electrical", testHippoAlias, "електрика", "literal", "", "")
	store.AddTriple("", "http://test.org/electrical", testHippoAlias, "проводка", "literal", "", "")

	// Search by alias keyword should find the resource.
	results := searchAndParse(t, store, map[string]any{"query": "електрика"})
	if len(results) == 0 {
		t.Fatal("expected to find resource via hippo:alias match")
	}
	if results[0].URI != "http://test.org/electrical" {
		t.Errorf("expected http://test.org/electrical, got %s", results[0].URI)
	}

	// Search by English label should still work.
	results = searchAndParse(t, store, map[string]any{"query": "wiring"})
	if len(results) == 0 {
		t.Fatal("expected to find resource via rdfs:label match")
	}
}

func TestSearch_ZeroResultHint(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Seed some data so resource count > 0.
	store.AddTriple("", "http://test.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/A", rdfsLabel, "Alpha Resource", "literal", "", "")
	store.AddTriple("", "http://test.org/B", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple("", "http://test.org/B", rdfsLabel, "Beta Note", "literal", "", "")

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "zzzznonexistent"}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)
	if res.IsError {
		t.Fatalf("search returned error: %s", text)
	}

	// Parse as hint object.
	var hint struct {
		Results []SearchResult `json:"results"`
		Hint    string         `json:"hint"`
	}
	if err := json.Unmarshal([]byte(text), &hint); err != nil {
		t.Fatalf("expected hint JSON object, got: %s", text)
	}

	if len(hint.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(hint.Results))
	}
	if hint.Hint == "" {
		t.Fatal("expected non-empty hint")
	}

	// Hint should mention the query and resource count.
	if !strings.Contains(hint.Hint, "zzzznonexistent") {
		t.Errorf("hint should contain the query, got: %s", hint.Hint)
	}
	if !strings.Contains(hint.Hint, "2 resources") {
		t.Errorf("hint should mention resource count (2), got: %s", hint.Hint)
	}
}

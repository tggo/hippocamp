package tools

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestSearch_RelatedViaRelatedTo(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	testHippoRelatedTo := testHippoNS + "relatedTo"

	// Topic A with label "Networking".
	store.AddTriple("", "http://test.org/topic/networking", rdfType, testHippoTopic, "uri", "", "")
	store.AddTriple("", "http://test.org/topic/networking", rdfsLabel, "Networking", "literal", "", "")

	// Entity B linked to A via hippo:relatedTo. B has NO "networking" in any field.
	store.AddTriple("", "http://test.org/entity/firewall", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/entity/firewall", rdfsLabel, "Firewall Config", "literal", "", "")
	store.AddTriple("", "http://test.org/entity/firewall", testHippoRelatedTo, "http://test.org/topic/networking", "uri", "", "")

	// Without related: only the topic.
	results := searchAndParse(t, store, map[string]any{"query": "Networking"})
	if len(results) != 1 {
		t.Errorf("without related: expected 1 result, got %d", len(results))
	}

	// With related: topic + firewall entity via relatedTo.
	results = searchAndParse(t, store, map[string]any{"query": "Networking", "related": true})
	foundTopic := false
	foundFirewall := false
	for _, r := range results {
		if r.URI == "http://test.org/topic/networking" {
			foundTopic = true
		}
		if r.URI == "http://test.org/entity/firewall" {
			foundFirewall = true
		}
	}
	if !foundTopic {
		t.Error("expected to find Networking topic (direct match)")
	}
	if !foundFirewall {
		t.Error("expected to find Firewall Config entity (via hippo:relatedTo)")
	}
}

func TestSearch_TypeFilterOnRelated(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Topic "Budget" — this is the direct match.
	store.AddTriple("", "http://test.org/topic/budget", rdfType, testHippoTopic, "uri", "", "")
	store.AddTriple("", "http://test.org/topic/budget", rdfsLabel, "Budget", "literal", "", "")

	// Entity "Cost" linked via hasTopic to Budget.
	store.AddTriple("", "http://test.org/entity/cost", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/entity/cost", rdfsLabel, "Cost Analysis", "literal", "", "")
	store.AddTriple("", "http://test.org/entity/cost", testHippoHasTopic, "http://test.org/topic/budget", "uri", "", "")

	// Note "Report" linked via hasTopic to Budget.
	store.AddTriple("", "http://test.org/note/report", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple("", "http://test.org/note/report", rdfsLabel, "Financial Report", "literal", "", "")
	store.AddTriple("", "http://test.org/note/report", testHippoHasTopic, "http://test.org/topic/budget", "uri", "", "")

	// First: search with related=true but NO type filter — should find all three.
	allResults := searchAndParse(t, store, map[string]any{
		"query":   "Budget",
		"related": true,
	})
	if len(allResults) != 3 {
		t.Fatalf("without type filter: expected 3 results (topic + entity + note), got %d", len(allResults))
	}

	// Now: search with related=true and type filter for Topic — only the direct match.
	topicResults := searchAndParse(t, store, map[string]any{
		"query":   "Budget",
		"related": true,
		"type":    testHippoTopic,
	})
	foundTopic := false
	for _, r := range topicResults {
		if r.URI == "http://test.org/topic/budget" {
			foundTopic = true
		}
		if r.URI == "http://test.org/entity/cost" {
			t.Error("Entity should be filtered out by type filter for Topic")
		}
		if r.URI == "http://test.org/note/report" {
			t.Error("Note should be filtered out by type filter for Topic")
		}
	}
	if !foundTopic {
		t.Error("expected to find Budget topic (direct match with matching type)")
	}

	// Also: type filter for Entity — direct match (Topic) is filtered out, so no
	// related traversal occurs (no direct matches survive). Entity should NOT appear.
	entityResults := searchAndParse(t, store, map[string]any{
		"query":   "Budget",
		"related": true,
		"type":    testHippoEntity,
	})
	if len(entityResults) != 0 {
		t.Errorf("type=Entity filters out the only direct match (Topic), so related traversal should not run; got %d results", len(entityResults))
	}
}

func TestSearch_ZeroResultHintWithScope(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Seed resources into a specific named graph.
	graphName := "http://test.org/graph/scoped"
	store.AddTriple(graphName, "http://test.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple(graphName, "http://test.org/A", rdfsLabel, "Alpha Resource", "literal", "", "")
	store.AddTriple(graphName, "http://test.org/B", rdfType, testHippoNote, "uri", "", "")
	store.AddTriple(graphName, "http://test.org/B", rdfsLabel, "Beta Note", "literal", "", "")
	store.AddTriple(graphName, "http://test.org/C", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple(graphName, "http://test.org/C", rdfsLabel, "Gamma Entity", "literal", "", "")

	// Also seed resources in a different graph to ensure they are NOT counted.
	store.AddTriple("http://test.org/graph/other", "http://test.org/D", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("http://test.org/graph/other", "http://test.org/D", rdfsLabel, "Delta Entity", "literal", "", "")

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "zzzznonexistent",
		"scope": graphName,
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)

	var hint struct {
		Results []SearchResult `json:"results"`
		Hint    string         `json:"hint"`
	}
	if err := json.Unmarshal([]byte(text), &hint); err != nil {
		t.Fatalf("expected hint JSON, got: %s", text)
	}

	if len(hint.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(hint.Results))
	}

	// Hint should report 3 resources (only the scoped graph), not 4.
	if !strings.Contains(hint.Hint, "3 resources") {
		t.Errorf("hint should report 3 resources for scoped graph, got: %s", hint.Hint)
	}
}

func TestSearch_PrefixMatchOnSubjectURI(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Resource whose URI contains "budgetplan" but label does NOT.
	store.AddTriple("", "http://test.org/budgetplan/item1", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/budgetplan/item1", rdfsLabel, "Monthly Allocation", "literal", "", "")

	// Search "budge" — 5 chars, would trigger prefix matching on predicate values.
	// Subject URI contains "budge" as a substring, so it should be found via Contains.
	// But we want to verify prefix matching logic does NOT apply to subject URI scan.
	results := searchAndParse(t, store, map[string]any{"query": "budge"})

	// The subject URI "http://test.org/budgetplan/item1" contains "budge" as substring,
	// so it should be found by the Contains check (lines 213-238).
	if len(results) == 0 {
		t.Fatal("expected to find resource via subject URI substring match")
	}

	// Now search with a keyword that shares a 4+ char prefix with "budgetplan" but is NOT
	// a substring of the URI — e.g. "budgetx". If prefix matching applied to URIs,
	// it would match. Since it doesn't, it should NOT match via URI (only via Contains).
	store2 := rdfstore.NewStore()
	defer store2.Close()

	store2.AddTriple("", "http://test.org/budgetplan/item1", rdfType, testHippoEntity, "uri", "", "")
	store2.AddTriple("", "http://test.org/budgetplan/item1", rdfsLabel, "Monthly Allocation", "literal", "", "")

	results2 := searchAndParse(t, store2, map[string]any{"query": "budgetx"})
	// "budgetx" is NOT a substring of "http://test.org/budgetplan/item1"
	// and prefix matching should NOT apply to subject URIs.
	// The label "Monthly Allocation" also doesn't match.
	if len(results2) != 0 {
		t.Errorf("expected no results (prefix matching should not apply to subject URIs), got %d", len(results2))
	}
}

func TestSearch_NegativeLimit(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Seed more than 20 resources so we can verify the default limit.
	for i := 0; i < 25; i++ {
		uri := fmt.Sprintf("http://test.org/item/%d", i)
		store.AddTriple("", uri, rdfType, testHippoEntity, "uri", "", "")
		store.AddTriple("", uri, rdfsLabel, fmt.Sprintf("Searchable Item %d", i), "literal", "", "")
	}

	// Search with negative limit — should default to 20.
	results := searchAndParse(t, store, map[string]any{"query": "Searchable", "limit": -5})
	if len(results) != 20 {
		t.Errorf("negative limit should default to 20, got %d results", len(results))
	}
}

func TestSearch_EmptyWordsPrefix(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Resource with an empty label — words slice will be empty.
	store.AddTriple("", "http://test.org/empty", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/empty", rdfsLabel, "", "literal", "", "")

	// Search with a keyword that is long enough to trigger prefix matching (4+ chars).
	// This should not panic or error when encountering the empty words slice.
	results := searchAndParse(t, store, map[string]any{"query": "testing"})

	// We don't expect to find the empty-label resource.
	for _, r := range results {
		if r.URI == "http://test.org/empty" {
			t.Error("empty label should not match any keyword")
		}
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

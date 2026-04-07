package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// TestSearchQuality_TemporalReranking verifies that temporal boost actually
// changes ranking order, not just finds results.
func TestSearchQuality_TemporalReranking(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	g := "urn:quality:temporal"
	store.CreateGraph(g)

	// Two decisions with identical text relevance but different timestamps.
	seed := []struct{ s, p, o, ot string }{
		// Fresh decision — created today-ish
		{"urn:q:fresh", rdfType, hippoNS + "Decision", "uri"},
		{"urn:q:fresh", rdfsLabel, "Use microservices", "literal"},
		{"urn:q:fresh", hippoSummary, "Architecture decision for the backend", "literal"},
		{"urn:q:fresh", hippoRationale, "Better scaling", "literal"},
		{"urn:q:fresh", hippoCreatedAt, "2026-04-07T09:00:00", "literal"},

		// Stale decision — created a year ago
		{"urn:q:stale", rdfType, hippoNS + "Decision", "uri"},
		{"urn:q:stale", rdfsLabel, "Use monolith", "literal"},
		{"urn:q:stale", hippoSummary, "Architecture decision for the backend", "literal"},
		{"urn:q:stale", hippoRationale, "Simpler deployment", "literal"},
		{"urn:q:stale", hippoCreatedAt, "2025-02-01T09:00:00", "literal"},

		// Unrelated entity — should not appear for "architecture decision"
		{"urn:q:unrelated", rdfType, hippoNS + "Entity", "uri"},
		{"urn:q:unrelated", rdfsLabel, "Redis Cache", "literal"},
		{"urn:q:unrelated", hippoSummary, "In-memory data store for caching", "literal"},
		{"urn:q:unrelated", hippoCreatedAt, "2026-04-07T08:00:00", "literal"},
	}
	for _, tr := range seed {
		if err := store.AddTriple(g, tr.s, tr.p, tr.o, tr.ot, "", ""); err != nil {
			t.Fatal(err)
		}
	}

	handler := HandlerFor(store, "search")

	// --- Test 1: Without temporal signal, both decisions should have equal/similar ranking ---
	t.Run("without_temporal_equal_text_scores", func(t *testing.T) {
		results := searchJSON(t, handler, map[string]any{
			"query": "architecture decision backend",
			"scope": g,
		})
		if len(results) < 2 {
			t.Fatalf("expected >= 2 results, got %d", len(results))
		}

		// Both should be found.
		found := map[string]bool{}
		for _, r := range results {
			found[r.URI] = true
		}
		if !found["urn:q:fresh"] || !found["urn:q:stale"] {
			t.Error("expected both fresh and stale decisions")
		}

		// Without temporal, their text scores should be identical.
		// (Same summary text = same score.)
		if results[0].Score != results[1].Score {
			t.Logf("NOTE: scores differ without temporal: %d vs %d (may differ by label)", results[0].Score, results[1].Score)
		}

		// Redis should NOT appear.
		if found["urn:q:unrelated"] {
			t.Error("unrelated Redis entity should not match 'architecture decision backend'")
		}
	})

	// --- Test 2: With "today", fresh decision MUST rank above stale ---
	t.Run("temporal_today_reranks", func(t *testing.T) {
		results := searchJSON(t, handler, map[string]any{
			"query":   "architecture decision today",
			"scope":   g,
			"explain": true,
		})
		if len(results) < 2 {
			t.Fatalf("expected >= 2 results, got %d", len(results))
		}

		// Fresh must be #1.
		if results[0].URI != "urn:q:fresh" {
			t.Errorf("expected fresh decision as #1, got %s (score=%d, conf=%.1f)", results[0].URI, results[0].Score, results[0].Confidence)
			for i, r := range results {
				t.Logf("  [%d] %s score=%d conf=%.1f temporal=%.2f", i, r.URI, r.Score, r.Confidence, explainTemporal(r))
			}
		}

		// Confidence gap should be meaningful (not both at 99-100%).
		gap := results[0].Confidence - results[1].Confidence
		t.Logf("confidence gap: %.1f%% (fresh=%.1f%%, stale=%.1f%%)", gap, results[0].Confidence, results[1].Confidence)
		if gap < 5 {
			t.Errorf("confidence gap = %.1f%%, want >= 5%% to be useful for LLMs", gap)
		}

		// Fresh should have high temporal score, stale should have low.
		freshTemporal := explainTemporal(results[0])
		staleTemporal := explainTemporal(results[1])
		t.Logf("temporal scores: fresh=%.3f, stale=%.3f", freshTemporal, staleTemporal)
		if freshTemporal <= staleTemporal {
			t.Errorf("fresh temporal (%.3f) should be > stale temporal (%.3f)", freshTemporal, staleTemporal)
		}
	})

	// --- Test 3: With "february 2025", stale decision should rank above fresh ---
	t.Run("temporal_past_reranks_opposite", func(t *testing.T) {
		results := searchJSON(t, handler, map[string]any{
			"query":   "architecture decision february 2025",
			"scope":   g,
			"explain": true,
		})
		if len(results) < 2 {
			t.Fatalf("expected >= 2 results, got %d", len(results))
		}

		// Stale (Feb 2025) must be #1 when query asks for "february 2025".
		if results[0].URI != "urn:q:stale" {
			t.Errorf("expected stale decision as #1 for 'february 2025', got %s", results[0].URI)
			for i, r := range results {
				t.Logf("  [%d] %s score=%d conf=%.1f temporal=%.2f", i, r.URI, r.Score, r.Confidence, explainTemporal(r))
			}
		}
	})
}

// TestSearchQuality_ConfidenceDistribution checks that confidence values
// are spread out enough to be useful, not bunched at 95-100%.
func TestSearchQuality_ConfidenceDistribution(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	g := "urn:quality:confidence"
	store.CreateGraph(g)

	// Resources with varying relevance to "database" query.
	seed := []struct{ s, p, o, ot string }{
		// High relevance: "database" in label AND summary
		{"urn:c:db", rdfType, hippoNS + "Entity", "uri"},
		{"urn:c:db", rdfsLabel, "PostgreSQL Database", "literal"},
		{"urn:c:db", hippoSummary, "Primary relational database for user data", "literal"},
		{"urn:c:db", hippoContent, "PostgreSQL database handles all CRUD operations", "literal"},

		// Medium relevance: "database" only in summary
		{"urn:c:orm", rdfType, hippoNS + "Entity", "uri"},
		{"urn:c:orm", rdfsLabel, "ORM Layer", "literal"},
		{"urn:c:orm", hippoSummary, "Abstraction over the database connection pool", "literal"},

		// Low relevance: "database" only in content (weight=1)
		{"urn:c:config", rdfType, hippoNS + "Entity", "uri"},
		{"urn:c:config", rdfsLabel, "Config Service", "literal"},
		{"urn:c:config", hippoSummary, "Application configuration management", "literal"},
		{"urn:c:config", hippoContent, "Stores database connection strings", "literal"},

		// No relevance: should not appear
		{"urn:c:auth", rdfType, hippoNS + "Entity", "uri"},
		{"urn:c:auth", rdfsLabel, "Auth Service", "literal"},
		{"urn:c:auth", hippoSummary, "Handles JWT token validation", "literal"},
	}
	for _, tr := range seed {
		if err := store.AddTriple(g, tr.s, tr.p, tr.o, tr.ot, "", ""); err != nil {
			t.Fatal(err)
		}
	}

	handler := HandlerFor(store, "search")
	results := searchJSON(t, handler, map[string]any{
		"query": "database",
		"scope": g,
	})

	if len(results) < 3 {
		t.Fatalf("expected >= 3 results, got %d", len(results))
	}

	// Log the confidence distribution.
	t.Log("Confidence distribution for 'database':")
	for i, r := range results {
		t.Logf("  [%d] %s — score=%d, confidence=%.1f%%", i, r.Label, r.Score, r.Confidence)
	}

	// Top result must be PostgreSQL Database (label + summary + content match).
	if results[0].URI != "urn:c:db" {
		t.Errorf("expected PostgreSQL Database as #1, got %s", results[0].Label)
	}

	// Confidence should be monotonically non-increasing.
	for i := 1; i < len(results); i++ {
		if results[i].Confidence > results[i-1].Confidence {
			t.Errorf("confidence not monotonic: [%d]=%.1f > [%d]=%.1f", i, results[i].Confidence, i-1, results[i-1].Confidence)
		}
	}

	// The spread should be meaningful: min confidence < 70% of max.
	last := results[len(results)-1]
	spread := results[0].Confidence - last.Confidence
	t.Logf("Confidence spread: %.1f%% (top=%.1f%%, bottom=%.1f%%)", spread, results[0].Confidence, last.Confidence)
	if spread < 30 {
		t.Errorf("confidence spread = %.1f%%, want >= 30%% for useful differentiation", spread)
	}

	// Auth service should NOT appear.
	for _, r := range results {
		if r.URI == "urn:c:auth" {
			t.Error("Auth Service should not match 'database'")
		}
	}
}

// TestSearchQuality_PrecisionUnderPressure seeds 200 resources and checks
// that a specific query doesn't return noise.
func TestSearchQuality_PrecisionUnderPressure(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	g := "urn:quality:pressure"
	store.CreateGraph(g)

	// Seed 3 relevant resources.
	for _, tr := range []struct{ s, p, o, ot string }{
		{"urn:p:auth-svc", rdfType, hippoNS + "Entity", "uri"},
		{"urn:p:auth-svc", rdfsLabel, "Authentication Service", "literal"},
		{"urn:p:auth-svc", hippoSummary, "Handles OAuth2 authentication flows", "literal"},

		{"urn:p:auth-middleware", rdfType, hippoNS + "Entity", "uri"},
		{"urn:p:auth-middleware", rdfsLabel, "Auth Middleware", "literal"},
		{"urn:p:auth-middleware", hippoSummary, "Express middleware that validates authentication tokens", "literal"},

		{"urn:p:auth-decision", rdfType, hippoNS + "Decision", "uri"},
		{"urn:p:auth-decision", rdfsLabel, "Use JWT for authentication", "literal"},
		{"urn:p:auth-decision", hippoRationale, "Stateless authentication scales better", "literal"},
	} {
		if err := store.AddTriple(g, tr.s, tr.p, tr.o, tr.ot, "", ""); err != nil {
			t.Fatal(err)
		}
	}

	// Seed 200 irrelevant resources (various topics, none mentioning "auth").
	noiseTopics := []string{
		"payment processing", "email notifications", "user profile management",
		"image upload pipeline", "search indexing engine", "rate limiting proxy",
		"database migration tool", "log aggregation service", "CI/CD pipeline runner",
		"feature flag controller",
	}
	for i := 0; i < 200; i++ {
		uri := fmt.Sprintf("urn:p:noise-%d", i)
		label := fmt.Sprintf("Service %d", i)
		summary := noiseTopics[i%len(noiseTopics)]
		store.AddTriple(g, uri, rdfType, hippoNS+"Entity", "uri", "", "")
		store.AddTriple(g, uri, rdfsLabel, label, "literal", "", "")
		store.AddTriple(g, uri, hippoSummary, summary, "literal", "", "")
	}

	handler := HandlerFor(store, "search")
	results := searchJSON(t, handler, map[string]any{
		"query": "authentication",
		"scope": g,
		"limit": 10.0,
	})

	t.Logf("Search 'authentication' in 203 resources: got %d results", len(results))
	for i, r := range results {
		t.Logf("  [%d] %s (conf=%.1f%%)", i, r.Label, r.Confidence)
	}

	// Should find all 3 auth-related resources.
	found := map[string]bool{}
	for _, r := range results {
		found[r.URI] = true
	}
	for _, want := range []string{"urn:p:auth-svc", "urn:p:auth-middleware", "urn:p:auth-decision"} {
		if !found[want] {
			t.Errorf("RECALL: missing %s", want)
		}
	}

	// No noise should appear in the results — none of them mention "auth".
	noiseCount := 0
	for _, r := range results {
		if len(r.URI) > 9 && r.URI[:9] == "urn:p:noi" {
			noiseCount++
		}
	}
	if noiseCount > 0 {
		t.Errorf("PRECISION: %d noise results in top %d", noiseCount, len(results))
	}

	// Result count should be exactly 3 (only the relevant ones).
	if len(results) != 3 {
		t.Errorf("expected exactly 3 results, got %d", len(results))
	}
}

// --- helpers ---

func searchJSON(t *testing.T, handler handlerFunc, args map[string]any) []SearchResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	text := ResultText(res)
	if res.IsError {
		t.Fatalf("search error: %s", text)
	}
	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		// Try hint format.
		var hint struct {
			Results []SearchResult `json:"results"`
		}
		if err2 := json.Unmarshal([]byte(text), &hint); err2 != nil {
			t.Fatalf("unmarshal: %v (text: %s)", err, text)
		}
		return hint.Results
	}
	return results
}

func explainTemporal(r SearchResult) float64 {
	if r.Explain == nil {
		return 0
	}
	return r.Explain.TemporalScore
}

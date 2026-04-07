package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// --- Temporal parsing unit tests ---

func TestParseTemporalRange_Today(t *testing.T) {
	ref := time.Date(2026, 4, 6, 14, 30, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("what happened today", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'today'")
	}
	if tr.Label != "today" {
		t.Errorf("label = %q, want 'today'", tr.Label)
	}
	expectDate(t, tr.Start, 2026, 4, 6)
	expectDate(t, tr.End, 2026, 4, 6)
}

func TestParseTemporalRange_Yesterday(t *testing.T) {
	ref := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("decisions yesterday", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'yesterday'")
	}
	expectDate(t, tr.Start, 2026, 4, 5)
}

func TestParseTemporalRange_LastWeek(t *testing.T) {
	// 2026-04-06 is Monday. Last week = Mon Mar 30 – Sun Apr 5.
	ref := time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("notes from last week", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'last week'")
	}
	expectDate(t, tr.Start, 2026, 3, 30)
	expectDate(t, tr.End, 2026, 4, 5)
}

func TestParseTemporalRange_ThisWeek(t *testing.T) {
	ref := time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC) // Monday
	tr := parseTemporalRangeWithRef("this week", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'this week'")
	}
	expectDate(t, tr.Start, 2026, 4, 6)
	expectDate(t, tr.End, 2026, 4, 12)
}

func TestParseTemporalRange_LastMonth(t *testing.T) {
	ref := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("last month summary", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'last month'")
	}
	expectDate(t, tr.Start, 2026, 3, 1)
	expectDate(t, tr.End, 2026, 3, 31)
}

func TestParseTemporalRange_ThisMonth(t *testing.T) {
	ref := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("this month", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'this month'")
	}
	expectDate(t, tr.Start, 2026, 4, 1)
	expectDate(t, tr.End, 2026, 4, 30)
}

func TestParseTemporalRange_Recent(t *testing.T) {
	ref := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("recent decisions", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'recent'")
	}
	if tr.Label != "recent (last 7 days)" {
		t.Errorf("label = %q, want 'recent (last 7 days)'", tr.Label)
	}
	expectDate(t, tr.Start, 2026, 3, 31)
	expectDate(t, tr.End, 2026, 4, 6)
}

func TestParseTemporalRange_ISODate(t *testing.T) {
	tr := parseTemporalRangeWithRef("notes from 2026-03-25", time.Now())
	if tr == nil {
		t.Fatal("expected temporal range for ISO date")
	}
	expectDate(t, tr.Start, 2026, 3, 25)
	expectDate(t, tr.End, 2026, 3, 25)
}

func TestParseTemporalRange_MonthYear(t *testing.T) {
	tr := parseTemporalRangeWithRef("notes from march 2026", time.Now())
	if tr == nil {
		t.Fatal("expected temporal range for 'march 2026'")
	}
	expectDate(t, tr.Start, 2026, 3, 1)
	expectDate(t, tr.End, 2026, 3, 31)
}

func TestParseTemporalRange_BareMonth(t *testing.T) {
	ref := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)
	tr := parseTemporalRangeWithRef("february notes", ref)
	if tr == nil {
		t.Fatal("expected temporal range for 'february'")
	}
	expectDate(t, tr.Start, 2026, 2, 1)
	expectDate(t, tr.End, 2026, 2, 28)
}

func TestParseTemporalRange_NoMatch(t *testing.T) {
	tr := parseTemporalRangeWithRef("how does authentication work", time.Now())
	if tr != nil {
		t.Errorf("expected nil for non-temporal query, got %+v", tr)
	}
}

func TestParseTemporalRange_Empty(t *testing.T) {
	tr := parseTemporalRangeWithRef("", time.Now())
	if tr != nil {
		t.Errorf("expected nil for empty query, got %+v", tr)
	}
}

// --- Temporal proximity scoring ---

func TestTemporalProximityScore_InsideRange(t *testing.T) {
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 26, 23, 59, 59, 0, time.UTC)
	note := time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC)
	score := temporalProximityScore(note, start, end)
	if score != 1.0 {
		t.Errorf("inside range score = %f, want 1.0", score)
	}
}

func TestTemporalProximityScore_AtBoundary(t *testing.T) {
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 26, 23, 59, 59, 0, time.UTC)
	if s := temporalProximityScore(start, start, end); s != 1.0 {
		t.Errorf("at start boundary = %f, want 1.0", s)
	}
	if s := temporalProximityScore(end, start, end); s != 1.0 {
		t.Errorf("at end boundary = %f, want 1.0", s)
	}
}

func TestTemporalProximityScore_OneDayBefore(t *testing.T) {
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 26, 23, 59, 59, 0, time.UTC)
	note := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	score := temporalProximityScore(note, start, end)
	expected := 1.0 / (1.0 + 1.0*0.1) // ~0.909
	if diff := score - expected; diff > 0.01 || diff < -0.01 {
		t.Errorf("1 day before score = %f, want ~%f", score, expected)
	}
}

func TestTemporalProximityScore_FarAway(t *testing.T) {
	start := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 26, 23, 59, 59, 0, time.UTC)
	note := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	score := temporalProximityScore(note, start, end)
	if score >= 0.1 {
		t.Errorf("far away score = %f, want < 0.1", score)
	}
	if score <= 0 {
		t.Errorf("far away score = %f, want > 0", score)
	}
}

// --- Strip temporal keywords ---

func TestStripTemporalKeywords(t *testing.T) {
	tests := []struct {
		input []string
		want  []string
	}{
		{[]string{"decisions", "last", "week"}, []string{"decisions"}},
		{[]string{"auth", "today"}, []string{"auth"}},
		{[]string{"march", "2026", "notes"}, []string{"notes"}},
		{[]string{"notes", "from", "2026-03-25"}, []string{"notes", "from"}},
		{[]string{"last", "week"}, nil}, // all temporal → empty
		{[]string{"authentication"}, []string{"authentication"}},
	}
	for _, tc := range tests {
		got := stripTemporalKeywords(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("stripTemporalKeywords(%v) = %v, want %v", tc.input, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("stripTemporalKeywords(%v)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

// --- Confidence normalization ---

func TestConfidenceNormalization(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "store triple persistence"}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	text := ResultText(res)

	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results")
	}

	// Top result should have confidence = 100%.
	if results[0].Confidence != 100.0 {
		t.Errorf("top result confidence = %f, want 100.0", results[0].Confidence)
	}

	// All results should have confidence > 0 and <= 100.
	for i, r := range results {
		if r.Confidence <= 0 || r.Confidence > 100 {
			t.Errorf("result[%d] confidence = %f, want (0, 100]", i, r.Confidence)
		}
		if r.Score <= 0 {
			t.Errorf("result[%d] score = %d, want > 0", i, r.Score)
		}
	}

	// If there are multiple results, later ones should have lower confidence.
	if len(results) > 1 {
		if results[1].Confidence > results[0].Confidence {
			t.Errorf("second result confidence (%f) > first (%f)", results[1].Confidence, results[0].Confidence)
		}
	}
}

// --- Explain mode ---

func TestSearchExplain(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "triple", "explain": true}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected results")
	}

	// All results should have Explain when explain=true.
	for i, r := range results {
		if r.Explain == nil {
			t.Errorf("result[%d] (%s): expected explain detail", i, r.URI)
			continue
		}
		if len(r.Explain.FieldScores) == 0 {
			t.Errorf("result[%d] (%s): expected non-empty field scores", i, r.URI)
		}
	}

	// Top result (AddTriple) should show label and summary scores.
	top := results[0]
	if top.Explain.FieldScores["rdfs:label"] == 0 {
		t.Error("expected rdfs:label score > 0 for AddTriple")
	}
}

func TestSearchExplainOff(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedSearchData(t, store)

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": "triple"}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for i, r := range results {
		if r.Explain != nil {
			t.Errorf("result[%d]: explain should be nil when explain=false", i)
		}
	}
}

// --- Temporal search integration ---

func seedTemporalData(t *testing.T, store *rdfstore.Store) {
	t.Helper()
	graph := "urn:test:temporal"
	store.CreateGraph(graph)

	// Resource created "today" (use a fixed recent date).
	for _, tr := range []struct{ s, p, o, ot string }{
		{"urn:test:recent-decision", rdfType, hippoNS + "Decision", "uri"},
		{"urn:test:recent-decision", rdfsLabel, "Use RRF for search", "literal"},
		{"urn:test:recent-decision", hippoRationale, "Better ranking quality", "literal"},
		{"urn:test:recent-decision", hippoCreatedAt, "2026-04-06T10:00:00", "literal"},

		{"urn:test:old-decision", rdfType, hippoNS + "Decision", "uri"},
		{"urn:test:old-decision", rdfsLabel, "Use BadgerDB backend", "literal"},
		{"urn:test:old-decision", hippoRationale, "Context-aware store needed", "literal"},
		{"urn:test:old-decision", hippoCreatedAt, "2025-01-15T09:00:00", "literal"},

		{"urn:test:march-note", rdfType, hippoNS + "Note", "uri"},
		{"urn:test:march-note", rdfsLabel, "March architecture review", "literal"},
		{"urn:test:march-note", hippoContent, "Discussed search improvements", "literal"},
		{"urn:test:march-note", hippoCreatedAt, "2026-03-15T14:00:00", "literal"},

		{"urn:test:no-date-note", rdfType, hippoNS + "Note", "uri"},
		{"urn:test:no-date-note", rdfsLabel, "Timeless design principles", "literal"},
		{"urn:test:no-date-note", hippoContent, "Search should be fast", "literal"},
	} {
		if err := store.AddTriple(graph, tr.s, tr.p, tr.o, tr.ot, "", ""); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSearchTemporal_TodayBoostsRecent(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedTemporalData(t, store)

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":   "decision today",
		"scope":   "urn:test:temporal",
		"explain": true,
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatal(err)
	}

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	// Both decisions should match "decision" keyword.
	// Recent one should rank higher due to temporal boost.
	if results[0].URI != "urn:test:recent-decision" {
		t.Errorf("expected recent-decision as top result, got %s", results[0].URI)
	}

	// Recent should have higher confidence than old.
	if results[0].Confidence <= results[1].Confidence {
		t.Errorf("recent confidence (%f) should be > old (%f)", results[0].Confidence, results[1].Confidence)
	}

	// Explain should show temporal score > 0 for the recent one.
	if results[0].Explain == nil || results[0].Explain.TemporalScore == 0 {
		t.Error("expected temporal score > 0 for recent decision")
	}
	if results[0].Explain != nil && results[0].Explain.TemporalRange != "today" {
		t.Errorf("temporal range = %q, want 'today'", results[0].Explain.TemporalRange)
	}
}

func TestSearchTemporal_MarchFindsNote(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedTemporalData(t, store)

	handler := HandlerFor(store, "search")
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query":   "architecture march 2026",
		"scope":   "urn:test:temporal",
		"explain": true,
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	var results []SearchResult
	if err := json.Unmarshal([]byte(ResultText(res)), &results); err != nil {
		t.Fatal(err)
	}

	if len(results) == 0 {
		t.Fatal("expected results for 'architecture march 2026'")
	}

	found := false
	for _, r := range results {
		if r.URI == "urn:test:march-note" {
			found = true
			if r.Explain != nil && r.Explain.TemporalScore < 0.5 {
				t.Errorf("march-note temporal score = %f, expected >= 0.5", r.Explain.TemporalScore)
			}
		}
	}
	if !found {
		t.Error("expected to find march-note in results")
	}
}

func TestSearchTemporal_PureTemporalQuery(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedTemporalData(t, store)

	handler := HandlerFor(store, "search")
	// When all keywords are temporal, the original keywords are used for text matching.
	// "this month" alone should still find resources created this month.
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"query": "this month",
		"scope": "urn:test:temporal",
	}

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}

	// This is an edge case — "this" and "month" are temporal keywords.
	// stripTemporalKeywords returns empty, so we fall back to original keywords.
	// The text match on "this" and "month" might not match anything useful,
	// but temporal-only results (Phase 4) should kick in.
	text := ResultText(res)
	// Just verify no error — the behavior depends on current date.
	if res.IsError {
		t.Errorf("expected no error, got: %s", text)
	}
}

// --- Helper ---

func expectDate(t *testing.T, got time.Time, year int, month time.Month, day int) {
	t.Helper()
	if got.Year() != year || got.Month() != month || got.Day() != day {
		t.Errorf("date = %s, want %d-%02d-%02d", got.Format("2006-01-02"), year, month, day)
	}
}

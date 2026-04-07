package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

const (
	rdfType     = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfsLabel   = "http://www.w3.org/2000/01/rdf-schema#label"
	hippoNS     = "https://hippocamp.dev/ontology#"
	hippoSummary   = hippoNS + "summary"
	hippoFilePath  = hippoNS + "filePath"
	hippoSignature = hippoNS + "signature"
	hippoContent    = hippoNS + "content"
	hippoURL        = hippoNS + "url"
	hippoRationale  = hippoNS + "rationale"
	hippoStatus_    = hippoNS + "status"
	hippoAlias      = hippoNS + "alias"
	hippoCreatedAt  = hippoNS + "createdAt"
	hippoUpdatedAt  = hippoNS + "updatedAt"
)

// fieldWeight maps searchable predicates to their scoring weight.
// Higher weight = matches in this field rank higher.
var fieldWeight = map[string]int{
	rdfsLabel:       4, // label matches are most relevant
	hippoSummary:    3, // summary is a concise description
	hippoAlias:      3, // synonyms, colloquial terms
	hippoFilePath:   2,
	hippoSignature:  2,
	hippoContent:    1, // content is long text, lower signal
	hippoURL:        1,
	hippoRationale:  2,
	hippoStatus_:    1,
}

// searchablePredicates derived from fieldWeight keys.
var searchablePredicates = func() map[string]bool {
	m := make(map[string]bool, len(fieldWeight))
	for k := range fieldWeight {
		m[k] = true
	}
	return m
}()

// SearchResult is a single hit returned by the search tool.
type SearchResult struct {
	URI        string            `json:"uri"`
	Score      int               `json:"score"`
	Confidence float64           `json:"confidence"` // 0-100%, top result = 100%
	Type       string            `json:"type,omitempty"`
	Label      string            `json:"label,omitempty"`
	Summary    string            `json:"summary,omitempty"`
	Props      map[string]string `json:"props,omitempty"`
	Explain    *ExplainDetail    `json:"explain,omitempty"`
}

// ExplainDetail breaks down how a search result was scored.
type ExplainDetail struct {
	FieldScores    map[string]int `json:"field_scores"`              // per-predicate score contributions
	TemporalScore  float64        `json:"temporal_score,omitempty"`  // 0.0-1.0 temporal proximity
	RelatedFrom    string         `json:"related_from,omitempty"`    // URI of the direct match this was linked from
	TemporalRange  string         `json:"temporal_range,omitempty"`  // parsed date range description
}

// temporalRange represents a parsed time window from the query.
type temporalRange struct {
	Start time.Time
	End   time.Time
	Label string // human description like "last week", "2026-03-25"
}

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription(`Semantic search over the RDF knowledge graph. Finds resources by keyword matching across labels, summaries, aliases, file paths, and signatures.

Results include confidence scores (0-100%, top result = 100%) for easy ranking comparison.

Supports temporal search: queries like "decisions today", "notes last week", "march 2026" automatically detect date ranges and boost resources whose hippo:createdAt/hippo:updatedAt fall within that range. Supported: today, yesterday, this/last week, this/last month, recent, ISO dates (2026-03-25), month names with optional year.

Supports multilingual search: add hippo:alias triples with language tags for synonyms and translations (e.g. hippo:alias "електрика"@uk). Aliases are searched with the same boost as summaries.

Uses prefix matching: keywords sharing 4+ characters with stored words match even without exact substring (e.g. "електр" matches "електропостачання").

Examples:
  {"query": "authentication"}
  {"query": "decisions last week"}
  {"query": "Store", "type": "https://hippocamp.dev/ontology#Struct"}
  {"query": "triple", "scope": "project:hippocamp", "limit": 5}
  {"query": "auth", "explain": true}

Returns an array of matching resources with their type, label, summary, score, confidence (0-100%), and properties. Set explain=true for per-field score breakdown. Returns a hint object with suggestions when no results are found.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search keywords (matched case-insensitively against labels, summaries, aliases, file paths, signatures). Supports prefix matching for cross-language stems."),
		),
		mcp.WithString("type",
			mcp.Description("Filter by rdf:type URI (e.g. https://hippocamp.dev/ontology#Function)"),
		),
		mcp.WithString("scope",
			mcp.Description("Named graph URI to search in (omit to search all graphs)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum results to return (default: 20)"),
		),
		mcp.WithBoolean("related",
			mcp.Description("Include resources linked to direct matches via hasTopic, references, partOf (1-hop graph traversal). Default: false."),
		),
		mcp.WithBoolean("explain",
			mcp.Description("Include per-field score breakdown in each result. Shows which fields matched and their individual scores. Default: false."),
		),
	)
}

func searchHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: query"), nil
		}

		typeFilter := req.GetString("type", "")
		scope := req.GetString("scope", "")
		limit := int(req.GetFloat("limit", 20))
		if limit <= 0 {
			limit = 20
		}
		related := req.GetBool("related", false)
		explain := req.GetBool("explain", false)

		results, err := searchGraph(store, query, typeFilter, scope, limit, related, explain)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search error: %v", err)), nil
		}

		if len(results) == 0 {
			// Count distinct subjects in the searched scope to give context.
			resourceCount := countResources(store, scope)
			hint := struct {
				Results []SearchResult `json:"results"`
				Hint    string         `json:"hint"`
			}{
				Results: []SearchResult{},
				Hint:    fmt.Sprintf("0 matches for '%s'. Graph has %d resources. Try: English terms, rdfs:label values, or hippo:alias values.", query, resourceCount),
			}
			data, err := json.Marshal(hint)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(string(data)), nil
		}

		data, err := json.Marshal(results)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

// searchGraph performs a keyword search over the RDF graph.
// It iterates over all triples, finds subjects whose searchable predicates
// contain any of the query keywords, then enriches results with type/label/summary.
// Relationship predicates that connect resources (used for related search).
var relationPredicates = map[string]bool{
	hippoNS + "hasTopic":    true,
	hippoNS + "references":  true,
	hippoNS + "partOf":      true,
	hippoNS + "relatedTo":   true,
}

func searchGraph(store *rdfstore.Store, query, typeFilter, scope string, limit int, related, explain bool) ([]SearchResult, error) {
	keywords := strings.Fields(strings.ToLower(query))
	if len(keywords) == 0 {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Parse temporal range from query (e.g. "last week", "yesterday", "March 2026").
	tRange := parseTemporalRange(query)

	// Strip temporal keywords from the query for text matching.
	// Only do this if we have a temporal range AND non-temporal keywords remain.
	textKeywords := keywords
	if tRange != nil {
		stripped := stripTemporalKeywords(keywords)
		if len(stripped) > 0 {
			textKeywords = stripped
		}
		// If all keywords are temporal, still search with original keywords
	}

	// Collect graphs to search.
	graphNames := []string{}
	if scope != "" {
		graphNames = append(graphNames, scope)
	} else {
		graphNames = store.ListGraphs()
	}

	// Phase 1: find matching subject URIs by scanning searchable predicates.
	type candidate struct {
		graph       string
		score       int
		fieldScores map[string]int // predicate → score (for explain)
		relatedFrom string         // if populated, this is a related result
	}
	candidates := make(map[string]*candidate) // subject URI → candidate

	for _, gn := range graphNames {
		triples, err := store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		for _, t := range triples {
			weight, ok := fieldWeight[t.Predicate]
			if !ok {
				continue
			}
			val := strings.ToLower(t.Object)
			words := strings.Fields(val)
			score := 0
			for _, kw := range textKeywords {
				if strings.Contains(val, kw) {
					matchScore := weight // base: field weight
					// Word boundary bonus: keyword matches at the start of a word.
					for _, w := range words {
						if strings.HasPrefix(w, kw) {
							matchScore += weight // double for word-start match
							break
						}
					}
					score += matchScore
				} else if len(kw) >= minPrefixLen {
					// Prefix matching: keyword and word share a common prefix of 4+ chars.
					// Catches stem variations (електр → електрика, електричний).
					if prefixMatch(kw, words) {
						score += weight / 2 // half weight for prefix match
					}
				}
			}
			if score > 0 {
				if c, ok := candidates[t.Subject]; ok {
					c.score += score // accumulate across predicates
					if c.fieldScores != nil {
						c.fieldScores[t.Predicate] += score
					}
				} else {
					fs := map[string]int{t.Predicate: score}
					candidates[t.Subject] = &candidate{graph: gn, score: score, fieldScores: fs}
				}
			}
		}
	}

	// Also match subject URIs themselves against keywords (low weight).
	for _, gn := range graphNames {
		triples, err := store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		seen := map[string]bool{}
		for _, t := range triples {
			if seen[t.Subject] {
				continue
			}
			seen[t.Subject] = true
			subLower := strings.ToLower(t.Subject)
			score := 0
			for _, kw := range textKeywords {
				if strings.Contains(subLower, kw) {
					score++
				}
			}
			if score > 0 {
				if c, ok := candidates[t.Subject]; ok {
					c.score += score
					if c.fieldScores != nil {
						c.fieldScores["uri"] += score
					}
				} else {
					fs := map[string]int{"uri": score}
					candidates[t.Subject] = &candidate{graph: gn, score: score, fieldScores: fs}
				}
			}
		}
	}

	// Phase 2: enrich candidates with type, label, summary, and other properties.
	// Also apply type filter and temporal scoring.
	type scored struct {
		result        SearchResult
		score         int
		temporalScore float64
		finalScore    float64 // combined score for ranking
	}
	var results []scored

	for subj, c := range candidates {
		triples, err := store.ListTriples(c.graph, subj, "", "")
		if err != nil {
			continue
		}

		sr := SearchResult{
			URI:   subj,
			Props: make(map[string]string),
		}

		var createdAt, updatedAt string
		for _, t := range triples {
			switch t.Predicate {
			case rdfType:
				sr.Type = t.Object
			case rdfsLabel:
				sr.Label = t.Object
			case hippoSummary:
				sr.Summary = t.Object
			case hippoCreatedAt:
				createdAt = t.Object
			case hippoUpdatedAt:
				updatedAt = t.Object
			default:
				// Store additional properties with short key.
				key := t.Predicate
				if strings.HasPrefix(key, hippoNS) {
					key = "hippo:" + strings.TrimPrefix(key, hippoNS)
				}
				sr.Props[key] = t.Object
			}
		}

		// Apply type filter.
		if typeFilter != "" && sr.Type != typeFilter {
			continue
		}

		// Use subject URI fragment as label fallback.
		if sr.Label == "" {
			if idx := strings.LastIndexAny(subj, "/#"); idx >= 0 {
				sr.Label = subj[idx+1:]
			}
		}

		// Temporal scoring: boost results whose timestamps fall in the temporal range.
		var tScore float64
		if tRange != nil {
			tScore = computeTemporalScore(createdAt, updatedAt, tRange)
		}

		// Combined score: text score + temporal boost.
		// Temporal adds up to 50% of max text score as a bonus lane.
		finalScore := float64(c.score)
		if tRange != nil && tScore > 0 {
			finalScore += tScore * float64(c.score) * 0.5
		}

		sr.Score = c.score

		if explain {
			// Shorten predicate keys for readability.
			shortFS := make(map[string]int, len(c.fieldScores))
			for pred, s := range c.fieldScores {
				key := pred
				if strings.HasPrefix(key, hippoNS) {
					key = "hippo:" + strings.TrimPrefix(key, hippoNS)
				} else if strings.HasPrefix(key, "http://www.w3.org/2000/01/rdf-schema#") {
					key = "rdfs:" + strings.TrimPrefix(key, "http://www.w3.org/2000/01/rdf-schema#")
				}
				shortFS[key] = s
			}
			detail := &ExplainDetail{
				FieldScores:   shortFS,
				TemporalScore: tScore,
			}
			if tRange != nil {
				detail.TemporalRange = tRange.Label
			}
			sr.Explain = detail
		}

		results = append(results, scored{result: sr, score: c.score, temporalScore: tScore, finalScore: finalScore})
	}

	// Phase 3: Graph-aware search — find resources that link TO direct matches.
	if related && len(results) > 0 {
		// Collect direct match URIs.
		directURIs := make(map[string]bool, len(results))
		for _, r := range results {
			directURIs[r.result.URI] = true
		}

		// Scan all triples for relationship predicates pointing to direct matches.
		for _, gn := range graphNames {
			triples, err := store.ListTriples(gn, "", "", "")
			if err != nil {
				continue
			}
			for _, t := range triples {
				if !relationPredicates[t.Predicate] {
					continue
				}
				// If this triple points TO a direct match and the subject is NOT already a result...
				if directURIs[t.Object] && !directURIs[t.Subject] {
					// Enrich this related subject.
					relTriples, _ := store.ListTriples(gn, t.Subject, "", "")
					sr := SearchResult{
						URI:   t.Subject,
						Props: make(map[string]string),
					}
					for _, rt := range relTriples {
						switch rt.Predicate {
						case rdfType:
							sr.Type = rt.Object
						case rdfsLabel:
							sr.Label = rt.Object
						case hippoSummary:
							sr.Summary = rt.Object
						default:
							key := rt.Predicate
							if strings.HasPrefix(key, hippoNS) {
								key = "hippo:" + strings.TrimPrefix(key, hippoNS)
							}
							sr.Props[key] = rt.Object
						}
					}
					if sr.Label == "" {
						if idx := strings.LastIndexAny(t.Subject, "/#"); idx >= 0 {
							sr.Label = t.Subject[idx+1:]
						}
					}
					if typeFilter != "" && sr.Type != typeFilter {
						continue
					}
					// Related results get a lower score than direct matches.
					sr.Score = 1
					if explain {
						sr.Explain = &ExplainDetail{
							FieldScores: map[string]int{"related": 1},
							RelatedFrom: t.Object,
						}
					}
					results = append(results, scored{result: sr, score: 1, finalScore: 1})
					directURIs[t.Subject] = true // prevent duplicates
				}
			}
		}
	}

	// Phase 4: Temporal-only results — if we have a temporal range but few text matches,
	// also surface resources that match temporally even without text match.
	if tRange != nil && len(results) < limit {
		existingURIs := make(map[string]bool, len(results))
		for _, r := range results {
			existingURIs[r.result.URI] = true
		}
		for _, gn := range graphNames {
			triples, err := store.ListTriples(gn, "", "", "")
			if err != nil {
				continue
			}
			// Collect subjects with timestamps.
			subjectTimestamps := make(map[string][2]string) // subject → [createdAt, updatedAt]
			for _, t := range triples {
				if t.Predicate == hippoCreatedAt {
					ts := subjectTimestamps[t.Subject]
					ts[0] = t.Object
					subjectTimestamps[t.Subject] = ts
				} else if t.Predicate == hippoUpdatedAt {
					ts := subjectTimestamps[t.Subject]
					ts[1] = t.Object
					subjectTimestamps[t.Subject] = ts
				}
			}
			for subj, ts := range subjectTimestamps {
				if existingURIs[subj] {
					continue
				}
				tScore := computeTemporalScore(ts[0], ts[1], tRange)
				if tScore < 0.5 {
					continue // only add if good temporal match
				}
				// Enrich
				subjTriples, _ := store.ListTriples(gn, subj, "", "")
				sr := SearchResult{URI: subj, Props: make(map[string]string), Score: 0}
				for _, st := range subjTriples {
					switch st.Predicate {
					case rdfType:
						sr.Type = st.Object
					case rdfsLabel:
						sr.Label = st.Object
					case hippoSummary:
						sr.Summary = st.Object
					default:
						key := st.Predicate
						if strings.HasPrefix(key, hippoNS) {
							key = "hippo:" + strings.TrimPrefix(key, hippoNS)
						}
						sr.Props[key] = st.Object
					}
				}
				if typeFilter != "" && sr.Type != typeFilter {
					continue
				}
				if sr.Label == "" {
					if idx := strings.LastIndexAny(subj, "/#"); idx >= 0 {
						sr.Label = subj[idx+1:]
					}
				}
				if explain {
					sr.Explain = &ExplainDetail{
						FieldScores:   map[string]int{},
						TemporalScore: tScore,
						TemporalRange: tRange.Label,
					}
				}
				results = append(results, scored{result: sr, score: 0, temporalScore: tScore, finalScore: tScore * 10})
				existingURIs[subj] = true
			}
		}
	}

	// Sort by finalScore descending.
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].finalScore > results[j-1].finalScore; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	// Apply limit.
	if len(results) > limit {
		results = results[:limit]
	}

	// Confidence normalization: top result = 100%, rest proportional.
	var maxScore float64
	if len(results) > 0 {
		maxScore = results[0].finalScore
	}

	out := make([]SearchResult, len(results))
	for i, r := range results {
		r.result.Score = r.score
		if maxScore > 0 {
			r.result.Confidence = math.Round(r.finalScore/maxScore*1000) / 10 // one decimal place
		}
		out[i] = r.result
	}
	return out, nil
}

// minPrefixLen is the minimum shared prefix length for prefix matching.
// Shorter prefixes produce too many false positives.
const minPrefixLen = 4

// prefixMatch returns true if keyword shares a prefix of minPrefixLen+ chars
// with any word in words. This catches stem variations across languages
// (e.g. "електр" matches "електрика", "електричний", "електропостачання").
func prefixMatch(kw string, words []string) bool {
	kwRunes := []rune(kw)
	for _, w := range words {
		wRunes := []rune(w)
		shared := 0
		limit := len(kwRunes)
		if len(wRunes) < limit {
			limit = len(wRunes)
		}
		for i := 0; i < limit; i++ {
			if kwRunes[i] == wRunes[i] {
				shared++
			} else {
				break
			}
		}
		if shared >= minPrefixLen {
			return true
		}
	}
	return false
}

// --- Temporal search ---

// parseTemporalRange extracts a date range from natural-language temporal keywords in the query.
// Returns nil if no temporal signal is found.
func parseTemporalRange(query string) *temporalRange {
	return parseTemporalRangeWithRef(query, time.Now())
}

// parseTemporalRangeWithRef is the testable version with injectable reference time.
func parseTemporalRangeWithRef(query string, now time.Time) *temporalRange {
	lower := strings.ToLower(query)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// "today" / "this morning"
	if strings.Contains(lower, "today") || strings.Contains(lower, "this morning") {
		return &temporalRange{Start: today, End: endOfDay(today), Label: "today"}
	}

	// "yesterday"
	if strings.Contains(lower, "yesterday") {
		y := today.AddDate(0, 0, -1)
		return &temporalRange{Start: y, End: endOfDay(y), Label: "yesterday"}
	}

	// "last week"
	if strings.Contains(lower, "last week") {
		mon := mondayOfWeek(today).AddDate(0, 0, -7)
		sun := mon.AddDate(0, 0, 6)
		return &temporalRange{Start: mon, End: endOfDay(sun), Label: "last week"}
	}

	// "this week"
	if strings.Contains(lower, "this week") {
		mon := mondayOfWeek(today)
		sun := mon.AddDate(0, 0, 6)
		return &temporalRange{Start: mon, End: endOfDay(sun), Label: "this week"}
	}

	// "last month"
	if strings.Contains(lower, "last month") {
		first := time.Date(today.Year(), today.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return &temporalRange{Start: first, End: endOfDay(last), Label: "last month"}
	}

	// "this month"
	if strings.Contains(lower, "this month") {
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(today.Year(), today.Month()+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return &temporalRange{Start: first, End: endOfDay(last), Label: "this month"}
	}

	// "recent" / "recently"
	if strings.Contains(lower, "recent") {
		weekAgo := today.AddDate(0, 0, -6)
		return &temporalRange{Start: weekAgo, End: endOfDay(today), Label: "recent (last 7 days)"}
	}

	// ISO date: "2026-03-25"
	if tr := findISODate(lower); tr != nil {
		return tr
	}

	// "Month Year" or bare month: "March 2026", "march"
	if tr := parseMonthWithOptionalYear(lower, today.Year()); tr != nil {
		return tr
	}

	return nil
}

// computeTemporalScore scores a resource by proximity to the temporal range.
// Uses the best (most recent relevant) of createdAt and updatedAt.
// Inside range: 1.0. Outside: smooth decay 1/(1 + days_away * 0.1).
func computeTemporalScore(createdAt, updatedAt string, tr *temporalRange) float64 {
	var bestScore float64
	for _, ts := range []string{createdAt, updatedAt} {
		if ts == "" {
			continue
		}
		t, err := parseISO8601(ts)
		if err != nil {
			continue
		}
		s := temporalProximityScore(t, tr.Start, tr.End)
		if s > bestScore {
			bestScore = s
		}
	}
	return bestScore
}

// temporalProximityScore: 1.0 inside range, smooth decay outside.
func temporalProximityScore(noteTime, rangeStart, rangeEnd time.Time) float64 {
	if !noteTime.Before(rangeStart) && !noteTime.After(rangeEnd) {
		return 1.0
	}
	var daysAway float64
	if noteTime.Before(rangeStart) {
		daysAway = rangeStart.Sub(noteTime).Hours() / 24
	} else {
		daysAway = noteTime.Sub(rangeEnd).Hours() / 24
	}
	return 1.0 / (1.0 + daysAway*0.1)
}

// parseISO8601 parses ISO 8601 datetime strings (with or without time part).
func parseISO8601(s string) (time.Time, error) {
	// Try full datetime first.
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as ISO 8601", s)
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, time.UTC)
}

func mondayOfWeek(t time.Time) time.Time {
	wd := t.Weekday()
	if wd == time.Sunday {
		wd = 7
	}
	return t.AddDate(0, 0, -int(wd-time.Monday))
}

// findISODate looks for YYYY-MM-DD in the query string.
func findISODate(query string) *temporalRange {
	for i := 0; i <= len(query)-10; i++ {
		if query[i+4] == '-' && query[i+7] == '-' {
			candidate := query[i : i+10]
			if t, err := time.Parse("2006-01-02", candidate); err == nil {
				return &temporalRange{Start: t, End: endOfDay(t), Label: candidate}
			}
		}
	}
	return nil
}

var monthNames = map[string]time.Month{
	"january": time.January, "jan": time.January,
	"february": time.February, "feb": time.February,
	"march": time.March, "mar": time.March,
	"april": time.April, "apr": time.April,
	"may": time.May,
	"june": time.June, "jun": time.June,
	"july": time.July, "jul": time.July,
	"august": time.August, "aug": time.August,
	"september": time.September, "sep": time.September,
	"october": time.October, "oct": time.October,
	"november": time.November, "nov": time.November,
	"december": time.December, "dec": time.December,
}

func parseMonthWithOptionalYear(query string, currentYear int) *temporalRange {
	words := strings.Fields(query)
	for i, w := range words {
		month, ok := monthNames[w]
		if !ok {
			continue
		}
		year := currentYear
		if i+1 < len(words) {
			if y := 0; len(words[i+1]) == 4 {
				_, err := fmt.Sscanf(words[i+1], "%d", &y)
				if err == nil && y >= 1900 && y <= 2100 {
					year = y
				}
			}
		}
		first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(year, month+1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
		return &temporalRange{Start: first, End: endOfDay(last), Label: fmt.Sprintf("%s %d", month, year)}
	}
	return nil
}

// temporalKeywords is the set of words consumed by temporal parsing.
var temporalKeywords = map[string]bool{
	"today": true, "yesterday": true, "this": true, "last": true,
	"week": true, "month": true, "morning": true, "recent": true, "recently": true,
}

func init() {
	for k := range monthNames {
		temporalKeywords[k] = true
	}
}

// stripTemporalKeywords removes temporal keywords from the keyword list,
// keeping non-temporal search terms intact.
func stripTemporalKeywords(keywords []string) []string {
	var out []string
	for _, kw := range keywords {
		if temporalKeywords[kw] {
			continue
		}
		// Skip 4-digit years.
		if len(kw) == 4 && kw[0] >= '1' && kw[0] <= '2' {
			isYear := true
			for _, c := range kw {
				if c < '0' || c > '9' {
					isYear = false
					break
				}
			}
			if isYear {
				continue
			}
		}
		// Skip ISO dates.
		if len(kw) == 10 && kw[4] == '-' && kw[7] == '-' {
			continue
		}
		out = append(out, kw)
	}
	return out
}

// countResources counts distinct subjects across the given scope (or all graphs).
func countResources(store *rdfstore.Store, scope string) int {
	graphNames := []string{}
	if scope != "" {
		graphNames = append(graphNames, scope)
	} else {
		graphNames = store.ListGraphs()
	}

	subjects := make(map[string]bool)
	for _, gn := range graphNames {
		triples, err := store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		for _, t := range triples {
			subjects[t.Subject] = true
		}
	}
	return len(subjects)
}

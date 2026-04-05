package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
)

// fieldWeight maps searchable predicates to their scoring weight.
// Higher weight = matches in this field rank higher.
var fieldWeight = map[string]int{
	rdfsLabel:       4, // label matches are most relevant
	hippoSummary:    3, // summary is a concise description
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
	URI     string            `json:"uri"`
	Type    string            `json:"type,omitempty"`
	Label   string            `json:"label,omitempty"`
	Summary string            `json:"summary,omitempty"`
	Props   map[string]string `json:"props,omitempty"`
}

func searchTool() mcp.Tool {
	return mcp.NewTool("search",
		mcp.WithDescription(`Semantic search over the RDF knowledge graph. Finds resources by keyword matching across labels, summaries, file paths, and signatures.

Examples:
  {"query": "authentication"}
  {"query": "Store", "type": "https://hippocamp.dev/ontology#Struct"}
  {"query": "triple", "scope": "project:hippocamp", "limit": 5}

Returns an array of matching resources with their type, label, summary, and properties.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search keywords (matched case-insensitively against labels, summaries, file paths, signatures)"),
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

		results, err := searchGraph(store, query, typeFilter, scope, limit, related)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("search error: %v", err)), nil
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

func searchGraph(store *rdfstore.Store, query, typeFilter, scope string, limit int, related bool) ([]SearchResult, error) {
	keywords := strings.Fields(strings.ToLower(query))
	if len(keywords) == 0 {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Collect graphs to search.
	graphNames := []string{}
	if scope != "" {
		graphNames = append(graphNames, scope)
	} else {
		graphNames = store.ListGraphs()
	}

	// Phase 1: find matching subject URIs by scanning searchable predicates.
	// Score = sum of (keyword matches * field weight * word-boundary bonus) across all predicates.
	type candidate struct {
		graph string
		score int
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
			for _, kw := range keywords {
				if !strings.Contains(val, kw) {
					continue
				}
				matchScore := weight // base: field weight
				// Word boundary bonus: keyword matches at the start of a word.
				for _, w := range words {
					if strings.HasPrefix(w, kw) {
						matchScore += weight // double for word-start match
						break
					}
				}
				score += matchScore
			}
			if score > 0 {
				if c, ok := candidates[t.Subject]; ok {
					c.score += score // accumulate across predicates
				} else {
					candidates[t.Subject] = &candidate{graph: gn, score: score}
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
			for _, kw := range keywords {
				if strings.Contains(subLower, kw) {
					score++
				}
			}
			if score > 0 {
				if c, ok := candidates[t.Subject]; ok {
					c.score += score
				} else {
					candidates[t.Subject] = &candidate{graph: gn, score: score}
				}
			}
		}
	}

	// Phase 2: enrich candidates with type, label, summary, and other properties.
	// Also apply type filter.
	type scored struct {
		result SearchResult
		score  int
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

		for _, t := range triples {
			switch t.Predicate {
			case rdfType:
				sr.Type = t.Object
			case rdfsLabel:
				sr.Label = t.Object
			case hippoSummary:
				sr.Summary = t.Object
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

		results = append(results, scored{result: sr, score: c.score})
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
					results = append(results, scored{result: sr, score: 1})
					directURIs[t.Subject] = true // prevent duplicates
				}
			}
		}
	}

	// Sort by score descending (simple insertion sort — result sets are small).
	for i := 1; i < len(results); i++ {
		for j := i; j > 0 && results[j].score > results[j-1].score; j-- {
			results[j], results[j-1] = results[j-1], results[j]
		}
	}

	// Apply limit.
	if len(results) > limit {
		results = results[:limit]
	}

	out := make([]SearchResult, len(results))
	for i, r := range results {
		out[i] = r.result
	}
	return out, nil
}

// Package healthcheck runs background graph validation and caches results.
// The validate tool reads from the cache for instant responses.
// The cache invalidates on any store mutation (tracked via Store.IsDirty).
package healthcheck

import (
	"log"
	"sync"
	"time"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

const (
	rdfType        = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	rdfsLabel      = "http://www.w3.org/2000/01/rdf-schema#label"
	hippoNS        = "https://hippocamp.dev/ontology#"
	analyticsGraph = "urn:hippocamp:analytics"
	analyticsNS    = "http://purl.org/hippocamp/analytics#"
)

// relationPredicates are object properties where the object should be a valid resource.
var relationPredicates = map[string]bool{
	hippoNS + "hasTopic":    true,
	hippoNS + "references":  true,
	hippoNS + "partOf":      true,
	hippoNS + "relatedTo":   true,
	hippoNS + "hasTag":      true,
	hippoNS + "sourceOf":    true,
}

// Report is the cached validation result.
type Report struct {
	Timestamp       time.Time `json:"timestamp"`
	DanglingRefs    []DanglingRef    `json:"dangling_refs,omitempty"`
	OrphanResources []string         `json:"orphan_resources,omitempty"`
	MissingAliases  []AliasSuggestion `json:"missing_aliases,omitempty"`
	Stats           Stats            `json:"stats"`
}

// DanglingRef is a relationship triple where the object doesn't exist as a subject.
type DanglingRef struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Graph     string `json:"graph"`
}

// AliasSuggestion recommends adding an alias based on zero-result search queries.
type AliasSuggestion struct {
	Query      string `json:"query"`
	Suggestion string `json:"suggestion"`
}

// Stats summarizes the graph health.
type Stats struct {
	Resources    int `json:"resources"`
	DanglingRefs int `json:"dangling_refs"`
	Orphans      int `json:"orphan_resources"`
}

// Checker runs background validation and caches results.
type Checker struct {
	store *rdfstore.Store

	mu     sync.RWMutex
	report *Report
	dirty  bool // tracks if store changed since last scan
}

// New creates a Checker and starts background scanning.
func New(store *rdfstore.Store, interval time.Duration) *Checker {
	c := &Checker{
		store: store,
		dirty: true, // force initial scan
	}
	go c.loop(interval)
	return c
}

// Report returns the latest cached report (may be nil on first call).
func (c *Checker) Report() *Report {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.report
}

// MarkDirty signals that the store was mutated and results need refresh.
func (c *Checker) MarkDirty() {
	c.mu.Lock()
	c.dirty = true
	c.mu.Unlock()
}

func (c *Checker) loop(interval time.Duration) {
	// Initial scan immediately.
	c.scan()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.RLock()
		needsScan := c.dirty
		c.mu.RUnlock()
		if needsScan {
			c.scan()
		}
	}
}

func (c *Checker) scan() {
	start := time.Now()
	report := c.buildReport()
	report.Timestamp = start

	c.mu.Lock()
	c.report = report
	c.dirty = false
	c.mu.Unlock()

	total := report.Stats.DanglingRefs + report.Stats.Orphans + len(report.MissingAliases)
	if total > 0 {
		log.Printf("healthcheck: %d issues found (%d dangling, %d orphans, %d alias suggestions) in %s",
			total, report.Stats.DanglingRefs, report.Stats.Orphans, len(report.MissingAliases),
			time.Since(start).Round(time.Millisecond))
	}
}

func (c *Checker) buildReport() *Report {
	r := &Report{}

	graphs := c.store.ListGraphs()

	// Collect all subjects (resources that exist) across all graphs.
	// Skip the analytics graph.
	allSubjects := make(map[string]bool)
	// Track which subjects are referenced by relationship predicates (incoming links).
	referenced := make(map[string]bool)
	// Track which subjects have outgoing relationship links.
	hasOutgoing := make(map[string]bool)

	for _, gn := range graphs {
		if gn == analyticsGraph {
			continue
		}
		triples, err := c.store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		for _, t := range triples {
			allSubjects[t.Subject] = true
			if relationPredicates[t.Predicate] {
				referenced[t.Object] = true
				hasOutgoing[t.Subject] = true
			}
		}
	}

	r.Stats.Resources = len(allSubjects)

	// Find dangling references: relationship objects that don't exist as subjects.
	for _, gn := range graphs {
		if gn == analyticsGraph {
			continue
		}
		triples, err := c.store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		for _, t := range triples {
			if !relationPredicates[t.Predicate] {
				continue
			}
			if t.ObjType != "uri" {
				continue
			}
			if !allSubjects[t.Object] {
				r.DanglingRefs = append(r.DanglingRefs, DanglingRef{
					Subject:   t.Subject,
					Predicate: t.Predicate,
					Object:    t.Object,
					Graph:     gn,
				})
			}
		}
	}
	r.Stats.DanglingRefs = len(r.DanglingRefs)

	// Find orphan resources: typed resources with no incoming relationship links
	// and no outgoing relationship links (isolated nodes).
	for _, gn := range graphs {
		if gn == analyticsGraph {
			continue
		}
		triples, err := c.store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		seen := make(map[string]bool)
		for _, t := range triples {
			if seen[t.Subject] {
				continue
			}
			seen[t.Subject] = true
			if t.Predicate == rdfType && !referenced[t.Subject] && !hasOutgoing[t.Subject] {
				r.OrphanResources = append(r.OrphanResources, t.Subject)
			}
		}
	}
	r.Stats.Orphans = len(r.OrphanResources)

	// Analyze zero-result search queries from analytics for alias suggestions.
	r.MissingAliases = c.analyzeZeroResults()

	return r
}

// analyzeZeroResults finds search queries that returned 0 results.
func (c *Checker) analyzeZeroResults() []AliasSuggestion {
	triples, err := c.store.ListTriples(analyticsGraph, "", "", "")
	if err != nil || len(triples) == 0 {
		return nil
	}

	// Group triples by subject to find ToolCall resources.
	type call struct {
		tool        string
		input       string
		resultCount string
	}
	calls := make(map[string]*call)
	for _, t := range triples {
		c, ok := calls[t.Subject]
		if !ok {
			c = &call{}
			calls[t.Subject] = c
		}
		switch t.Predicate {
		case analyticsNS + "tool":
			c.tool = t.Object
		case analyticsNS + "input":
			c.input = t.Object
		case analyticsNS + "resultCount":
			c.resultCount = t.Object
		}
	}

	// Deduplicate zero-result queries.
	seen := make(map[string]bool)
	var suggestions []AliasSuggestion
	for _, c := range calls {
		if c.tool != "search" || c.resultCount != "0" || c.input == "" {
			continue
		}
		if seen[c.input] {
			continue
		}
		seen[c.input] = true
		suggestions = append(suggestions, AliasSuggestion{
			Query:      c.input,
			Suggestion: "Add hippo:alias triples matching these terms to relevant resources",
		})
	}
	return suggestions
}

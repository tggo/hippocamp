// Package analytics records tool call metrics as RDF triples in a dedicated named graph.
// It provides mcp-go hook functions that log every tool invocation (tool name, key parameters,
// result count, duration, errors) both to the Go logger and to the store's analytics graph.
package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// GraphURI is the named graph where analytics triples are stored.
const GraphURI = "urn:hippocamp:analytics"

// Namespace for analytics predicates.
const ns = "http://purl.org/hippocamp/analytics#"

// Collector hooks into mcp-go's OnBeforeCallTool / OnAfterCallTool
// to record timing and metadata for every tool invocation.
type Collector struct {
	store *rdfstore.Store

	mu     sync.Mutex
	starts map[any]callStart // keyed by request ID
	seq    int64             // monotonic counter for unique bnode IDs
}

type callStart struct {
	tool string
	args map[string]any
	t    time.Time
}

// New creates a Collector that writes analytics triples to store.
func New(store *rdfstore.Store) *Collector {
	// Ensure the analytics graph exists.
	_ = store.CreateGraph(GraphURI)
	return &Collector{
		store:  store,
		starts: make(map[any]callStart),
	}
}

// BeforeCallTool records the start time and parameters.
func (c *Collector) BeforeCallTool(ctx context.Context, id any, req *mcp.CallToolRequest) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.starts[id] = callStart{
		tool: req.Params.Name,
		args: req.GetArguments(),
		t:    time.Now(),
	}
}

// AfterCallTool logs the call and stores analytics triples.
func (c *Collector) AfterCallTool(ctx context.Context, id any, req *mcp.CallToolRequest, result any) {
	c.mu.Lock()
	start, ok := c.starts[id]
	if ok {
		delete(c.starts, id)
	}
	c.seq++
	seq := c.seq
	c.mu.Unlock()

	if !ok {
		return
	}

	// Skip recording queries that target the analytics graph itself to avoid loops.
	if isAnalyticsQuery(start.args) {
		return
	}

	duration := time.Since(start.t)
	toolName := start.tool
	args := start.args

	// Extract key parameters per tool type.
	info := extractInfo(toolName, args, result)

	// Level 1: log line.
	log.Printf("analytics: tool=%s %s duration=%s", toolName, info.logSuffix, duration.Round(time.Microsecond))

	// Level 2: store as triples.
	c.storeTriples(seq, toolName, args, info, duration)
}

// isAnalyticsQuery checks if the tool call targets the analytics graph.
func isAnalyticsQuery(args map[string]any) bool {
	if g := strArg(args, "graph"); g == GraphURI {
		return true
	}
	if g := strArg(args, "scope"); g == GraphURI {
		return true
	}
	// SPARQL queries mentioning the analytics graph URI.
	if q := strArg(args, "query"); strings.Contains(q, GraphURI) {
		return true
	}
	return false
}

// callInfo holds extracted metadata for logging and triple creation.
type callInfo struct {
	logSuffix   string
	resultCount int
	isError     bool
	input       string // key input parameter (query text, action, etc.)
}

func extractInfo(tool string, args map[string]any, result any) callInfo {
	ci := callInfo{}

	switch tool {
	case "search":
		ci.input = strArg(args, "query")
		ci.logSuffix = fmt.Sprintf("query=%q", ci.input)
	case "sparql":
		q := strArg(args, "query")
		// Truncate long queries for the log line.
		if len(q) > 120 {
			q = q[:120] + "..."
		}
		ci.input = strArg(args, "query")
		ci.logSuffix = fmt.Sprintf("query=%q", q)
	case "triple":
		action := strArg(args, "action")
		ci.input = action
		subj := strArg(args, "subject")
		ci.logSuffix = fmt.Sprintf("action=%s subject=%s", action, subj)
	case "graph":
		action := strArg(args, "action")
		ci.input = action
		ci.logSuffix = fmt.Sprintf("action=%s", action)
	case "validate":
		ci.input = "validate"
		graph := strArg(args, "graph")
		if graph != "" {
			ci.logSuffix = fmt.Sprintf("graph=%s", graph)
		} else {
			ci.logSuffix = "graph=default"
		}
	default:
		ci.logSuffix = fmt.Sprintf("args=%v", args)
	}

	// Try to extract result count and error status.
	if r, ok := result.(*mcp.CallToolResult); ok && r != nil {
		ci.isError = r.IsError
		ci.resultCount = countResults(r)
	}

	if ci.isError {
		ci.logSuffix += " error=true"
	} else {
		ci.logSuffix += fmt.Sprintf(" results=%d", ci.resultCount)
	}

	return ci
}

// countResults tries to determine how many results were returned.
func countResults(r *mcp.CallToolResult) int {
	if r == nil || len(r.Content) == 0 {
		return 0
	}
	for _, c := range r.Content {
		tc, ok := c.(mcp.TextContent)
		if !ok {
			continue
		}
		text := tc.Text

		// Try to parse as JSON array (search results, SPARQL rows).
		var arr []json.RawMessage
		if json.Unmarshal([]byte(text), &arr) == nil {
			return len(arr)
		}

		// "ok" from SPARQL update, "true"/"false" from ASK.
		if text == "ok" || text == "true" || text == "false" {
			return 1
		}

		// Triple list returns JSON array.
		// Count lines as fallback.
		return strings.Count(text, "\n") + 1
	}
	return 0
}

// storeTriples writes one set of analytics triples per tool call.
func (c *Collector) storeTriples(seq int64, tool string, args map[string]any, info callInfo, duration time.Duration) {
	subject := fmt.Sprintf("urn:hippocamp:analytics:call:%d", seq)
	now := time.Now().UTC().Format(time.RFC3339)

	triples := []struct {
		pred, obj, objType, lang, dt string
	}{
		{"http://www.w3.org/1999/02/22-rdf-syntax-ns#type", ns + "ToolCall", "uri", "", ""},
		{ns + "tool", tool, "literal", "", ""},
		{ns + "timestamp", now, "literal", "", "http://www.w3.org/2001/XMLSchema#dateTime"},
		{ns + "durationMs", fmt.Sprintf("%d", duration.Milliseconds()), "literal", "", "http://www.w3.org/2001/XMLSchema#integer"},
		{ns + "resultCount", fmt.Sprintf("%d", info.resultCount), "literal", "", "http://www.w3.org/2001/XMLSchema#integer"},
	}

	if info.input != "" {
		triples = append(triples, struct {
			pred, obj, objType, lang, dt string
		}{ns + "input", info.input, "literal", "", ""})
	}

	if info.isError {
		triples = append(triples, struct {
			pred, obj, objType, lang, dt string
		}{ns + "error", "true", "literal", "", "http://www.w3.org/2001/XMLSchema#boolean"})
	}

	// Store graph name for scoped queries.
	if g := strArg(args, "graph"); g != "" {
		triples = append(triples, struct {
			pred, obj, objType, lang, dt string
		}{ns + "graph", g, "literal", "", ""})
	}

	for _, t := range triples {
		if err := c.store.AddTriple(GraphURI, subject, t.pred, t.obj, t.objType, t.lang, t.dt); err != nil {
			log.Printf("analytics: triple store error: %v", err)
		}
	}
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

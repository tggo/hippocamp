package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// ── constants ────────────────��──────────────────────────────

// hippoNS and rdfType are defined in search.go; reuse via package scope.
const (
	rdfsLbl = "http://www.w3.org/2000/01/rdf-schema#label"
)

// hub types excluded from god_nodes (naturally high-degree, not interesting)
var hubTypes = map[string]bool{
	hippoNS + "Topic":   true,
	hippoNS + "Tag":     true,
	hippoNS + "Project": true,
}

// metadata predicates excluded from "surprising" analysis
var metaPredicates = map[string]bool{
	rdfType:               true,
	rdfsLbl:               true,
	hippoNS + "summary":   true,
	hippoNS + "content":   true,
	hippoNS + "alias":     true,
	hippoNS + "status":    true,
	hippoNS + "createdAt": true,
	hippoNS + "updatedAt": true,
	hippoNS + "url":       true,
	hippoNS + "filePath":  true,
	hippoNS + "signature": true,
	hippoNS + "lineNumber": true,
	hippoNS + "rationale": true,
	hippoNS + "version":   true,
	hippoNS + "language":  true,
	hippoNS + "rootPath":  true,
	hippoNS + "confidence": true,
	hippoNS + "provenance": true,
	hippoNS + "source":     true,
	hippoNS + "validFrom": true,
	hippoNS + "validTo":   true,
}

// ── tool definition ─────────────────────────────────────────

func analyzeTool() mcp.Tool {
	return mcp.NewTool("analyze",
		mcp.WithDescription(`Analyze graph structure: find hubs, clusters, cross-topic bridges, and visualize.

Actions:
  god_nodes   — most-connected resources by degree (excludes Topic/Tag/Project hubs)
  components  — connected components (clusters) via BFS
  surprising  — cross-topic or cross-graph edges (bridges between clusters)
  export_html  — start a local HTTP server with interactive vis.js graph visualization
  consolidate — find resources with missing/sparse summaries and suggest enrichments with graph context

Examples:
  {"action":"god_nodes","limit":5}
  {"action":"components","scope":"urn:hippocamp:default"}
  {"action":"surprising"}
  {"action":"export_html"}
  {"action":"consolidate","limit":10}

Returns JSON arrays. export_html returns {"url":"http://localhost:PORT"}.`),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Analysis operation"),
			mcp.Enum("god_nodes", "components", "surprising", "export_html", "consolidate"),
		),
		mcp.WithString("scope",
			mcp.Description("Named graph URI to analyze (omit for all graphs)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results for god_nodes (default 10)"),
		),
	)
}

func analyzeHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter \"action\""), nil
		}
		scope := req.GetString("scope", "")

		switch action {
		case "god_nodes":
			limit := int(req.GetFloat("limit", 10))
			return handleGodNodes(store, scope, limit)
		case "components":
			return handleComponents(store, scope)
		case "surprising":
			return handleSurprising(store, scope)
		case "export_html":
			return handleExportHTML(store, scope)
		case "consolidate":
			limit := int(req.GetFloat("limit", 20))
			return handleConsolidate(store, scope, limit)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q", action)), nil
		}
	}
}

// ── triple collection ───────────────────────────────────────

type graphTriple struct {
	rdfstore.Triple
	Graph string
}

// collectTriples gathers all triples from the given scope (or all graphs).
func collectTriples(store *rdfstore.Store, scope string) ([]graphTriple, error) {
	var graphs []string
	if scope != "" {
		graphs = []string{scope}
	} else {
		graphs = store.ListGraphs()
	}

	var all []graphTriple
	for _, g := range graphs {
		triples, err := store.ListTriples(g, "", "", "")
		if err != nil {
			return nil, fmt.Errorf("list triples in %s: %w", g, err)
		}
		for _, t := range triples {
			all = append(all, graphTriple{Triple: t, Graph: g})
		}
	}
	return all, nil
}

// buildIndex builds lookup maps from a triple set.
type nodeInfo struct {
	URI       string
	Label     string
	Type      string
	Topics    []string
	InDegree  int
	OutDegree int
	Graph     string // first graph seen
}

func buildIndex(triples []graphTriple) map[string]*nodeInfo {
	idx := make(map[string]*nodeInfo)

	ensure := func(uri, graph string) *nodeInfo {
		n, ok := idx[uri]
		if !ok {
			n = &nodeInfo{URI: uri, Graph: graph}
			idx[uri] = n
		}
		return n
	}

	for _, t := range triples {
		subj := ensure(t.Subject, t.Graph)

		switch t.Predicate {
		case rdfType:
			subj.Type = t.Object
		case rdfsLbl:
			subj.Label = t.Object
		case hippoNS + "hasTopic":
			subj.Topics = append(subj.Topics, t.Object)
		}

		if t.ObjType == "uri" {
			ensure(t.Object, t.Graph)
		}
	}
	return idx
}

// ── god_nodes ───────────────────────────────────────────────

type GodNode struct {
	URI       string `json:"uri"`
	Label     string `json:"label,omitempty"`
	Type      string `json:"type,omitempty"`
	Degree    int    `json:"degree"`
	InDegree  int    `json:"in_degree"`
	OutDegree int    `json:"out_degree"`
}

func handleGodNodes(store *rdfstore.Store, scope string, limit int) (*mcp.CallToolResult, error) {
	triples, err := collectTriples(store, scope)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx := buildIndex(triples)

	// Count degrees (only relationship triples, skip metadata)
	for _, t := range triples {
		if metaPredicates[t.Predicate] {
			continue
		}
		if t.ObjType != "uri" {
			continue
		}
		if n, ok := idx[t.Subject]; ok {
			n.OutDegree++
		}
		if n, ok := idx[t.Object]; ok {
			n.InDegree++
		}
	}

	// Collect non-hub nodes with degree > 0
	var nodes []GodNode
	for _, n := range idx {
		degree := n.InDegree + n.OutDegree
		if degree == 0 {
			continue
		}
		if hubTypes[n.Type] {
			continue
		}
		nodes = append(nodes, GodNode{
			URI:       n.URI,
			Label:     n.Label,
			Type:      localName(n.Type),
			Degree:    degree,
			InDegree:  n.InDegree,
			OutDegree: n.OutDegree,
		})
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Degree > nodes[j].Degree })

	if len(nodes) > limit {
		nodes = nodes[:limit]
	}

	return jsonResult(nodes)
}

// ── components ──────────────────────────────────────────────

type Component struct {
	ID      int      `json:"id"`
	Size    int      `json:"size"`
	Members []string `json:"members"`          // URIs (capped at 20)
	Labels  []string `json:"labels,omitempty"`  // for context
	Topics  []string `json:"topics,omitempty"`  // distinct hasTopic values
}

func handleComponents(store *rdfstore.Store, scope string) (*mcp.CallToolResult, error) {
	triples, err := collectTriples(store, scope)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx := buildIndex(triples)

	// Build undirected adjacency (URI-to-URI only, skip metadata predicates)
	adj := make(map[string]map[string]bool)
	ensureAdj := func(u string) {
		if adj[u] == nil {
			adj[u] = make(map[string]bool)
		}
	}

	for _, t := range triples {
		if metaPredicates[t.Predicate] || t.ObjType != "uri" {
			continue
		}
		ensureAdj(t.Subject)
		ensureAdj(t.Object)
		adj[t.Subject][t.Object] = true
		adj[t.Object][t.Subject] = true
	}

	// Also add isolated typed nodes (have type but no relationship edges)
	for uri, n := range idx {
		if n.Type != "" && adj[uri] == nil {
			adj[uri] = make(map[string]bool)
		}
	}

	// BFS
	visited := make(map[string]bool)
	var components []Component
	id := 0

	for start := range adj {
		if visited[start] {
			continue
		}
		id++
		var members []string
		queue := []string{start}
		visited[start] = true

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			members = append(members, cur)
			for neighbor := range adj[cur] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}

		// Collect labels and topics
		labelSet := make(map[string]bool)
		topicSet := make(map[string]bool)
		for _, uri := range members {
			if n, ok := idx[uri]; ok {
				if n.Label != "" {
					labelSet[n.Label] = true
				}
				for _, t := range n.Topics {
					topicSet[localName(t)] = true
				}
			}
		}

		labels := setToSlice(labelSet)
		topics := setToSlice(topicSet)

		// Cap members list
		displayMembers := members
		if len(displayMembers) > 20 {
			displayMembers = displayMembers[:20]
		}

		components = append(components, Component{
			ID:      id,
			Size:    len(members),
			Members: displayMembers,
			Labels:  labels,
			Topics:  topics,
		})
	}

	// Sort by size descending
	sort.Slice(components, func(i, j int) bool { return components[i].Size > components[j].Size })

	return jsonResult(components)
}

// ── surprising ──────────────────────────────────────────────

type SurprisingEdge struct {
	Subject      string `json:"subject"`
	SubjectLabel string `json:"subject_label,omitempty"`
	Predicate    string `json:"predicate"`
	Object       string `json:"object"`
	ObjectLabel  string `json:"object_label,omitempty"`
	Reason       string `json:"reason"`
	SubjectTopic string `json:"subject_topic,omitempty"`
	ObjectTopic  string `json:"object_topic,omitempty"`
}

func handleSurprising(store *rdfstore.Store, scope string) (*mcp.CallToolResult, error) {
	triples, err := collectTriples(store, scope)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx := buildIndex(triples)

	var results []SurprisingEdge

	for _, t := range triples {
		if metaPredicates[t.Predicate] || t.ObjType != "uri" {
			continue
		}

		sn := idx[t.Subject]
		on := idx[t.Object]
		if sn == nil || on == nil {
			continue
		}

		var reasons []string

		// Cross-topic: subject and object have different hasTopic values
		if len(sn.Topics) > 0 && len(on.Topics) > 0 {
			if !hasOverlap(sn.Topics, on.Topics) {
				reasons = append(reasons, "cross-topic")
			}
		}

		// Cross-graph: subject and object first seen in different graphs
		if sn.Graph != on.Graph && sn.Graph != "" && on.Graph != "" {
			reasons = append(reasons, "cross-graph")
		}

		if len(reasons) == 0 {
			continue
		}

		sTopic := ""
		if len(sn.Topics) > 0 {
			sTopic = localName(sn.Topics[0])
		}
		oTopic := ""
		if len(on.Topics) > 0 {
			oTopic = localName(on.Topics[0])
		}

		results = append(results, SurprisingEdge{
			Subject:      t.Subject,
			SubjectLabel: sn.Label,
			Predicate:    localName(t.Predicate),
			Object:       t.Object,
			ObjectLabel:  on.Label,
			Reason:       strings.Join(reasons, ", "),
			SubjectTopic: sTopic,
			ObjectTopic:  oTopic,
		})
	}

	return jsonResult(results)
}

// ── consolidate ─────────────────────────────────────────────

type ConsolidateSuggestion struct {
	URI            string            `json:"uri"`
	Label          string            `json:"label,omitempty"`
	Type           string            `json:"type,omitempty"`
	Issue          string            `json:"issue"`
	Context        map[string][]string `json:"context"`
	SuggestedPrompt string           `json:"suggested_prompt"`
}

func handleConsolidate(store *rdfstore.Store, scope string, limit int) (*mcp.CallToolResult, error) {
	triples, err := collectTriples(store, scope)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	idx := buildIndex(triples)

	// Also collect summaries and hasTopic per resource.
	summaries := make(map[string]string)
	for _, t := range triples {
		switch t.Predicate {
		case hippoNS + "summary":
			summaries[t.Subject] = t.Object
		}
	}

	// Build reverse adjacency for "referenced_by".
	referencedBy := make(map[string][]string) // object → []subject labels
	references := make(map[string][]string)   // subject → []object labels
	decisions := make(map[string][]string)     // topic URI → []decision labels

	for _, t := range triples {
		if metaPredicates[t.Predicate] || t.ObjType != "uri" {
			continue
		}
		sLabel := idx[t.Subject].Label
		if sLabel == "" {
			sLabel = localName(t.Subject)
		}
		oLabel := ""
		if on, ok := idx[t.Object]; ok {
			oLabel = on.Label
			if oLabel == "" {
				oLabel = localName(t.Object)
			}
		}
		references[t.Subject] = appendUnique(references[t.Subject], oLabel)
		referencedBy[t.Object] = appendUnique(referencedBy[t.Object], sLabel)
	}

	// Collect decisions per topic.
	for _, t := range triples {
		if t.Predicate == hippoNS+"hasTopic" {
			if n, ok := idx[t.Subject]; ok && n.Type == hippoNS+"Decision" {
				decisions[t.Object] = appendUnique(decisions[t.Object], n.Label)
			}
		}
	}

	var suggestions []ConsolidateSuggestion

	for uri, n := range idx {
		if n.Type == "" {
			continue // skip untyped resources
		}

		var issue string
		summary := summaries[uri]

		if summary == "" {
			issue = "missing_summary"
		} else if len(summary) < 20 {
			issue = "sparse_summary"
		} else if len(n.Topics) == 0 && n.Type != hippoNS+"Topic" && n.Type != hippoNS+"Tag" {
			issue = "no_topic"
		} else {
			continue // resource is fine
		}

		ctx := make(map[string][]string)
		if refs := references[uri]; len(refs) > 0 {
			ctx["references"] = refs
		}
		if refBy := referencedBy[uri]; len(refBy) > 0 {
			ctx["referenced_by"] = refBy
		}
		if len(n.Topics) > 0 {
			topicNames := make([]string, len(n.Topics))
			for i, t := range n.Topics {
				topicNames[i] = localName(t)
			}
			ctx["topics"] = topicNames
			// Include related decisions.
			for _, t := range n.Topics {
				if decs := decisions[t]; len(decs) > 0 {
					ctx["related_decisions"] = appendUnique(ctx["related_decisions"], decs...)
				}
			}
		}

		// Build suggested prompt.
		action := "summary"
		if issue == "no_topic" {
			action = "hasTopic"
		}
		prompt := fmt.Sprintf("Add hippo:%s to %s (type: %s).", action, n.Label, localName(n.Type))
		if refs := ctx["references"]; len(refs) > 0 {
			prompt += fmt.Sprintf(" References: %s.", strings.Join(refs, ", "))
		}
		if refBy := ctx["referenced_by"]; len(refBy) > 0 {
			prompt += fmt.Sprintf(" Referenced by: %s.", strings.Join(refBy, ", "))
		}
		if topics := ctx["topics"]; len(topics) > 0 {
			prompt += fmt.Sprintf(" Topics: %s.", strings.Join(topics, ", "))
		}

		suggestions = append(suggestions, ConsolidateSuggestion{
			URI:             uri,
			Label:           n.Label,
			Type:            localName(n.Type),
			Issue:           issue,
			Context:         ctx,
			SuggestedPrompt: prompt,
		})
	}

	// Sort by issue severity: missing_summary first, then sparse, then no_topic.
	sort.SliceStable(suggestions, func(i, j int) bool {
		return issueOrder(suggestions[i].Issue) < issueOrder(suggestions[j].Issue)
	})

	// Apply limit after sorting (so most severe issues always appear first).
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return jsonResult(suggestions)
}

func issueOrder(issue string) int {
	switch issue {
	case "missing_summary":
		return 0
	case "sparse_summary":
		return 1
	case "no_topic":
		return 2
	default:
		return 3
	}
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

// ── export_html ─────────────────────────────────────────────

// vizPort holds the port of the running visualization server (0 = not started).
var vizPort int

// StartVisualizationServer starts the graph visualization HTTP server on the
// preferred port (trying 39322, 39323, ... up to 39332). Called from main.go
// on startup. The server dynamically renders the current graph state on each request.
func StartVisualizationServer(store *rdfstore.Store) (int, error) {
	const basePort = 39322

	var listener net.Listener
	var err error
	for port := basePort; port < basePort+10; port++ {
		listener, err = net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			break
		}
	}
	if listener == nil {
		return 0, fmt.Errorf("cannot bind ports %d–%d: %w", basePort, basePort+9, err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	vizPort = port

	mux := http.NewServeMux()

	// Dynamic handler: reads current graph state on every request
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := renderGraphHTML(store, "")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(html))
	})

	// JSON API for nodes/edges (for potential future use)
	mux.HandleFunc("/api/graph", func(w http.ResponseWriter, r *http.Request) {
		scope := r.URL.Query().Get("scope")
		nodes, edges := buildVisData(store, scope)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"nodes": nodes, "edges": edges})
	})

	srv := &http.Server{Handler: mux}
	go func() {
		srv.Serve(listener)
	}()

	return port, nil
}

func handleExportHTML(store *rdfstore.Store, scope string) (*mcp.CallToolResult, error) {
	if vizPort == 0 {
		return mcp.NewToolResultError("visualization server not started"), nil
	}
	url := fmt.Sprintf("http://localhost:%d", vizPort)
	if scope != "" {
		url += "?scope=" + scope
	}
	return jsonResult(map[string]string{"url": url, "status": "running"})
}

// ── vis.js data building ────────────────────────────────────

type visNode struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Type  string `json:"group"`
	Size  int    `json:"value"`
}

type visEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

func buildVisData(store *rdfstore.Store, scope string) ([]visNode, []visEdge) {
	triples, err := collectTriples(store, scope)
	if err != nil {
		return nil, nil
	}

	idx := buildIndex(triples)

	// Count degrees
	for _, t := range triples {
		if metaPredicates[t.Predicate] || t.ObjType != "uri" {
			continue
		}
		if n, ok := idx[t.Subject]; ok {
			n.OutDegree++
		}
		if n, ok := idx[t.Object]; ok {
			n.InDegree++
		}
	}

	var nodes []visNode
	for uri, n := range idx {
		if n.Type == "" && n.Label == "" {
			continue
		}
		label := n.Label
		if label == "" {
			label = localName(uri)
		}
		degree := n.InDegree + n.OutDegree
		if degree < 1 {
			degree = 1
		}
		nodes = append(nodes, visNode{
			ID:    uri,
			Label: label,
			Type:  localName(n.Type),
			Size:  degree,
		})
	}

	var edges []visEdge
	seen := make(map[string]bool)
	for _, t := range triples {
		if metaPredicates[t.Predicate] || t.ObjType != "uri" {
			continue
		}
		key := t.Subject + "|" + t.Predicate + "|" + t.Object
		if seen[key] {
			continue
		}
		seen[key] = true
		edges = append(edges, visEdge{
			From:  t.Subject,
			To:    t.Object,
			Label: localName(t.Predicate),
		})
	}

	return nodes, edges
}

func renderGraphHTML(store *rdfstore.Store, scope string) string {
	nodes, edges := buildVisData(store, scope)
	nodesJSON, _ := json.Marshal(nodes)
	edgesJSON, _ := json.Marshal(edges)
	return buildVisualizationHTML(string(nodesJSON), string(edgesJSON))
}

// ── HTML template ───────────────────────────────────────────

func buildVisualizationHTML(nodesJSON, edgesJSON string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Hippocamp — Knowledge Graph</title>
<script src="https://unpkg.com/vis-network/standalone/umd/vis-network.min.js"></script>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; display: flex; height: 100vh; background: #0a0a0a; color: #e0e0e0; }
#sidebar { width: 320px; padding: 16px; overflow-y: auto; background: #141414; border-right: 1px solid #333; display: flex; flex-direction: column; gap: 12px; }
#sidebar h1 { font-size: 18px; color: #fff; }
#sidebar input, #sidebar select { width: 100%; padding: 8px 10px; border-radius: 6px; border: 1px solid #444; background: #1e1e1e; color: #e0e0e0; font-size: 13px; }
#sidebar input:focus, #sidebar select:focus { outline: none; border-color: #7c5cfc; }
#details { font-size: 13px; line-height: 1.6; }
#details .prop { color: #aaa; }
#details .val { color: #fff; word-break: break-all; }
#details h3 { margin-top: 8px; color: #7c5cfc; font-size: 14px; }
#graph { flex: 1; }
.stats { font-size: 12px; color: #888; }
</style>
</head>
<body>
<div id="sidebar">
  <h1>🦛 Hippocamp</h1>
  <input id="search" type="text" placeholder="Search nodes...">
  <select id="typeFilter"><option value="">All types</option></select>
  <div class="stats" id="stats"></div>
  <div id="details"><p style="color:#666">Click a node to inspect</p></div>
</div>
<div id="graph"></div>
<script>
const rawNodes = ` + nodesJSON + `;
const rawEdges = ` + edgesJSON + `;

const typeColors = {
  'Topic':'#4a9eff','Entity':'#34d399','Note':'#fbbf24','Source':'#a78bfa',
  'Decision':'#f97316','Question':'#f472b6','Tag':'#94a3b8',
  'Project':'#22d3ee','Module':'#6ee7b7','File':'#60a5fa',
  'Symbol':'#c084fc','Function':'#c084fc','Struct':'#c084fc',
  'Interface':'#c084fc','Class':'#c084fc','Dependency':'#fb923c','Concept':'#e879f9',
  '':'#666'
};

const nodes = new vis.DataSet(rawNodes.map(n => ({
  id: n.id, label: n.label, group: n.group,
  value: n.value,
  color: { background: typeColors[n.group]||'#666', border: '#222',
           highlight: { background: '#fff', border: typeColors[n.group]||'#666' }},
  font: { color: '#e0e0e0', size: 12 }
})));

const edges = new vis.DataSet(rawEdges.map((e,i) => ({
  id: i, from: e.from, to: e.to, label: e.label,
  arrows: 'to', color: { color: '#555', highlight: '#999' },
  font: { color: '#888', size: 10, strokeWidth: 0 }
})));

const container = document.getElementById('graph');
const network = new vis.Network(container, { nodes, edges }, {
  physics: { solver: 'forceAtlas2Based', forceAtlas2Based: { gravitationalConstant: -60, springLength: 120 }, stabilization: { iterations: 200 }},
  interaction: { hover: true, tooltipDelay: 100 },
  scaling: { min: 10, max: 40 }
});

// Stats
const types = {};
rawNodes.forEach(n => { types[n.group||'(untyped)'] = (types[n.group||'(untyped)']||0)+1; });
document.getElementById('stats').textContent = rawNodes.length + ' nodes, ' + rawEdges.length + ' edges';

// Type filter
const sel = document.getElementById('typeFilter');
Object.keys(types).sort().forEach(t => {
  const o = document.createElement('option');
  o.value = t; o.textContent = t + ' (' + types[t] + ')';
  sel.appendChild(o);
});

sel.onchange = function() {
  const v = this.value;
  nodes.forEach(n => {
    const orig = rawNodes.find(r => r.id === n.id);
    nodes.update({ id: n.id, hidden: v && orig.group !== v });
  });
};

// Search
document.getElementById('search').oninput = function() {
  const q = this.value.toLowerCase();
  nodes.forEach(n => {
    const match = !q || n.label.toLowerCase().includes(q);
    nodes.update({ id: n.id, opacity: match ? 1 : 0.15 });
  });
};

// Click to inspect
const allTriples = rawEdges;
network.on('click', function(params) {
  const det = document.getElementById('details');
  if (!params.nodes.length) { det.innerHTML = '<p style="color:#666">Click a node to inspect</p>'; return; }
  const nid = params.nodes[0];
  const n = rawNodes.find(r => r.id === nid);
  let h = '<h3>' + esc(n.label) + '</h3>';
  h += '<div class="prop">Type: <span class="val">' + esc(n.group||'—') + '</span></div>';
  h += '<div class="prop">URI: <span class="val">' + esc(n.id) + '</span></div>';
  h += '<div class="prop">Degree: <span class="val">' + n.value + '</span></div>';
  const outE = allTriples.filter(e => e.from === nid);
  const inE = allTriples.filter(e => e.to === nid);
  if (outE.length) {
    h += '<h3>Outgoing (' + outE.length + ')</h3>';
    outE.forEach(e => { const t=rawNodes.find(r=>r.id===e.to); h += '<div class="prop">→ '+esc(e.label)+' → <span class="val">'+esc(t?t.label:e.to)+'</span></div>'; });
  }
  if (inE.length) {
    h += '<h3>Incoming (' + inE.length + ')</h3>';
    inE.forEach(e => { const t=rawNodes.find(r=>r.id===e.from); h += '<div class="prop">← '+esc(e.label)+' ← <span class="val">'+esc(t?t.label:e.from)+'</span></div>'; });
  }
  det.innerHTML = h;
});

function esc(s) { const d=document.createElement('div'); d.textContent=s; return d.innerHTML; }
</script>
</body>
</html>`
}

// ── shared utilities ────────────────────────────────────────

func localName(uri string) string {
	if i := strings.LastIndex(uri, "#"); i >= 0 {
		return uri[i+1:]
	}
	if i := strings.LastIndex(uri, "/"); i >= 0 {
		return uri[i+1:]
	}
	return uri
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func hasOverlap(a, b []string) bool {
	set := make(map[string]bool, len(a))
	for _, v := range a {
		set[v] = true
	}
	for _, v := range b {
		if set[v] {
			return true
		}
	}
	return false
}

func setToSlice(m map[string]bool) []string {
	s := make([]string, 0, len(m))
	for k := range m {
		s = append(s, k)
	}
	sort.Strings(s)
	return s
}

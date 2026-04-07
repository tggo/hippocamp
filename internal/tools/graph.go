package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func graphTool() mcp.Tool {
	return mcp.NewTool("graph",
		mcp.WithDescription(`Manage RDF named graphs, persistence, and namespace prefixes.

Graph actions: create, delete, list, stats, clear, load, dump, summary, migrate
Prefix actions: prefix_add, prefix_list, prefix_remove

Examples:
  {"action":"list"}
  {"action":"create","name":"http://example.org/myGraph"}
  {"action":"stats","name":"http://example.org/myGraph"}
  {"action":"summary"}  — compact overview: type counts, top entities, recent decisions, topics (~500 tokens)
  {"action":"migrate"}  — apply pending schema migrations (add provenance defaults, etc.)
  {"action":"dump","file":"./backup.trig"}
  {"action":"load","file":"./backup.trig"}
  {"action":"import","data":"@prefix ex: ... ex:Alice ex:knows ex:Bob ."}
  {"action":"prefix_add","prefix":"ex","uri":"http://example.org/"}`),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation: create|delete|list|stats|clear|load|dump|import|summary|migrate|prefix_add|prefix_list|prefix_remove"),
			mcp.Enum("create", "delete", "list", "stats", "clear", "load", "dump", "import", "summary", "migrate",
				"prefix_add", "prefix_list", "prefix_remove"),
		),
		mcp.WithString("name",
			mcp.Description("Graph URI (for create/delete/stats/clear; omit for default graph)"),
		),
		mcp.WithString("file",
			mcp.Description("File path for load/dump operations"),
		),
		mcp.WithString("data",
			mcp.Description("TriG/Turtle data string for import action (bulk load without writing to file)"),
		),
		mcp.WithString("format",
			mcp.Description("Serialization format: trig (default) | turtle | nt | nq"),
			mcp.Enum("trig", "turtle", "nt", "nq"),
		),
		mcp.WithString("prefix",
			mcp.Description("Namespace prefix (for prefix_add/prefix_remove)"),
		),
		mcp.WithString("uri",
			mcp.Description("Namespace URI (for prefix_add)"),
		),
	)
}

func graphHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: action"), nil
		}

		switch action {
		case "list":
			return handleGraphList(store)
		case "create":
			return handleGraphCreate(store, req)
		case "delete":
			return handleGraphDelete(store, req)
		case "stats":
			return handleGraphStats(store, req)
		case "clear":
			return handleGraphClear(store, req)
		case "dump":
			return handleGraphDump(store, req)
		case "load":
			return handleGraphLoad(store, req)
		case "import":
			return handleGraphImport(store, req)
		case "summary":
			return handleGraphSummary(store)
		case "migrate":
			return handleGraphMigrate(store)
		case "prefix_add":
			return handlePrefixAdd(store, req)
		case "prefix_list":
			return handlePrefixList(store)
		case "prefix_remove":
			return handlePrefixRemove(store, req)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q", action)), nil
		}
	}
}

func handleGraphList(store *rdfstore.Store) (*mcp.CallToolResult, error) {
	graphs := store.ListGraphs()
	data, _ := json.Marshal(graphs)
	return mcp.NewToolResultText(string(data)), nil
}

func handleGraphCreate(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("create requires: name (graph URI)"), nil
	}
	if err := store.CreateGraph(name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("created graph %q", name)), nil
}

func handleGraphDelete(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("delete requires: name (graph URI)"), nil
	}
	if err := store.DeleteGraph(name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("deleted graph %q", name)), nil
}

func handleGraphStats(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	stats := store.Stats(name)
	data, _ := json.Marshal(stats)
	return mcp.NewToolResultText(string(data)), nil
}

func handleGraphClear(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := req.GetString("name", "")
	if err := store.ClearGraph(name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func handleGraphDump(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("file", "")
	if path == "" {
		return mcp.NewToolResultError("dump requires: file (path)"), nil
	}
	if err := rdfstore.Save(store, path); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("saved to %q", path)), nil
}

func handleGraphLoad(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := req.GetString("file", "")
	if path == "" {
		return mcp.NewToolResultError("load requires: file (path)"), nil
	}
	if err := rdfstore.Load(store, path); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("loaded from %q", path)), nil
}

func handlePrefixAdd(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefix := req.GetString("prefix", "")
	uri := req.GetString("uri", "")
	if prefix == "" || uri == "" {
		return mcp.NewToolResultError("prefix_add requires: prefix, uri"), nil
	}
	store.BindPrefix(prefix, uri)
	return mcp.NewToolResultText(fmt.Sprintf("bound %q → %q", prefix, uri)), nil
}

func handlePrefixList(store *rdfstore.Store) (*mcp.CallToolResult, error) {
	prefixes := store.ListPrefixes()
	data, _ := json.Marshal(prefixes)
	return mcp.NewToolResultText(string(data)), nil
}

func handlePrefixRemove(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	prefix := req.GetString("prefix", "")
	if prefix == "" {
		return mcp.NewToolResultError("prefix_remove requires: prefix"), nil
	}
	store.RemovePrefix(prefix)
	return mcp.NewToolResultText(fmt.Sprintf("removed prefix %q", prefix)), nil
}

func handleGraphImport(store *rdfstore.Store, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	data := req.GetString("data", "")
	if data == "" {
		return mcp.NewToolResultError("import requires: data (TriG/Turtle string)"), nil
	}
	targetGraph := req.GetString("name", "")

	count, err := store.Import(data, targetGraph)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("import error: %v", err)), nil
	}

	if targetGraph != "" {
		return mcp.NewToolResultText(fmt.Sprintf("imported %d triples into %s", count, targetGraph)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("imported %d triples", count)), nil
}

// handleGraphSummary returns a compact overview of the entire knowledge graph
// for LLM "wake-up" context: type counts, top entities by degree, recent decisions, topics.
func handleGraphSummary(store *rdfstore.Store) (*mcp.CallToolResult, error) {
	graphNames := store.ListGraphs()

	// Collect all triples across all graphs.
	typeCounts := map[string]int{}   // hippo type local name → count
	labels := map[string]string{}    // subject → label
	degree := map[string]int{}       // subject → in+out degree (relationship triples only)
	topics := map[string]string{}    // topic URI → label
	decisions := []map[string]string{} // recent decisions
	totalTriples := 0
	invalidated := 0

	for _, gn := range graphNames {
		triples, err := store.ListTriples(gn, "", "", "")
		if err != nil {
			continue
		}
		totalTriples += len(triples)

		for _, t := range triples {
			switch t.Predicate {
			case rdfType:
				localName := t.Object
				if idx := len(hippoNS); len(t.Object) > idx && t.Object[:idx] == hippoNS {
					localName = t.Object[idx:]
				}
				typeCounts[localName]++
			case rdfsLabel:
				labels[t.Subject] = t.Object
			case hippoValidTo:
				invalidated++
			}

			// Count degree for relationship predicates only.
			if !isMetaPredicate(t.Predicate) {
				degree[t.Subject]++
				if t.ObjType == "uri" {
					degree[t.Object]++
				}
			}
		}
	}

	// Collect topic labels.
	for _, gn := range graphNames {
		triples, _ := store.ListTriples(gn, "", rdfType, hippoNS+"Topic")
		for _, t := range triples {
			if lbl, ok := labels[t.Subject]; ok {
				topics[t.Subject] = lbl
			} else {
				topics[t.Subject] = t.Subject
			}
		}
	}

	// Collect decisions (up to 5 most recent).
	for _, gn := range graphNames {
		triples, _ := store.ListTriples(gn, "", rdfType, hippoNS+"Decision")
		for _, t := range triples {
			d := map[string]string{"uri": t.Subject}
			if lbl, ok := labels[t.Subject]; ok {
				d["label"] = lbl
			}
			decisions = append(decisions, d)
		}
	}
	if len(decisions) > 5 {
		decisions = decisions[len(decisions)-5:]
	}

	// Top entities by degree (up to 10).
	type entityDegree struct {
		URI    string `json:"uri"`
		Label  string `json:"label,omitempty"`
		Degree int    `json:"degree"`
	}
	var topEntities []entityDegree
	for uri, deg := range degree {
		topEntities = append(topEntities, entityDegree{URI: uri, Label: labels[uri], Degree: deg})
	}
	sort.Slice(topEntities, func(i, j int) bool {
		return topEntities[i].Degree > topEntities[j].Degree
	})
	if len(topEntities) > 10 {
		topEntities = topEntities[:10]
	}

	// Topic list.
	var topicList []string
	for _, lbl := range topics {
		topicList = append(topicList, lbl)
	}
	sort.Strings(topicList)

	summary := map[string]any{
		"graphs":        len(graphNames),
		"total_triples": totalTriples,
		"invalidated":   invalidated,
		"type_counts":   typeCounts,
		"topics":        topicList,
		"top_entities":  topEntities,
		"decisions":     decisions,
	}

	data, _ := json.Marshal(summary)
	return mcp.NewToolResultText(string(data)), nil
}

// isMetaPredicate checks if a predicate is a metadata/annotation predicate
// (not a relationship between resources).
func isMetaPredicate(pred string) bool {
	switch pred {
	case rdfType, rdfsLabel,
		hippoNS + "summary", hippoNS + "content", hippoNS + "alias",
		hippoNS + "status", hippoNS + "createdAt", hippoNS + "updatedAt",
		hippoNS + "url", hippoNS + "filePath", hippoNS + "signature",
		hippoNS + "lineNumber", hippoNS + "rationale", hippoNS + "version",
		hippoNS + "language", hippoNS + "rootPath", hippoNS + "confidence",
		hippoNS + "provenance", hippoNS + "source", hippoNS + "revision",
		hippoNS + "validFrom", hippoNS + "validTo":
		return true
	}
	return false
}

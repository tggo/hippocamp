package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func sparqlTool() mcp.Tool {
	return mcp.NewTool("sparql",
		mcp.WithDescription(`Execute SPARQL queries or updates against the RDF store.

Examples:
  SELECT: {"query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"}
  ASK:    {"query": "ASK { <http://ex.org/Alice> a <http://ex.org/Person> }"}
  UPDATE: {"query": "INSERT DATA { <http://ex.org/Alice> <http://ex.org/name> \"Alice\" }"}
  Named graph: {"query": "SELECT ?s WHERE { ?s ?p ?o }", "graph": "http://ex.org/g1"}

Returns JSON bindings for SELECT, "true"/"false" for ASK, "ok" for updates.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("SPARQL SELECT, ASK, CONSTRUCT, or UPDATE string"),
		),
		mcp.WithString("graph",
			mcp.Description("Named graph URI to query (omit for default graph)"),
		),
	)
}

func sparqlHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		query, err := req.RequireString("query")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: query"), nil
		}
		graphName := req.GetString("graph", "")

		if isUpdate(query) {
			return handleSPARQLUpdate(store, graphName, query)
		}
		return handleSPARQLQuery(store, graphName, query)
	}
}

// isUpdate detects SPARQL Update operations by checking for UPDATE keywords.
func isUpdate(query string) bool {
	upper := strings.ToUpper(strings.TrimSpace(query))
	for _, kw := range []string{"INSERT", "DELETE", "LOAD", "CLEAR", "DROP", "CREATE", "COPY", "MOVE", "ADD"} {
		if strings.HasPrefix(upper, kw) {
			return true
		}
	}
	return false
}

func handleSPARQLQuery(store *rdfstore.Store, graphName, query string) (*mcp.CallToolResult, error) {
	result, err := store.SPARQLQuery(graphName, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SPARQL query error: %v", err)), nil
	}

	// ASK result
	if result.Vars == nil && result.Bindings == nil {
		return mcp.NewToolResultText(fmt.Sprintf("%v", result.AskResult)), nil
	}

	// SELECT result — convert Term values to strings
	rows := make([]map[string]string, 0, len(result.Bindings))
	for _, binding := range result.Bindings {
		row := make(map[string]string, len(binding))
		for k, v := range binding {
			if v != nil {
				row[k] = v.String()
			}
		}
		rows = append(rows, row)
	}

	data, err := json.Marshal(rows)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func handleSPARQLUpdate(store *rdfstore.Store, defaultGraph, update string) (*mcp.CallToolResult, error) {
	if err := store.SPARQLUpdate(defaultGraph, update); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("SPARQL update error: %v", err)), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

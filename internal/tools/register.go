// Package tools registers all MCP tools and provides handler lookup for testing.
package tools

import (
	"context"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

type handlerFunc = func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)

// handlers holds named tool handlers for test access.
var handlers = map[string]func(*rdfstore.Store) handlerFunc{
	"triple": tripleHandlerFactory,
	"sparql": sparqlHandlerFactory,
	"graph":  graphHandlerFactory,
	"search":   searchHandlerFactory,
	"validate": validateHandlerFactory,
}

// Register adds all MCP tools to the server.
func Register(s *server.MCPServer, store *rdfstore.Store) {
	s.AddTool(tripleTool(), tripleHandlerFactory(store))
	s.AddTool(sparqlTool(), sparqlHandlerFactory(store))
	s.AddTool(graphTool(), graphHandlerFactory(store))
	s.AddTool(searchTool(), searchHandlerFactory(store))
	s.AddTool(validateTool(), validateHandlerFactory(store))
}

// HandlerFor returns a handler bound to the given store, for testing only.
func HandlerFor(store *rdfstore.Store, toolName string) handlerFunc {
	factory, ok := handlers[toolName]
	if !ok {
		return nil
	}
	return factory(store)
}

// ResultText extracts the text content from a CallToolResult.
func ResultText(r *mcp.CallToolResult) string {
	if r == nil || len(r.Content) == 0 {
		return ""
	}
	var parts []string
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "")
}

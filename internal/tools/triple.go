package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func tripleTool() mcp.Tool {
	return mcp.NewTool("triple",
		mcp.WithDescription(`Manage RDF triples. Actions: add, remove, list (with optional wildcards).

Examples:
  add:    {"action":"add","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"Alice","object_type":"literal"}
  remove: {"action":"remove","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"http://ex.org/Bob"}
  list:   {"action":"list","subject":"http://ex.org/Alice"}  (empty fields = wildcard)`),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation: add | remove | list"),
			mcp.Enum("add", "remove", "list"),
		),
		mcp.WithString("graph",
			mcp.Description("Named graph URI (omit for default graph)"),
		),
		mcp.WithString("subject",
			mcp.Description("Subject URI (required for add/remove, wildcard if empty for list)"),
		),
		mcp.WithString("predicate",
			mcp.Description("Predicate URI (required for add/remove, wildcard if empty for list)"),
		),
		mcp.WithString("object",
			mcp.Description("Object value (required for add/remove, wildcard if empty for list)"),
		),
		mcp.WithString("object_type",
			mcp.Description("Object type: uri (default) | literal | bnode"),
			mcp.Enum("uri", "literal", "bnode"),
		),
		mcp.WithString("lang",
			mcp.Description("Language tag for literal objects (e.g. 'en')"),
		),
		mcp.WithString("datatype",
			mcp.Description("XSD datatype URI for typed literals (e.g. 'http://www.w3.org/2001/XMLSchema#integer')"),
		),
	)
}

func tripleHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		action, err := req.RequireString("action")
		if err != nil {
			return mcp.NewToolResultError("missing required parameter: action"), nil
		}

		graphName := req.GetString("graph", "")

		switch action {
		case "add":
			return handleTripleAdd(store, req, graphName)
		case "remove":
			return handleTripleRemove(store, req, graphName)
		case "list":
			return handleTripleList(store, req, graphName)
		default:
			return mcp.NewToolResultError(fmt.Sprintf("unknown action %q", action)), nil
		}
	}
}

func handleTripleAdd(store *rdfstore.Store, req mcp.CallToolRequest, graphName string) (*mcp.CallToolResult, error) {
	subject := req.GetString("subject", "")
	predicate := req.GetString("predicate", "")
	object := req.GetString("object", "")

	if subject == "" || predicate == "" || object == "" {
		return mcp.NewToolResultError("add requires: subject, predicate, object"), nil
	}

	objType := req.GetString("object_type", "uri")
	lang := req.GetString("lang", "")
	datatype := req.GetString("datatype", "")

	if err := store.AddTriple(graphName, subject, predicate, object, objType, lang, datatype); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func handleTripleRemove(store *rdfstore.Store, req mcp.CallToolRequest, graphName string) (*mcp.CallToolResult, error) {
	subject := req.GetString("subject", "")
	predicate := req.GetString("predicate", "")
	object := req.GetString("object", "")

	if subject == "" || predicate == "" || object == "" {
		return mcp.NewToolResultError("remove requires: subject, predicate, object"), nil
	}

	if err := store.RemoveTriple(graphName, subject, predicate, object); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func handleTripleList(store *rdfstore.Store, req mcp.CallToolRequest, graphName string) (*mcp.CallToolResult, error) {
	subject := req.GetString("subject", "")
	predicate := req.GetString("predicate", "")
	object := req.GetString("object", "")

	triples, err := store.ListTriples(graphName, subject, predicate, object)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	data, err := json.Marshal(triples)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

const (
	hippoValidFrom = hippoNS + "validFrom"
	hippoValidTo   = hippoNS + "validTo"
)

func tripleTool() mcp.Tool {
	return mcp.NewTool("triple",
		mcp.WithDescription(`Manage RDF triples. Actions: add, remove, list, invalidate.

Examples:
  add:        {"action":"add","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"Alice","object_type":"literal"}
  remove:     {"action":"remove","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"http://ex.org/Bob"}
  list:       {"action":"list","subject":"http://ex.org/Alice"}  (empty fields = wildcard)
  invalidate: {"action":"invalidate","subject":"http://ex.org/Alice","predicate":"http://ex.org/worksAt","object":"http://ex.org/OldCo"}
              Sets hippo:validTo=now on the subject, marking the fact as no longer current. The triple stays in the graph for historical queries.`),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation: add | remove | list | invalidate"),
			mcp.Enum("add", "remove", "list", "invalidate"),
		),
		mcp.WithString("graph",
			mcp.Description("Named graph URI (omit for default graph)"),
		),
		mcp.WithString("subject",
			mcp.Description("Subject URI (required for add/remove/invalidate, wildcard if empty for list)"),
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
		case "invalidate":
			return handleTripleInvalidate(store, req, graphName)
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

	// Duplicate detection: check if exact S/P/O already exists.
	existing, _ := store.ListTriples(graphName, subject, predicate, "")
	for _, t := range existing {
		if t.Object == object {
			return mcp.NewToolResultText(fmt.Sprintf("duplicate: triple already exists in graph %q", graphName)), nil
		}
	}

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

// handleTripleInvalidate marks a fact as no longer current by setting hippo:validTo = now.
// The original triple stays in the graph for historical queries.
func handleTripleInvalidate(store *rdfstore.Store, req mcp.CallToolRequest, graphName string) (*mcp.CallToolResult, error) {
	subject := req.GetString("subject", "")

	if subject == "" {
		return mcp.NewToolResultError("invalidate requires: subject"), nil
	}

	// Check that the subject exists.
	triples, _ := store.ListTriples(graphName, subject, "", "")
	if len(triples) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("subject <%s> not found in graph %q", subject, graphName)), nil
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Set hippo:validTo on the subject (remove old value first if exists).
	_ = store.RemoveTriple(graphName, subject, hippoValidTo, "")
	if err := store.AddTriple(graphName, subject, hippoValidTo, now, "literal", "", "http://www.w3.org/2001/XMLSchema#dateTime"); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Also set hippo:status to "invalidated" if not already set.
	statusTriples, _ := store.ListTriples(graphName, subject, hippoStatus_, "")
	if len(statusTriples) == 0 {
		_ = store.AddTriple(graphName, subject, hippoStatus_, "invalidated", "literal", "", "")
	}

	result := map[string]string{
		"status":   "invalidated",
		"subject":  subject,
		"validTo":  now,
		"message":  "fact marked as no longer current; triple preserved for history",
	}
	data, _ := json.Marshal(result)
	return mcp.NewToolResultText(string(data)), nil
}

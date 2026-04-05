package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func graphTool() mcp.Tool {
	return mcp.NewTool("graph",
		mcp.WithDescription(`Manage RDF named graphs, persistence, and namespace prefixes.

Graph actions: create, delete, list, stats, clear, load, dump
Prefix actions: prefix_add, prefix_list, prefix_remove

Examples:
  {"action":"list"}
  {"action":"create","name":"http://example.org/myGraph"}
  {"action":"stats","name":"http://example.org/myGraph"}
  {"action":"dump","file":"./backup.trig"}
  {"action":"load","file":"./backup.trig"}
  {"action":"import","data":"@prefix ex: ... ex:Alice ex:knows ex:Bob ."}
  {"action":"prefix_add","prefix":"ex","uri":"http://example.org/"}`),
		mcp.WithString("action",
			mcp.Required(),
			mcp.Description("Operation: create|delete|list|stats|clear|load|dump|import|prefix_add|prefix_list|prefix_remove"),
			mcp.Enum("create", "delete", "list", "stats", "clear", "load", "dump", "import",
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

	count, err := store.Import(data)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("import error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("imported %d triples", count)), nil
}

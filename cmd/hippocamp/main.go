package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ruslanmv/hippocamp/internal/config"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/setup"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

// Injected at build time via ldflags.
var (
	version   = "dev"
	buildTime = "" // RFC 3339 timestamp
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	queryStr := flag.String("query", "", "one-shot search: query the persisted graph and exit")
	queryType := flag.String("type", "", "filter search results by rdf:type URI")
	queryScope := flag.String("scope", "", "named graph to search in")
	queryLimit := flag.Int("limit", 20, "max search results")
	flag.Parse()

	// Auto-setup Claude Code integration files (hooks, skills).
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		if setupErr := setup.Setup(cwd, buildTime); setupErr != nil {
			fmt.Fprintf(os.Stderr, "hippocamp: setup: %v\n", setupErr)
		}
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hippocamp: config error: %v\n", err)
		os.Exit(1)
	}

	store := rdfstore.NewStore()

	// Register user-defined prefixes from config.
	for prefix, uri := range cfg.Prefixes {
		store.BindPrefix(prefix, uri)
	}

	// Auto-load the default graph file on startup.
	if cfg.Store.AutoLoad {
		loaded, err := rdfstore.AutoLoad(store, cfg.Store.DefaultFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hippocamp: auto-load error: %v\n", err)
			os.Exit(1)
		}
		if loaded {
			fmt.Fprintf(os.Stderr, "hippocamp: loaded graph from %s\n", cfg.Store.DefaultFile)
		}
	}

	// One-shot query mode: search the graph and exit.
	if *queryStr != "" {
		handler := tools.HandlerFor(store, "search")
		args := map[string]any{"query": *queryStr}
		if *queryType != "" {
			args["type"] = *queryType
		}
		if *queryScope != "" {
			args["scope"] = *queryScope
		}
		if *queryLimit != 20 {
			args["limit"] = float64(*queryLimit)
		}
		req := mcp.CallToolRequest{}
		req.Params.Arguments = args
		result, err := handler(context.Background(), req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hippocamp: query error: %v\n", err)
			store.Close()
			os.Exit(1)
		}
		text := tools.ResultText(result)
		// Pretty-print JSON output.
		var parsed any
		if json.Unmarshal([]byte(text), &parsed) == nil {
			pretty, _ := json.MarshalIndent(parsed, "", "  ")
			fmt.Println(string(pretty))
		} else {
			fmt.Println(text)
		}
		store.Close()
		return
	}

	// Register signal handler for graceful shutdown (auto-save on exit).
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if store.IsDirty() {
			if err := rdfstore.Save(store, cfg.Store.DefaultFile); err != nil {
				fmt.Fprintf(os.Stderr, "hippocamp: auto-save error: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "hippocamp: saved graph to %s\n", cfg.Store.DefaultFile)
			}
		}
		store.Close()
		os.Exit(0)
	}()

	// Create MCP server and register tools.
	s := server.NewMCPServer(
		"hippocamp",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)
	tools.Register(s, store)

	// Serve over stdio (compatible with Claude Code, Desktop, IDE extensions).
	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "hippocamp: server error: %v\n", err)
		os.Exit(1)
	}
}

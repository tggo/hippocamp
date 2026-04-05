package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mark3labs/mcp-go/server"
	"github.com/ruslanmv/hippocamp/internal/config"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

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
		"0.1.0",
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

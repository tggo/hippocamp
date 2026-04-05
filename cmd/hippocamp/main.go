package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/ruslanmv/hippocamp/internal/analytics"
	"github.com/ruslanmv/hippocamp/internal/config"
	"github.com/ruslanmv/hippocamp/internal/healthcheck"
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
	// Handle subcommands before flag parsing.
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "install":
			runInstall()
			return
		case "uninstall":
			runUninstall()
			return
		case "version":
			fmt.Printf("hippocamp %s\n", version)
			return
		}
	}

	// Dev builds: log to /tmp/hippocamp-<timestamp>.log
	if version == "dev" {
		logName := fmt.Sprintf("/tmp/hippocamp-%s.log", time.Now().Format("2006-01-02_15-04-05"))
		if lf, err := os.OpenFile(logName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			log.SetOutput(lf)
			log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
			log.Printf("hippocamp dev started, pid=%d, cwd=%s", os.Getpid(), func() string { d, _ := os.Getwd(); return d }())
			// Also redirect stderr to log file so panics are captured.
			os.Stderr = lf
		}
	}

	configPath := flag.String("config", "config.yaml", "path to config file")
	queryStr := flag.String("query", "", "one-shot search: query the persisted graph and exit")
	queryType := flag.String("type", "", "filter search results by rdf:type URI")
	queryScope := flag.String("scope", "", "named graph to search in")
	queryLimit := flag.Int("limit", 20, "max search results")
	flag.Parse()

	// Auto-setup Claude Code integration files (hooks, skills).
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		if setupErr := setup.Setup(cwd, buildTime); setupErr != nil {
			log.Printf("setup error: %v", setupErr)
		} else {
			log.Printf("setup complete")
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
			log.Printf("loaded graph from %s", cfg.Store.DefaultFile)
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

	// Periodic auto-save: flush dirty graph to disk every 30 seconds.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if store.IsDirty() {
				if err := rdfstore.Save(store, cfg.Store.DefaultFile); err != nil {
					log.Printf("auto-save error: %v", err)
				} else {
					log.Printf("auto-saved graph to %s", cfg.Store.DefaultFile)
				}
			}
		}
	}()

	// Register signal handler for graceful shutdown (auto-save on exit).
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if store.IsDirty() {
			if err := rdfstore.Save(store, cfg.Store.DefaultFile); err != nil {
				log.Printf("shutdown save error: %v", err)
			} else {
				log.Printf("saved graph to %s", cfg.Store.DefaultFile)
			}
		}
		store.Close()
		os.Exit(0)
	}()

	// Start background health checker (scans every 30s when graph is dirty).
	checker := healthcheck.New(store, 30*time.Second)

	// Set up analytics collector for tool call tracking.
	collector := analytics.New(store)
	hooks := &server.Hooks{}
	hooks.AddBeforeCallTool(collector.BeforeCallTool)
	hooks.AddAfterCallTool(func(ctx context.Context, id any, req *mcp.CallToolRequest, result any) {
		collector.AfterCallTool(ctx, id, req, result)
		// Mark healthcheck dirty after mutations.
		if tools.IsMutatingCall(req.Params.Name, req.GetArguments()) {
			checker.MarkDirty()
		}
	})

	// Create MCP server and register tools.
	s := server.NewMCPServer(
		"hippocamp",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithHooks(hooks),
	)
	tools.Register(s, store)
	tools.SetHealthChecker(checker)

	log.Printf("MCP server starting (version=%s, tools=triple,sparql,graph,search,validate)", version)

	// Serve over stdio (compatible with Claude Code, Desktop, IDE extensions).
	if err := server.ServeStdio(s); err != nil {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
	log.Printf("server stopped cleanly")
}

// runInstall adds hippocamp as an MCP server to Claude Code via `claude mcp add`.
func runInstall() {
	// Use just "hippocamp" so it resolves via PATH at runtime.
	// This avoids hardcoding versioned Caskroom paths that break on brew upgrade.
	binPath := "hippocamp"

	// Check if claude CLI is available.
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Println("Claude CLI not found. Add hippocamp manually to your MCP config:")
		fmt.Println()
		printManualConfig(binPath)
		os.Exit(1)
	}

	// Run: claude mcp add hippocamp <binary-path>
	cmd := exec.Command(claudePath, "mcp", "add", "hippocamp", binPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	fmt.Printf("Adding hippocamp to Claude Code...\n")
	fmt.Printf("  command: %s\n", binPath)
	fmt.Println()

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nclaude mcp add failed: %v\n", err)
		fmt.Println("\nAdd manually instead:")
		printManualConfig(binPath)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("Done! Hippocamp is now available in Claude Code.")
	fmt.Println("Hooks and skills will auto-install on first run.")
}

// runUninstall removes all hippocamp files from the current project and Claude Code config.
func runUninstall() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hippocamp: cannot get working directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Uninstalling hippocamp from this project...")
	fmt.Println()

	// Remove .claude/skills/project-analyze.md
	removed := 0
	for _, rel := range []string{
		".claude/skills/project-analyze.md",
		".claude/hooks/hippocamp-pre-query.sh",
		".claude/hooks/hippocamp-post-edit.sh",
		".claude/.hippocamp-stale",
	} {
		path := filepath.Join(cwd, rel)
		if err := os.Remove(path); err == nil {
			fmt.Printf("  removed %s\n", rel)
			removed++
		}
	}

	// Remove data directory.
	dataDir := filepath.Join(cwd, "data")
	if info, err := os.Stat(dataDir); err == nil && info.IsDir() {
		if err := os.RemoveAll(dataDir); err == nil {
			fmt.Println("  removed data/")
			removed++
		}
	}

	// Clean up empty .claude subdirectories.
	for _, dir := range []string{".claude/skills", ".claude/hooks"} {
		dirPath := filepath.Join(cwd, dir)
		entries, err := os.ReadDir(dirPath)
		if err == nil && len(entries) == 0 {
			os.Remove(dirPath)
		}
	}

	if removed == 0 {
		fmt.Println("  nothing to remove (already clean)")
	}

	// Remove from Claude Code config.
	fmt.Println()
	claudePath, err := exec.LookPath("claude")
	if err == nil {
		cmd := exec.Command(claudePath, "mcp", "remove", "hippocamp")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			fmt.Println("Removed hippocamp from Claude Code config.")
		}
	} else {
		fmt.Println("Claude CLI not found — remove hippocamp from ~/.claude/settings.json manually.")
	}

	fmt.Println()
	fmt.Println("Done. To also remove the binary: brew uninstall hippocamp")
}


func printManualConfig(binPath string) {
	fmt.Println("Add to ~/.claude/settings.json:")
	fmt.Println()
	fmt.Printf("  \"mcpServers\": {\n")
	fmt.Printf("    \"hippocamp\": {\n")
	fmt.Printf("      \"command\": %q\n", binPath)
	fmt.Printf("    }\n")
	fmt.Printf("  }\n")
}

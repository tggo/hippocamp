# Hippocamp — RDF MCP Server

## Project overview

An MCP server that exposes an in-memory RDF knowledge graph to LLMs via three tools: `triple`, `sparql`, `graph`. Built in Go using `mark3labs/mcp-go` for the MCP protocol and `tggo/goRDFlib` for RDF graph operations (SPARQL 1.1, named graphs, TriG serialization).

## Commands

```bash
# Run all tests
go test ./...

# Build binary
go build -o hippocamp ./cmd/hippocamp/

# Run server (stdio transport)
./hippocamp --config config.yaml

# Tidy dependencies
go mod tidy
```

## Architecture

```
cmd/hippocamp/main.go          — entry point: config load, store init, signal handler, ServeStdio
internal/config/config.go      — YAML + ENV config loading
internal/rdfstore/store.go     — Store struct: wraps graph.Dataset (BadgerDB in-memory), dirty tracking
internal/rdfstore/persistence.go — Save/Load/AutoLoad (TriG format via trig.SerializeDataset/ParseDataset)
internal/tools/register.go     — MCP tool registration + HandlerFor() test helper
internal/tools/triple.go       — triple tool: add / remove / list
internal/tools/sparql.go       — sparql tool: SELECT / ASK / UPDATE (auto-detected)
internal/tools/graph.go        — graph tool: create/delete/list/stats/clear/dump/load/prefix_*
```

## Key design decisions

### BadgerDB as backing store (not MemoryStore)
`goRDFlib`'s default `MemoryStore` is **not context-aware** — it ignores the graph identifier and stores all triples in a flat index. Named graphs require a context-aware store. We use `badgerstore.New(badgerstore.WithInMemory())` which:
- Creates a fully isolated BadgerDB instance per `NewStore()` call
- Is context-aware (`ContextAware() = true`)
- Supports ACID transactions
- Is pure Go (no CGO)

`SQLiteStore` was tried first but its `WithInMemory()` uses a shared SQLite DSN (`file::memory:?mode=memory&cache=shared`), so all instances share state.

### TriG format for persistence
TriG extends Turtle with named graph blocks. It's the only text format that preserves named graph boundaries. `trig.SerializeDataset` / `trig.ParseDataset` operate directly on `*graph.Dataset`.

### Store closes BadgerDB
`Store.Close()` must be called on shutdown to release BadgerDB resources. The signal handler in `main.go` calls it before `os.Exit`.

### Tool grouping
- **`triple`** and **`sparql`** are separate tools (frequent use, rich descriptions with examples)
- **`graph`** groups infrequent operations: graph lifecycle, persistence, and prefix management

### SPARQL update detection
`isUpdate()` in `sparql.go` checks the first keyword of the query string (INSERT, DELETE, LOAD, etc.) to distinguish updates from queries. Updates go through `store.SPARQLUpdate()` which builds a `sparql.Dataset` struct from the store's named graphs.

## Configuration

```yaml
# config.yaml
store:
  default_file: "./data/default.trig"  # auto-loaded on startup, auto-saved on SIGINT/SIGTERM
  auto_load: true
  format: "trig"
prefixes:
  ex: "http://example.org/"
```

ENV overrides: `HIPPOCAMP_STORE_DEFAULT_FILE`, `HIPPOCAMP_STORE_AUTO_LOAD`, `HIPPOCAMP_STORE_FORMAT`

## Adding to Claude Code

```json
{
  "mcpServers": {
    "hippocamp": {
      "command": "/path/to/hippocamp",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

## Testing conventions

- All packages use table-driven tests where multiple cases exist
- Tool handlers are tested via `tools.HandlerFor(store, "toolname")` — no MCP server needed
- `tools.ResultText(result)` extracts the text payload from a `*mcp.CallToolResult`
- Each test creates its own `rdfstore.NewStore()` — no shared state between tests

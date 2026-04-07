# Hippocamp — RDF Knowledge Graph for LLMs

## Project overview

An MCP server that exposes an in-memory RDF knowledge graph to LLMs via five tools: `triple`, `sparql`, `graph`, `search`, `validate`. Built in Go using `mark3labs/mcp-go` for the MCP protocol and `tggo/goRDFlib` for RDF graph operations (SPARQL 1.1, named graphs, TriG serialization).

Hippocamp is a **general-purpose knowledge graph** — it works for code projects, research notes, API documentation, business processes, or any structured knowledge. The `hippo:` ontology has a domain-agnostic base layer (topics, entities, notes, sources, decisions) and an optional code layer (projects, files, symbols, dependencies).

## Commands

```bash
# Run all tests
go test ./...

# Build binary
go build -o hippocamp ./cmd/hippocamp/

# Run server (stdio transport)
./hippocamp --config config.yaml

# One-shot query (for hooks/scripts)
./hippocamp --config config.yaml --query "search terms" --limit 10

# Tidy dependencies
go mod tidy
```

## Architecture

```
cmd/hippocamp/main.go            — entry point: config, store, analytics, auto-setup, signal handler, ServeStdio, --query CLI
internal/analytics/analytics.go  — tool call analytics: logs + stores metrics as RDF triples in urn:hippocamp:analytics
internal/config/config.go        — YAML + ENV config loading
internal/healthcheck/healthcheck.go — background graph health scanner: dangling refs, orphans, alias suggestions
internal/rdfstore/store.go       — Store struct: wraps graph.Dataset (BadgerDB in-memory), dirty tracking
internal/rdfstore/persistence.go — Save/Load/AutoLoad (TriG format via trig.SerializeDataset/ParseDataset)
internal/tools/register.go       — MCP tool registration + HandlerFor() test helper
internal/tools/triple.go         — triple tool: add / remove / list
internal/tools/sparql.go         — sparql tool: SELECT / ASK / UPDATE (auto-detected)
internal/tools/graph.go          — graph tool: create/delete/list/stats/clear/dump/load/prefix_*
internal/tools/search.go         — search tool: keyword search with field boosting, word boundary scoring, graph traversal, temporal search, confidence normalization, explain mode
internal/tools/validate.go       — validate tool: checks ontology compliance (types, labels, rationale)
internal/tools/analyze.go        — analyze tool: god_nodes, components, surprising edges, export_html visualization
internal/setup/setup.go          — auto-setup: embeds hooks+skills, writes to .claude/ on first launch
internal/setup/embedded/         — canonical copies of hooks and skills (embedded in binary)
ontology/hippo.ttl               — hippo: ontology (base layer + code layer)
.claude/skills/project-analyze.md — Claude Code skill: scan any project → RDF triples (domain-agnostic)
.claude/hooks/                   — Claude Code hook templates (pre-query, post-edit)
testdata/                        — sample projects for integration tests (5 domains)
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
- **`search`** is a standalone tool for keyword search across the graph (matches labels, summaries, file paths, signatures, content, URLs)
- **`analyze`** is a read-only analysis tool: god_nodes, components, surprising, export_html, consolidate

### Analyze tool
`analyze` in `analyze.go` provides graph structure analysis via five actions:
- **`god_nodes`**: counts in/out degree per resource (excluding metadata predicates), filters out hub types (Topic, Tag, Project), returns top N by total degree
- **`components`**: builds undirected URI adjacency from relationship triples, runs BFS to find connected components, returns with member URIs, labels, and topic breakdown
- **`surprising`**: finds edges where subject and object have different `hippo:hasTopic` values (cross-topic) or live in different named graphs (cross-graph). Skips metadata predicates.
- **`export_html`**: returns the URL of the built-in visualization server (auto-started on port 39322+ at MCP boot). The page dynamically renders the current graph state. Returns `{"url":"http://localhost:39322"}`
- **`consolidate`**: finds resources with missing/sparse summaries or no topics. Returns suggestions with graph context (references, referenced_by, topics, related_decisions) and a suggested prompt for the LLM to generate the missing data

### Ontology provenance predicates
Three properties for tracking triple quality and origin:
- **`hippo:confidence`**: float 0.0–1.0, distinguishes firm facts from inferences
- **`hippo:provenance`**: string "extracted" / "inferred" / "ambiguous" — how a triple was created
- **`hippo:source`**: URI pointing to the agent/tool/process that produced the resource (inverse of `hippo:sourceOf`)
- **`hippo:revision`**: integer revision counter tracking how many times a resource has been updated (distinct from `hippo:version` which is a dependency version string)

### Search tool implementation
`search` in `search.go` does text matching in Go (not SPARQL FILTER/REGEX) for reliability:
- **Field boosting**: `rdfs:label` matches score 4x, `hippo:summary` and `hippo:alias` 3x, `hippo:content` 1x
- **`hippo:alias`**: alternative names, synonyms, and colloquial terms (e.g. Ukrainian labels for English-labeled resources). Searched with the same boost as `hippo:summary`
- **Word boundary bonus**: keyword matching at the start of a word scores double
- **Score accumulation**: scores from multiple matching predicates on the same subject are summed
- **Graph-aware**: `related=true` parameter enables 1-hop traversal via `hippo:hasTopic`, `hippo:references`, `hippo:partOf`, `hippo:relatedTo` — finds resources linked to direct matches
- **Prefix matching**: when exact substring match fails, tries matching words sharing a 4+ character prefix (e.g. `електрика` → `електропостачання` via shared stem `електр`). Scores at half field weight.
- **Zero-result hints**: when search returns no matches, the response includes a hint with resource count and suggestions for refining the query
- **Confidence normalization**: results include a `confidence` field (0-100%) where the top result is 100% and others are proportional. Helps LLMs assess relative relevance without understanding raw scores
- **Temporal search**: queries containing temporal keywords ("today", "yesterday", "last week", "this month", "recent", "March 2026", ISO dates) are parsed into date ranges. Resources with `hippo:createdAt`/`hippo:updatedAt` timestamps near the range get a scoring boost. Pure temporal queries (no text keywords) surface time-matching resources directly
- **Explain mode**: `explain=true` parameter adds per-field score breakdown to each result — which predicates matched, their individual scores, temporal proximity score, and related-from URI for graph-traversal results

### Validate tool
`validate` in `validate.go` checks graph compliance:
- All `rdf:type` values must be from the `hippo:` namespace (no custom types)
- All typed resources must have `rdfs:label`
- All `hippo:Decision` resources must have `hippo:rationale`
- **Fuzzy type matching**: unknown hippo types are compared to all 17 valid types via LCS-based string similarity. If a match >= 50% is found, the warning says "did you mean hippo:X?" and includes fix commands (remove wrong type + add suggested type)
- Returns JSON with `valid` (bool), `warnings` (array), `fixes` (array), and `stats`

### CLI query mode
`--query` flag in `main.go` enables one-shot search: loads the persisted graph, runs a search, prints JSON results, and exits. Used by hooks and scripts.

### Auto-setup mechanism
On startup (before ServeStdio), the binary writes hooks and skills to `.claude/` in the working directory. Files are embedded in the binary via Go `//go:embed`. Logic:
- First launch: creates `.claude/hooks/` and `.claude/skills/` with embedded files
- Subsequent launches: compares binary build time vs file mtime; overwrites only if binary is newer
- Dev builds (no buildTime injected): always overwrite
- Non-fatal: setup errors are logged to stderr but don't block the MCP server

Build time is injected via ldflags: `-X main.buildTime={{.Date}}` (GoReleaser) or `make build`.

### Ontology design
`ontology/hippo.ttl` has two layers:
- **Base layer** (domain-agnostic): Topic, Entity, Note, Source, Decision, Question, Tag — usable for any knowledge domain
- **Code layer** (software-specific): Project, Module, File, Symbol (Function, Struct, Interface, Class), Dependency, Concept

This allows Hippocamp to be a general-purpose knowledge graph, not just a code analyzer.

### SPARQL named graph support
`SPARQLQuery()` in `store.go` uses `sparql.Parse()` + `sparql.EvalQuery()` (not the simpler `sparql.Query()`). This allows populating `ParsedQuery.NamedGraphs` with all store graphs so `GRAPH <uri> { }` clauses work correctly. Without this, goRDFlib's `Query()` function doesn't expose named graphs to the query engine.

### SPARQL update detection
`isUpdate()` in `sparql.go` checks the first keyword of the query string (INSERT, DELETE, LOAD, etc.) to distinguish updates from queries. Updates go through `store.SPARQLUpdate()` which builds a `sparql.Dataset` struct from the store's named graphs.

### Analytics (tool call tracking)
Every tool call is recorded via mcp-go's `OnBeforeCallTool`/`OnAfterCallTool` hooks. The `analytics.Collector` in `internal/analytics/` does two things:
1. **Log line** (Level 1): `analytics: tool=search query="auth" results=3 duration=12ms`
2. **RDF triples** (Level 2): stored in `urn:hippocamp:analytics` named graph with predicates from `http://purl.org/hippocamp/analytics#` (tool, input, timestamp, durationMs, resultCount, error, graph)

The LLM can SPARQL the analytics graph to answer: "what queries returned 0 results?", "what are the most frequent search terms?", "which tools take longest?". Queries targeting the analytics graph are excluded from recording to prevent infinite loops.

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

### Search integration tests
`TestSearchProjects` in `internal/tools/testprojects_test.go` runs 100+ search flows across 5 domains:
- house-construction, tomato-garden, sales-department, accounting, recipe-collection
- Each project seeds 30-50 triples and runs 20+ search queries
- Tests check: MinResults, MaxResults, MustFind (recall), MustNotFind (precision)
- Metrics logged via `t.Logf`: recall, precision per query
- Helper builders (`entity()`, `topic()`, `note()`, `decision()`, etc.) reduce boilerplate
- To add a new test flow: add a `searchQuery` struct to a project's Queries slice
- To add a new project: write a `testdata_{name}_test.go` with a function returning `testProject`

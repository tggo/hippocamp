<p align="center">
  <a href="https://tggo.github.io/hippocamp/">
    <img src="docs/logo-sm.png" alt="Hippocamp Logo" width="200">
  </a>
</p>

# hippocamp

RDF knowledge graph for LLMs — an MCP server that gives any AI agent a structured, queryable memory via six tools: `triple`, `sparql`, `graph`, `search`, `validate`, `analyze`.

Plug it into any project as a persistent brain. Use the built-in ontology for research notes, construction planning, sales pipelines, recipe collections, or any structured knowledge. Auto-installs Claude Code skills and hooks.

On first launch, Hippocamp automatically sets up Claude Code hooks and skills in your project — no manual configuration needed.

## Why a graph?

Real test: the same question — "what's needed for electricity?" — asked two ways in a home construction project.

| | File scanning | Hippocamp |
|---|---|---|
| **Time** | ~3 min (38 tool calls) | ~2 sec (2 queries) |
| **Tokens** | 60,500 | ~2,000 |
| **Result** | Complete but verbose | Structured, concise |
| **Context** | Large text blocks | Compact JSON |

15× faster. 30× less context consumed. Same answer. The LLM learned on its own: *"from now on, always search Hippocamp first."*

## Install

**Homebrew (macOS / Linux):**
```bash
brew install tggo/tap/hippocamp
```

**Direct download** — pick your platform from [Releases](https://github.com/tggo/hippocamp/releases):
```bash
# macOS Apple Silicon
curl -L https://github.com/tggo/hippocamp/releases/latest/download/hippocamp_darwin_arm64.tar.gz | tar xz
sudo mv hippocamp /usr/local/bin/

# macOS Intel
curl -L https://github.com/tggo/hippocamp/releases/latest/download/hippocamp_darwin_amd64.tar.gz | tar xz
sudo mv hippocamp /usr/local/bin/

# Linux amd64
curl -L https://github.com/tggo/hippocamp/releases/latest/download/hippocamp_linux_amd64.tar.gz | tar xz
sudo mv hippocamp /usr/local/bin/
```

**Build from source:**
```bash
git clone https://github.com/tggo/hippocamp
cd hippocamp
make build
```

## Connect to Claude

Add to your MCP config (`~/.claude/claude_desktop_config.json` or Claude Code settings):

```json
{
  "mcpServers": {
    "hippocamp": {
      "command": "hippocamp",
      "args": ["--config", "/path/to/config.yaml"]
    }
  }
}
```

## Tools

### `triple` — add / remove / list triples

```
add:    {"action":"add","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"Alice","object_type":"literal"}
remove: {"action":"remove","subject":"http://ex.org/Alice","predicate":"http://ex.org/name","object":"http://ex.org/Bob"}
list:   {"action":"list","subject":"http://ex.org/Alice"}
```

| Parameter | Values | Notes |
|---|---|---|
| `action` | `add` \| `remove` \| `list` | required |
| `graph` | URI string | optional, omit for default graph |
| `subject` | URI | required for add/remove, wildcard if empty for list |
| `predicate` | URI | required for add/remove, wildcard if empty for list |
| `object` | string | required for add/remove, wildcard if empty for list |
| `object_type` | `uri` \| `literal` \| `bnode` | default: `uri` |
| `lang` | e.g. `en` | language tag for literals |
| `datatype` | XSD URI | e.g. `http://www.w3.org/2001/XMLSchema#integer` |

---

### `sparql` — SELECT / ASK / UPDATE

```
SELECT: {"query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o } LIMIT 10"}
ASK:    {"query": "ASK { <http://ex.org/Alice> a <http://ex.org/Person> }"}
UPDATE: {"query": "INSERT DATA { <http://ex.org/Alice> <http://ex.org/name> \"Alice\" }"}
Named:  {"query": "SELECT ?name WHERE { ?s <http://ex.org/name> ?name }", "graph": "http://ex.org/g1"}
```

Returns JSON bindings for SELECT, `"true"`/`"false"` for ASK, `"ok"` for updates.

---

### `graph` — graph lifecycle, persistence, prefixes

**Graph management:**
```
{"action":"list"}
{"action":"create","name":"http://example.org/myGraph"}
{"action":"delete","name":"http://example.org/myGraph"}
{"action":"stats","name":"http://example.org/myGraph"}
{"action":"clear","name":"http://example.org/myGraph"}
```

**Persistence:**
```
{"action":"dump","file":"./backup.trig"}
{"action":"load","file":"./backup.trig"}
```

**Overview and migration:**
```
{"action":"summary"}   — compact graph overview (~500 tokens): type counts, top entities, topics, recent decisions
{"action":"migrate"}   — apply pending schema migrations (e.g. add provenance defaults to old data)
```

**Namespace prefixes:**
```
{"action":"prefix_add","prefix":"ex","uri":"http://example.org/"}
{"action":"prefix_list"}
{"action":"prefix_remove","prefix":"ex"}
```

---

### `search` — semantic keyword search

```
{"query": "authentication"}
{"query": "Store", "type": "https://hippocamp.dev/ontology#Struct"}
{"query": "middleware", "scope": "project:myapp", "limit": 5}
```

| Parameter | Notes |
|---|---|
| `query` | Search keywords (required, case-insensitive) |
| `type` | Filter by `rdf:type` URI |
| `scope` | Named graph to search in (omit for all graphs) |
| `limit` | Max results (default: 20) |
| `related` | Follow `hasTopic`, `references`, `partOf` links to include related resources (default: false) |
| `explain` | Add per-field score breakdown, temporal proximity, and popularity data (default: false) |

Searches across `rdfs:label`, `hippo:summary`, `hippo:alias`, `hippo:filePath`, `hippo:signature`, `hippo:content`, `hippo:url`, `hippo:rationale`, and subject URIs. Uses field boosting (label matches score highest), word boundary scoring, and score accumulation across predicates.

Use `hippo:alias` to add synonyms and translations — e.g. Ukrainian labels for English-named resources so search finds them in either language.

With `related=true`, also returns resources that link TO direct matches via relationship predicates (1-hop graph traversal).

When no results are found, the response includes a hint with the total resource count and suggestions for refining the query.

---

### `validate` — ontology compliance check

```
{}
{"scope": "project:house-construction"}
```

| Parameter | Notes |
|---|---|
| `scope` | Named graph to validate (omit for all graphs) |

Checks:
- All `rdf:type` values are from the `hippo:` namespace
- All typed resources have `rdfs:label`
- All Decisions have `hippo:rationale`
- **Fuzzy type matching**: typos like `hippo:Entiy` get a suggestion — *"did you mean hippo:Entity? (91% similar)"* — with fix commands
- **Schema migrations**: warns if a newer schema version is available with `graph action=migrate` fix

Returns JSON with `valid` (bool), `warnings` (array), `fixes` (array of suggested commands), and `stats`.

---

### `analyze` — graph structure analysis

```
{"action":"god_nodes","limit":5}
{"action":"components"}
{"action":"surprising"}
{"action":"consolidate","limit":10}
{"action":"export_html"}
```

| Action | What it does |
|---|---|
| `god_nodes` | Most-connected resources by degree, excluding Topic/Tag/Project hubs |
| `components` | Connected components (clusters) via BFS |
| `surprising` | Cross-topic or cross-graph edges (bridges between clusters) |
| `consolidate` | Find resources with missing summaries/topics, suggest enrichments with graph context |
| `export_html` | Return URL of the live visualization server (auto-started on port 39322) |

| Parameter | Notes |
|---|---|
| `action` | required |
| `scope` | Named graph to analyze (omit for all graphs) |
| `limit` | Max results for god_nodes (default: 10) and consolidate (default: 20) |

---

## CLI query mode

For scripts and hooks, run a one-shot search without starting the MCP server:

```bash
hippocamp --config config.yaml --query "search terms"
hippocamp --config config.yaml --query "auth" --type "https://hippocamp.dev/ontology#Function" --limit 5
```

Loads the persisted graph, runs the search, prints pretty-printed JSON, and exits.

## Ontology

Hippocamp includes a lightweight RDF ontology (`ontology/hippo.ttl`) with two layers:

**Base layer** (any domain):
- `hippo:Topic` — subject areas, themes
- `hippo:Entity` — people, orgs, products, APIs, datasets
- `hippo:Note` — free-form observations
- `hippo:Source` — articles, papers, URLs, books
- `hippo:Decision` — recorded decisions with rationale
- `hippo:Question` — open questions
- `hippo:Tag` — lightweight labels

**Code layer** (software projects):
- `hippo:Project`, `hippo:Module`, `hippo:File`
- `hippo:Function`, `hippo:Struct`, `hippo:Interface`, `hippo:Class`
- `hippo:Dependency`, `hippo:Concept`

**Provenance and quality tracking:**
- `hippo:confidence` — float 0.0-1.0 (how certain is this fact?)
- `hippo:provenance` — `"extracted"` / `"inferred"` / `"ambiguous"`
- `hippo:source` — URI of the agent/tool that produced the resource

**Temporal validity:**
- `hippo:validFrom` / `hippo:validTo` — mark facts as time-bounded (invalidated facts stay in graph for history)

The ontology is open-world — extend with your own classes and properties.

## Schema migrations

When you upgrade hippocamp, your existing data keeps working (RDF is schema-less). But new features may benefit from enriching old data. The migration system handles this automatically:

1. Run `validate` — if a migration is available, you'll see:
   ```
   "schema update available — run graph action=migrate to apply 1 pending migration(s)"
   ```
2. Run `graph action=migrate` — applies all pending migrations and reports what changed
3. The graph's schema version is stored as a triple and checked on each validate

Current migrations:
- **v2**: Adds `hippo:provenance "extracted"` and `hippo:confidence 1.0` to all typed resources that lack them

Migrations are additive — they never delete data. Resources that already have the properties are skipped.

## Claude Code integration

### Auto-setup

Hippocamp automatically writes hooks and skills to your project's `.claude/` directory on first launch. No manual copying needed. On subsequent launches, files are only updated if the binary is newer.

### Skill: project-analyze

Ask Claude: *"analyze this project and build a knowledge graph"*. The skill file (`.claude/skills/project-analyze.md`) is auto-installed and instructs Claude how to scan your project, extract topics, entities, notes, decisions, questions, and sources into the graph.

### Hooks

Hooks are auto-installed. You can customize them in `.claude/settings.json`:

```json
{
  "hooks": {
    "UserPromptSubmit": [
      { "command": ".claude/hooks/hippocamp-pre-query.sh \"$PROMPT\"" }
    ],
    "PostToolUse": [
      {
        "command": ".claude/hooks/hippocamp-post-edit.sh \"$TOOL_NAME\" \"$FILE_PATH\"",
        "matcher": "Edit|Write"
      }
    ]
  }
}
```

- **Pre-query hook**: Before each prompt, queries the graph for relevant context
- **Post-edit hook**: After file edits, tracks which files need re-indexing

## Configuration

```yaml
# config.yaml
store:
  default_file: "./data/default.trig"  # loaded on startup, saved on SIGINT/SIGTERM
  auto_load: true
  format: "trig"

prefixes:
  ex: "http://example.org/"
  schema: "http://schema.org/"
```

**ENV overrides:**

| Variable | Default |
|---|---|
| `HIPPOCAMP_STORE_DEFAULT_FILE` | `./data/default.trig` |
| `HIPPOCAMP_STORE_AUTO_LOAD` | `true` |
| `HIPPOCAMP_STORE_FORMAT` | `trig` |

## Persistence

- **Auto-load** on startup from `store.default_file` (if `auto_load: true` and file exists)
- **Manual dump** via `{"action":"dump","file":"path.trig"}`
- **Auto-save** on `SIGINT` / `SIGTERM` if there are unsaved changes

Data is stored in [TriG](https://www.w3.org/TR/trig/) format — a superset of Turtle that preserves named graph boundaries.

## Named graphs

All tools accept an optional `graph` parameter (URI string). Omit it to use the default graph.

```
# Add to named graph
{"action":"add","graph":"http://ex.org/people","subject":"...","predicate":"...","object":"..."}

# Query named graph
{"query":"SELECT ?s WHERE { ?s ?p ?o }","graph":"http://ex.org/people"}

# Create / delete graphs
{"action":"create","name":"http://ex.org/people"}
{"action":"delete","name":"http://ex.org/people"}
```

## Benchmarks

Memory retrieval benchmarks adapted from [LongMemEval](https://github.com/xiaowu0162/LongMemEval) (ICLR 2025), [ConvoMem](https://github.com/SalesforceAIResearch/ConvoMem) (Salesforce), and [LoCoMo](https://github.com/snap-research/locomo) (Snap Research). Measures R@5 (correct result in top 5) using keyword search only, no embeddings.

| Benchmark | R@5 | Details |
|---|---|---|
| **MemEval** (v2) | **100%** | extraction 100%, reasoning 100%, temporal 100% |
| **ConvoMem** (v2) | **100%** | user facts 100%, preferences 100%, temporal 100%, implicit 100% |
| **LoCoMo** (v2) | **88.9%** | single-hop 100%, multi-hop 100%, temporal 100%, adversarial 0% |

Known weakness: adversarial/abstention queries (keyword search can't distinguish "no relevant result" from partial keyword matches).

Run benchmarks:
```bash
go test ./internal/tools/ -run TestMemoryBenchmark -v
```

## Visualization

Hippocamp starts an HTTP server on `localhost:39322` alongside the MCP server. Open it in a browser for an interactive graph visualization with:
- Nodes colored by type, sized by degree
- Search and type filter
- Click-to-inspect with full property list and edge details

## Development

```bash
make test      # run all tests
make build     # build binary
make lint      # go vet
make run       # build + run with config.yaml
```

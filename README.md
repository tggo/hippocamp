<p align="center">
  <a href="https://tggo.github.io/hippocamp/">
    <img src="docs/logo-sm.png" alt="Hippocamp Logo" width="200">
  </a>
</p>

# hippocamp

RDF knowledge graph for LLMs — an MCP server that gives any AI agent a structured, queryable memory via four tools: `triple`, `sparql`, `graph`, `search`.

Plug it into any project as a persistent brain. Use the built-in ontology for code analysis, research notes, API documentation, or any structured knowledge. Includes a Claude Code skill (`/project-analyze`) and hooks for automatic graph queries.

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

Searches across `rdfs:label`, `hippo:summary`, `hippo:filePath`, `hippo:signature`, `hippo:content`, `hippo:url`, and subject URIs. Returns JSON array of matching resources with type, label, summary, and properties.

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

The ontology is open-world — extend with your own classes and properties.

## Claude Code integration

### Skill: `/project-analyze`

Copy `.claude/skills/project-analyze.md` to your project. Run `/project-analyze` to scan the codebase and build a knowledge graph with files, symbols, dependencies, and architectural concepts.

### Hooks

Copy `.claude/hooks/` to your project and configure in `.claude/settings.json`:

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

## Development

```bash
make test      # run all tests
make build     # build binary
make lint      # go vet
make run       # build + run with config.yaml
```

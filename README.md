# hippocamp

RDF MCP server — exposes an in-memory knowledge graph to LLMs via three tools: `triple`, `sparql`, `graph`.

Built with [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go) and [tggo/goRDFlib](https://github.com/tggo/goRDFlib).

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

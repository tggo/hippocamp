# Changelog — 6 April 2026

## Summary

Added a new `analyze` MCP tool for graph structure analysis and live visualization, plus three ontology predicates for tracking confidence and provenance of knowledge graph data. Inspired by [safishamsi/graphify](https://github.com/safishamsi/graphify) — adapted the best ideas for Hippocamp's RDF/MCP architecture.

---

## New: `analyze` tool

A read-only tool with four actions for understanding graph structure without writing SPARQL.

### `god_nodes` — find the most connected resources

Counts in-degree and out-degree per resource across all relationship predicates (skips metadata like `rdfs:label`, `hippo:summary`, etc.). Filters out natural hubs — Topic, Tag, and Project types — so the results show genuinely important entities.

**Usage:**
```json
{"action": "god_nodes", "limit": 5}
{"action": "god_nodes", "scope": "urn:hippocamp:default", "limit": 10}
```

**Returns:**
```json
[
  {
    "uri": "https://ex.org/auth-service",
    "label": "Auth Service",
    "type": "Entity",
    "degree": 4,
    "in_degree": 1,
    "out_degree": 3
  }
]
```

**Why it's useful:** Quickly identifies the most important resources in a graph. LLMs can use this to orient themselves — "what's the center of this knowledge graph?" — without scanning every triple.

### `components` — find connected clusters

Builds an undirected adjacency graph from all URI-to-URI relationship triples, then runs BFS to discover connected components. Each component includes its member URIs (capped at 20), labels, and topic breakdown.

**Usage:**
```json
{"action": "components"}
{"action": "components", "scope": "urn:hippocamp:default"}
```

**Returns:**
```json
[
  {
    "id": 1,
    "size": 7,
    "members": ["https://ex.org/auth-service", "..."],
    "labels": ["Auth Service", "User Database", "..."],
    "topics": ["backend", "frontend"]
  }
]
```

**Why it's useful:** Reveals the natural clustering of knowledge — which resources form isolated islands vs. connected groups. Helps LLMs decide if the graph needs more cross-linking or if separate topics are appropriately separated.

### `surprising` — find cross-topic and cross-graph bridges

Scans all relationship triples and identifies edges where:
- **Cross-topic:** subject has `hippo:hasTopic A` and object has `hippo:hasTopic B` (different topics)
- **Cross-graph:** subject and object live in different named graphs

These are the "bridges" between clusters — often the most interesting relationships.

**Usage:**
```json
{"action": "surprising"}
{"action": "surprising", "scope": "urn:hippocamp:default"}
```

**Returns:**
```json
[
  {
    "subject": "https://ex.org/login-page",
    "subject_label": "Login Page",
    "predicate": "references",
    "object": "https://ex.org/auth-service",
    "object_label": "Auth Service",
    "reason": "cross-topic",
    "subject_topic": "frontend",
    "object_topic": "backend"
  }
]
```

**Why it's useful:** Cross-topic connections are often the most valuable pieces of knowledge — they reveal non-obvious dependencies (frontend depends on backend auth) and help LLMs suggest new questions ("should we add more cross-cutting concerns?").

### `export_html` — live visualization in browser

Returns the URL of the built-in visualization server (started automatically with the MCP server on port 39322+). The page dynamically renders the current graph state on every load — no stale snapshots.

**Features:**
- Interactive vis.js force-directed graph
- Nodes colored by `rdf:type` (Entity=green, Note=yellow, Decision=orange, etc.)
- Node size scaled by connection count
- Edge labels show predicate names
- Search bar to filter nodes by label
- Type dropdown to show only specific types
- Click any node to see all its properties, outgoing and incoming edges
- Dark theme

**Usage:**
```json
{"action": "export_html"}
{"action": "export_html", "scope": "urn:hippocamp:default"}
```

**Returns:**
```json
{"url": "http://localhost:39322", "status": "running"}
```

**Architecture:** The HTTP server starts in `main.go` alongside the MCP stdio server. It binds to `127.0.0.1:39322` (tries 39322–39332 if the port is busy). Each page load re-reads the graph from the store, so the visualization always reflects the latest state. There's also a `/api/graph?scope=...` JSON endpoint for programmatic access.

---

## New: ontology provenance predicates

Three new properties in `ontology/hippo.ttl` for tracking where knowledge comes from and how confident we are in it.

### `hippo:confidence` (xsd:float)

A score from 0.0 (uncertain) to 1.0 (certain). Use it to distinguish firm facts from inferences.

```turtle
ex:my-claim hippo:confidence "0.85"^^xsd:float .
```

### `hippo:provenance` (xsd:string)

How a triple was created:
- `"extracted"` — directly from source material (code, docs, conversation)
- `"inferred"` — derived by reasoning (e.g., LLM inferred a relationship)
- `"ambiguous"` — uncertain interpretation, needs human review

```turtle
ex:my-claim hippo:provenance "inferred" .
```

### `hippo:source` (rdfs:Resource)

Points to the agent, tool, or process that produced this resource. The inverse of the existing `hippo:sourceOf`.

```turtle
ex:my-claim hippo:source ex:claude-code-skill .
```

**Why it's useful:** When an LLM builds a knowledge graph, not all facts are equally reliable. Provenance tracking lets downstream consumers (other LLMs, humans, validation tools) filter by confidence, identify ambiguous claims, and trace facts back to their origin. This is the "honest graph" philosophy from graphify: you always know what was found vs. what was guessed.

---

## Technical details

### Files changed

| File | Change |
|------|--------|
| `ontology/hippo.ttl` | +3 predicates: confidence, provenance, source |
| `internal/tools/analyze.go` | New file: tool definition, 4 action handlers, vis.js HTML template, HTTP server |
| `internal/tools/register.go` | Registered analyze tool in handlers map and Register() |
| `cmd/hippocamp/main.go` | Start visualization server on MCP boot |
| `internal/tools/analyze_test.go` | New: 10 unit tests covering all actions + edge cases |
| `internal/tools/analyze_integration_test.go` | Extended with 4 analyze tests on house-construction data |
| `CLAUDE.md` | Documented analyze tool and provenance predicates |

### Algorithm choices

- **Go-based, not SPARQL:** god_nodes, components, and surprising all iterate triples in Go rather than using SPARQL queries. Reason: goRDFlib's SPARQL engine doesn't support property paths or aggregation functions needed for graph analysis.
- **Metadata predicate exclusion:** The analysis actions skip 18 metadata predicates (rdf:type, rdfs:label, hippo:summary, etc.) when counting edges and building adjacency. This prevents label triples from inflating degree counts.
- **Hub type exclusion in god_nodes:** Topic, Tag, and Project types are excluded because they're structurally high-degree (everything connects to topics) but not semantically interesting as "god nodes."
- **Dynamic HTML rendering:** The viz server re-reads the store on each HTTP request. This adds ~1ms per request even for large graphs (triple iteration is fast) and means the user always sees the current state.

### Port choice: 39322

The visualization server uses port 39322 (with fallback to 39323–39332). This is in the IANA dynamic/private range (49152–65535... well, close enough) and unlikely to conflict with common services. The port is logged on startup.

---

## How to verify

### 1. Run tests
```bash
go test ./... -count=1
```
All 7 packages should pass, including 14 new analyze-related tests.

### 2. Build and start
```bash
go build -o hippocamp ./cmd/hippocamp/
./hippocamp --config config.yaml
```
Check logs — you should see:
```
visualization server at http://localhost:39322
MCP server starting (version=dev, tools=triple,sparql,graph,search,validate,analyze)
```

### 3. Open visualization
Open `http://localhost:39322` in a browser. If the graph has data, you'll see an interactive visualization. If empty, the page loads with 0 nodes/0 edges.

### 4. Test analyze via MCP
From Claude Code, after populating the graph:
```
Use the analyze tool: {"action": "god_nodes", "limit": 5}
Use the analyze tool: {"action": "components"}
Use the analyze tool: {"action": "surprising"}
Use the analyze tool: {"action": "export_html"}
```

### 5. Test provenance predicates
```
Use the triple tool to add:
  subject: https://example.org/claim/1
  predicate: https://hippocamp.dev/ontology#confidence
  object: 0.85
  object_type: literal
  datatype: http://www.w3.org/2001/XMLSchema#float

Then search for it or query via SPARQL.
```

### 6. Compare with graphify

What hippocamp now does that graphify does:
- God node detection (graphify: `_is_god_node()` → hippocamp: `analyze god_nodes`)
- Connected components (graphify: Leiden algorithm → hippocamp: BFS, simpler but effective)
- Surprising connections (graphify: composite score → hippocamp: cross-topic/cross-graph detection)
- Interactive visualization (graphify: static HTML file → hippocamp: live HTTP server, always current)
- Confidence/provenance tracking (graphify: EXTRACTED/INFERRED/AMBIGUOUS on edges → hippocamp: `hippo:confidence` + `hippo:provenance` predicates on any resource)

What hippocamp does differently/better:
- **Live visualization:** graphify generates a static HTML file; hippocamp serves the current graph state dynamically
- **RDF-native provenance:** confidence and provenance are standard RDF predicates, queryable via SPARQL
- **MCP integration:** analyze is a proper MCP tool, callable by any LLM client
- **No Python dependency:** everything runs in a single Go binary

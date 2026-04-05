# Hippocamp Improvement Plan — 5 April 2026

## Vision

Turn Hippocamp into an **LLM-RDF-Graph brain** — a structured knowledge layer that plugs into any project via Claude Code. Inspired by Karpathy's LLM Knowledge Bases, but with RDF graphs instead of flat markdown: structured triples, SPARQL queries, named graphs per project, and ontology-based reasoning.

## What we're building

### 1. Hippocamp Ontology (`hippo:`)

A lightweight RDF ontology for representing project knowledge:

```turtle
@prefix hippo: <https://hippocamp.dev/ontology#> .
@prefix rdfs:  <http://www.w3.org/2000/01/rdf-schema#> .

# --- Classes ---
hippo:Project    a rdfs:Class .
hippo:Module     a rdfs:Class .  # Go package, Python module, etc.
hippo:File       a rdfs:Class .
hippo:Symbol     a rdfs:Class .  # Function, struct, class, variable
hippo:Function   rdfs:subClassOf hippo:Symbol .
hippo:Struct     rdfs:subClassOf hippo:Symbol .
hippo:Interface  rdfs:subClassOf hippo:Symbol .
hippo:Class      rdfs:subClassOf hippo:Symbol .
hippo:Dependency a rdfs:Class .  # External dependency
hippo:Concept    a rdfs:Class .  # Architectural concept, pattern, decision

# --- Properties ---
hippo:language     rdfs:domain hippo:Project ;  rdfs:range rdfs:Literal .
hippo:rootPath     rdfs:domain hippo:Project ;  rdfs:range rdfs:Literal .
hippo:inProject    rdfs:domain hippo:File ;     rdfs:range hippo:Project .
hippo:inModule     rdfs:domain hippo:File ;     rdfs:range hippo:Module .
hippo:filePath     rdfs:domain hippo:File ;     rdfs:range rdfs:Literal .
hippo:defines      rdfs:domain hippo:File ;     rdfs:range hippo:Symbol .
hippo:calls        rdfs:domain hippo:Symbol ;   rdfs:range hippo:Symbol .
hippo:imports      rdfs:domain hippo:File ;     rdfs:range hippo:Module .
hippo:dependsOn    rdfs:domain hippo:Project ;  rdfs:range hippo:Dependency .
hippo:implements   rdfs:domain hippo:Symbol ;   rdfs:range hippo:Interface .
hippo:summary      rdfs:range rdfs:Literal .
hippo:signature    rdfs:domain hippo:Symbol ;   rdfs:range rdfs:Literal .
hippo:lineNumber   rdfs:domain hippo:Symbol ;   rdfs:range rdfs:Literal .
hippo:relatedTo    rdfs:domain hippo:Concept ;  rdfs:range hippo:Concept .
hippo:describes    rdfs:domain hippo:Concept ;  rdfs:range rdfs:Resource .
hippo:version      rdfs:domain hippo:Dependency ; rdfs:range rdfs:Literal .
```

Each project gets a **named graph**: `<project:{project-name}>`.

### 2. New MCP tool: `search`

A semantic search tool that combines text matching with SPARQL for natural queries.

**Parameters:**
- `query` (required) — natural language or keyword search
- `type` — filter by RDF type (file, function, module, concept, etc.)
- `scope` — named graph to search in (project name)
- `limit` — max results (default: 20)

**Implementation:** The tool builds a SPARQL query that:
1. Matches `rdfs:label`, `hippo:summary`, `hippo:filePath` against the query text using FILTER + regex
2. Optionally filters by `rdf:type`
3. Returns subject URI, type, label, summary, and related triples

**File:** `internal/tools/search.go`

### 3. Claude Code Skill: `project-analyze`

A skill (`.claude/skills/project-analyze.md`) that instructs Claude to:

1. Scan the project structure (files, directories, modules)
2. Identify the language/framework
3. Parse key symbols (functions, structs, classes, interfaces)
4. Map dependencies (imports, calls, implements)
5. Extract architectural patterns and decisions (from CLAUDE.md, README, comments)
6. Emit all of this as RDF triples via the `triple` and `sparql` MCP tools
7. Store everything in a named graph `<project:{name}>`

The skill is invoked manually (`/project-analyze`) or triggered by a hook on first interaction with a new project.

### 4. Claude Code Hooks

**`UserPromptSubmit` hook — pre-query:**

Before every prompt, query Hippocamp for relevant context:
```bash
#!/bin/bash
# .claude/hooks/hippocamp-pre-query.sh
# Extracts keywords from the prompt and queries the graph
# Output goes to stderr so it appears in Claude's context
```

This hook calls `hippocamp` CLI in query mode (new `--query` flag) to search the graph and return relevant triples as context.

**`PostToolUse` hook — sync after edits:**

After file edits (Edit, Write tools), update the graph:
```bash
#!/bin/bash
# .claude/hooks/hippocamp-post-edit.sh
# Detects which file was modified and updates its triples
```

### 5. CLI query mode

Add a `--query` flag to the hippocamp binary for hook integration:
```bash
hippocamp --config config.yaml --query "authentication middleware"
```

This runs a one-shot search against the persisted graph and prints results to stdout, then exits. No MCP server started.

## Architecture after changes

```
hippocamp (MCP server + CLI)
├── tools: triple, sparql, graph      ← existing
├── tool: search                       ← NEW: semantic search
├── cmd: --query mode                  ← NEW: one-shot CLI search
└── persistence: TriG                  ← existing

.claude/skills/
└── project-analyze.md                 ← NEW: skill definition

.claude/hooks/ (per-project or global)
├── hippocamp-pre-query.sh            ← NEW: UserPromptSubmit hook
└── hippocamp-post-edit.sh            ← NEW: PostToolUse hook
```

## Why RDF beats flat markdown

| Flat markdown (Karpathy) | RDF graph (Hippocamp) |
|---|---|
| Text search only | SPARQL with pattern matching + inference |
| Manual index files | Graph IS the index — every triple is queryable |
| No types or schema | Ontology-based classification (`hippo:Function`, `hippo:Module`) |
| Per-topic wikis, no links | Named graphs per project, shared ontology, cross-references |
| LLM must read/parse markdown | Structured triples — no parsing needed |
| Grows linearly | Graph relationships enable sub-linear lookup |

## Implementation order

1. Define ontology in `ontology/hippo.ttl`
2. Build `search` tool in `internal/tools/search.go` + tests
3. Add `--query` CLI mode to `cmd/hippocamp/main.go`
4. Create `project-analyze` skill in `.claude/skills/project-analyze.md`
5. Create hooks in `.claude/hooks/`
6. Update CLAUDE.md, README.md, docs/index.html

## Files to create/modify

**Create:**
- `ontology/hippo.ttl` — ontology definition
- `internal/tools/search.go` — search tool
- `internal/tools/search_test.go` — search tool tests
- `.claude/skills/project-analyze.md` — skill definition
- `.claude/hooks/hippocamp-pre-query.sh` — pre-query hook
- `.claude/hooks/hippocamp-post-edit.sh` — post-edit hook

**Modify:**
- `internal/tools/register.go` — register search tool
- `cmd/hippocamp/main.go` — add --query CLI mode
- `CLAUDE.md` — document new features
- `README.md` — document search tool, skill, hooks
- `docs/index.html` — add search tool section, update feature list

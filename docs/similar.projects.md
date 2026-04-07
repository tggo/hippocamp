# Similar Projects — Inspiration for Hippocamp

Research conducted 6 April 2026. Projects grouped by relevance to hippocamp's architecture (Go, RDF, MCP, in-memory graph).

---

## Tier 1 — Most Relevant

### Graphiti by Zep
- **GitHub:** https://github.com/getzep/graphiti (20K+ stars)
- **What it is:** Temporal knowledge graph engine for AI agents. Each fact has a validity window (when it became true, when superseded). Built on Neo4j/FalkorDB.
- **Key features:** Episode-based ingestion, temporal fact tracking, fact invalidation (superseded, not deleted), MCP server with add_episode/search_facts/search_nodes tools, multiple LLM providers.
- **What we can take:**
  - `hippo:validFrom` / `hippo:validUntil` predicates for temporal facts
  - Episode-based ingestion — group facts by session, commit, or document
  - Fact supersession instead of deletion — mark old facts as replaced, not removed

### codebase-memory-mcp by DeusData
- **GitHub:** https://github.com/DeusData/codebase-memory-mcp
- **What it is:** High-performance code intelligence MCP server. Single static binary, SQLite-backed, 66 languages, sub-ms queries. Indexes codebases into a persistent knowledge graph.
- **Key features:** 14 MCP tools, call graph tracing, Louvain community detection, git diff impact mapping with risk classification, dead code detection, cross-service HTTP linking, ADR management, auto-sync on file changes.
- **What we can take:**
  - Impact analysis: map git diff to affected graph resources with risk scores
  - Louvain community detection as alternative to our BFS `components`
  - ADR management workflow — we have `hippo:Decision` but could improve the UX
  - Token reduction benchmarking (they report ~412K tokens via file search vs ~3.4K via graph)

### cognee by Topoteretes
- **GitHub:** https://github.com/topoteretes/cognee
- **What it is:** Knowledge engine for AI agent memory. Ingests any data format, builds knowledge graph + vector embeddings, supports ontology grounding against OWL vocabularies.
- **Key features:** ECL pipeline (Extract, Cognify, Load), ontology validation with fuzzy matching (80% cutoff), contradiction detection, multi-backend (Neo4j, FalkorDB, KuzuDB, NetworkX for graphs; Redis, Qdrant, Weaviate for vectors), multimodal support.
- **What we can take:**
  - Ontology fuzzy matching in `validate` — if LLM creates `hippo:Component` (nonexistent), suggest nearest valid type `hippo:Concept`
  - Contradiction detection — find conflicting facts in the graph (A says X, B says not-X)
  - Ontology grounding workflow — auto-map extracted entities to ontology classes

### obra/knowledge-graph
- **GitHub:** https://github.com/obra/knowledge-graph
- **What it is:** Obsidian vault parser that builds an untyped graph (files=nodes, wikilinks=edges), indexes into SQLite with vector embeddings and full-text search. 10 MCP operations. Claude Code plugin.
- **Key features:** Semantic search, path finding, community detection, `prove-claim` skill that teaches an agent a structured workflow for investigating claims through the graph.
- **What we can take:**
  - `prove-claim` skill pattern — teach LLM to verify facts via search + SPARQL traversal
  - Community detection on wiki-link graphs
  - Structured investigation workflows as Claude Code skills

---

## Tier 2 — Useful Ideas

### Karpathy's LLM Wiki
- **Gist:** https://gist.github.com/karpathy/442a6bf555914893e9891c11519de94f
- **Coverage:** https://venturebeat.com/data/karpathy-shares-llm-knowledge-base-architecture-that-bypasses-rag-with-an
- **What it is:** A pattern for LLM-maintained knowledge bases. Not RAG (rediscover knowledge per query) but "compiled wiki" — LLM writes and maintains markdown files, human sources and asks questions. Uses Obsidian as IDE, LLM as programmer, wiki as codebase.
- **Key workflow:** /raw folder for source materials -> LLM incrementally compiles into structured wiki pages -> cross-references, visualizations, summaries maintained automatically.
- **What we can take:**
  - Incremental compile pattern — update only changed parts of the graph, not rescan everything
  - Wiki/markdown export from the graph (`analyze export_wiki` — generate readable docs from RDF)
  - "LLM writes the knowledge, human curates" as a first-class workflow

### agent-memory by Martian Engineering
- **GitHub:** https://github.com/Martian-Engineering/agent-memory
- **What it is:** Three-layer memory: knowledge graph + daily notes + tacit knowledge. Automated extraction, contradiction detection, recency-scored retrieval.
- **What we can take:**
  - Recency boost in search — rank recently-updated resources higher using `hippo:updatedAt`
  - Tacit knowledge layer — patterns inferred from behavior, not explicitly stated (our `surprising` action does something similar)
  - Staleness detection — flag facts that haven't been verified/updated in N days

### memento-mcp
- **GitHub:** https://github.com/gannonh/memento-mcp
- **What it is:** Neo4j-backed knowledge graph memory with semantic retrieval, contextual recall, and temporal awareness. Uses Neo4j for both graph storage and vector search.
- **What we can take:**
  - Semantic similarity search via embeddings (would need an embedding model but dramatically improves recall)
  - Contextual recall — retrieve not just matches but surrounding context from the graph

### MegaMem
- **GitHub:** https://github.com/C-Bjorn/MegaMem
- **What it is:** Syncs Obsidian notes to a graph DB. Entities become nodes, relationships extracted by AI. Every fact timestamped. 11 graph tools + 10 vault file tools via MCP.
- **What we can take:**
  - Automatic entity extraction from markdown notes
  - Timestamped fact evolution — graph grows over time with full history

---

## Tier 3 — Interesting but Less Relevant

### TrustGraph
- **GitHub:** https://github.com/trustgraph-ai/trustgraph
- **What it is:** Context development platform. RDF triplestore with OWL ontologies, SPARQL template-driven retrieval, Cassandra/Neo4j backends, vector search via Qdrant.
- **Interesting:** Uses RDF Turtle and JSON-LD as output formats for LLMs. SPARQL templates guide retrieval instead of freeform queries.

### persistor
- **GitHub:** https://github.com/persistorai/persistor
- **What it is:** PostgreSQL-native encrypted knowledge graph + vector memory. Local-first, open source.
- **Interesting:** Encryption at rest for sensitive knowledge graphs.

### agentmemory
- **GitHub:** https://github.com/rohitg00/agentmemory
- **What it is:** 41 MCP tools, triple-stream retrieval (BM25 + vector + knowledge graph), 4-tier memory consolidation, provenance-tracked citations, cascading staleness.
- **Interesting:** Triple-stream retrieval combining three search strategies. Staleness cascading — retired facts don't pollute context.

### memory-graph
- **GitHub:** https://github.com/memory-graph/memory-graph
- **What it is:** Graph DB-based MCP memory server for coding agents. Intelligent relationship tracking across sessions.
- **Interesting:** Pattern storage — learns recurring code patterns and surfaces them proactively.

### graphiti-mcp-ollama
- **GitHub:** https://github.com/Flo976/graphiti-mcp-ollama
- **What it is:** Graphiti + Ollama + FalkorDB for fully local Claude Code memory. No cloud APIs.
- **Interesting:** Fully local LLM-powered knowledge graph with zero cloud dependency.

### Obsidian MCP ecosystem
- **jlevere/obsidian-mcp-plugin** — Embedded MCP server within Obsidian, structured data with custom schemas
- **cyanheads/obsidian-mcp-server** — Full vault CRUD via MCP (notes, tags, frontmatter)
- **Piotr1215/mcp-obsidian** — Regex search, tag-based search, discover-mocs for vault structure

---

## Priority Ideas for Hippocamp

| # | Idea | Source | Complexity | Impact |
|---|------|--------|-----------|--------|
| 1 | Temporal predicates (`validFrom`/`validUntil`) | Graphiti | Low | High — facts have lifecycle |
| 2 | Recency boost in search (`updatedAt` scoring) | agent-memory | Low | Medium — fresher results first |
| 3 | Ontology fuzzy matching in validate | cognee | Medium | High — auto-correct LLM type errors |
| 4 | Contradiction detection | cognee, agent-memory | Medium | High — find conflicting facts |
| 5 | Impact analysis (git diff -> graph) | codebase-memory-mcp | Medium | Medium — connect code changes to knowledge |
| 6 | Wiki/markdown export from graph | Karpathy wiki | Medium | Medium — human-readable output |
| 7 | prove-claim skill | obra/knowledge-graph | Low (skill only) | Medium — structured fact verification |
| 8 | Semantic search (embeddings) | memento-mcp | High | High — dramatically better recall |
| 9 | Fact supersession (not deletion) | Graphiti | Low | Medium — knowledge history |
| 10 | Staleness detection | agent-memory, agentmemory | Low | Medium — flag outdated facts |

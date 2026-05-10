---
name: project-analyze
description: Analyze any project and build a structured RDF knowledge graph in Hippocamp. Scans documents, extracts entities, topics, decisions, and relationships to create a queryable knowledge base. Works for any domain — construction, finance, research, sales, recipes, gardening, etc.
---

# Project Analyze

You are building a structured knowledge graph of the current project using the Hippocamp MCP server. Your output is RDF triples stored via the `triple` and `sparql` tools.

The graph is **domain-agnostic** — it works for any kind of project: business documents, research notes, personal collections, planning materials, or code repositories.

## Ontology

Use the `hippo:` namespace (`https://hippocamp.dev/ontology#`). Core types:

### Base layer (any domain)
- `hippo:Topic` — subject areas, themes, categories
- `hippo:Entity` — people, organizations, products, services, places, accounts, varieties, recipes, tools, equipment — ANY identifiable thing
- `hippo:Note` — observations, instructions, specifications, summaries
- `hippo:Source` — reference documents, articles, URLs, books, standards
- `hippo:Decision` — recorded decisions with rationale (MUST include `hippo:rationale`)
- `hippo:Question` — open questions, uncertainties, things to investigate
- `hippo:Tag` — lightweight labels for cross-cutting categorization

**IMPORTANT: Use ONLY these types.** Do NOT invent custom types like `hippo:TomatoVariety`, `hippo:Person`, `hippo:Recipe`, `hippo:Contractor`. Instead, use `hippo:Entity` for ALL concrete things and add `hippo:hasTag` for sub-classification. For example, a tomato variety is `hippo:Entity` with `hippo:hasTag` → `tag/tomato-variety`.

### Key properties
- `rdfs:label` — display name (always set). Use language tags for multilingual labels: `"Electrical"@en`, `"Електрика"@uk`
- `rdf:type` — classification
- `hippo:alias` — synonyms, abbreviations, colloquial terms, and translations. Searched with the same boost as summaries. Add aliases in the user's language if labels are in a different language (e.g. `hippo:alias "світло"@uk` for an English-labeled resource)
- `hippo:summary` — one-sentence description
- `hippo:content` — full text content (for notes, decisions)
- `hippo:url` — web reference
- `hippo:hasTopic` — links any resource to a topic
- `hippo:hasTag` — links any resource to a tag
- `hippo:references` — directed link between resources
- `hippo:partOf` — hierarchical containment
- `hippo:relatedTo` — general association
- `hippo:rationale` — why a decision was made
- `hippo:status` — current state (open, resolved, active, etc.)
- `hippo:createdAt` — ISO 8601 timestamp
- `hippo:sourceOf` — links a source to produced knowledge

### Provenance properties (use when confidence varies)
- `hippo:confidence` — float 0.0–1.0. Use 1.0 for facts directly stated in documents, 0.7–0.9 for inferred relationships, 0.3–0.6 for uncertain claims
- `hippo:provenance` — how the triple was created: `"extracted"` (from source text), `"inferred"` (by reasoning), `"ambiguous"` (uncertain)
- `hippo:source` — URI of the agent or process that produced this resource (e.g. `hippo:source <https://hippocamp.dev/skill/project-analyze>`)

### Temporal validity (use when facts have a lifespan)
- `hippo:validFrom` — ISO 8601 timestamp from which the fact is valid
- `hippo:validTo` — ISO 8601 timestamp until which the fact was valid (non-empty = no longer current)
- Use `triple action=invalidate subject=<uri>` to mark a resource as expired (sets `validTo=now`, `status=invalidated`). The original triples stay in the graph for history.

## Procedure

### Step 1: Setup prefixes and named graph

```
graph action=prefix_add prefix=hippo uri=https://hippocamp.dev/ontology#
graph action=prefix_add prefix=rdfs uri=http://www.w3.org/2000/01/rdf-schema#
graph action=prefix_add prefix=rdf uri=http://www.w3.org/1999/02/22-rdf-syntax-ns#
```

Derive the project name from the directory name. Create a named graph:
```
graph action=create name=project:{name}
```

**Do NOT register a `proj:` prefix mapping `https://hippocamp.dev/project/`.** Slashes are not valid in the local part of a TriG/Turtle prefixed name, so `proj:topic/foo` will fail to parse with `expected ':' in prefixed name`. Always write the full URI: `<https://hippocamp.dev/project/{name}/topic/foo>`. The only safe shortcuts are predicates whose local part has no slash (e.g. `hippo:summary`, `rdfs:label`).

Note: `graph action=import` ignores the named-graph parameter and writes to the default graph. To target a named graph, use `triple action=add graph=project:{name}` or `sparql query="INSERT DATA { ... }" graph=project:{name}`.

### Step 1.5: Check existing graph state and apply migrations

Run `graph action=summary` to see what's already in the graph. This returns a compact overview (~500 tokens): type counts, topics, top entities, recent decisions, invalidation stats. Use this to:
- Avoid re-creating entities that already exist (duplicate detection will warn you, but checking first is faster)
- Understand the current topic structure before adding new ones
- See how many resources have been invalidated

Then run `validate` to check for pending schema migrations. If you see a warning like *"schema update available — run graph action=migrate"*, run `graph action=migrate` before proceeding. This enriches old data with new properties (e.g. adds `hippo:provenance` defaults). **Always do this when starting with an existing graph after updating hippocamp.**

### Step 2: Check for incremental mode

Check if `.claude/.hippocamp-stale` exists. If it does:
- Read the file — it contains paths of files that changed since last analysis
- Only analyze those files (skip full scan)
- For each stale file, remove its old triples: `triple action=list` with the file's URI prefix, then remove matching triples
- Re-analyze only the changed files
- After re-indexing, delete `.claude/.hippocamp-stale`
- Skip to Step 11 (persist)

If `.claude/.hippocamp-stale` does NOT exist, proceed with full analysis below.

### Step 3: Scan the project — DEEP, not shallow

A folder name (`contacts/`, `контакти/`, `suppliers/`, `materials/`, `матеріали/`, `decisions/`, `meetings/`) is **a hint about content, not the content itself**. You MUST open the files inside before claiming the project is indexed. Reading only the root README and listing immediate subfolders is the most common reason a graph comes out impoverished.

**Procedure:**

1. **List recursively** to depth 3–4: `find . -maxdepth 4 -type f \( -name "*.md" -o -name "*.txt" -o -name "*.csv" -o -name "*.json" -o -name "*.yaml" -o -name "*.html" \) | head -200`. Note total count.
2. **Identify content-bearing subfolders.** Any subfolder with ≥1 `.md`/`.txt` is a candidate. Almost always rich in extractable entities: `contacts/`, `контакти/`, `suppliers/`, `постачальники/`, `materials/`, `матеріали/`, `vendors/`, `people/`, `decisions/`, `etapy/`, `етапи/`, `meetings/`, `quotes/`, `bom/`, `quotes/`.
3. **Read every leaf file** in those subfolders (not just folder-level README). For projects with >50 files, prioritize: per-folder README → files referenced from README → remaining files in batches.
4. **Extract structured signals from each file body** — see Step 6 patterns table.
5. **Re-read the top-level README/index at the end** so cross-links to extracted entities aren't missed.

A skip is only acceptable for: binary/image folders (`фото/`, `attachment/`, `*.png|jpg|stl|f3d|pdf`), archive folders (`archive/`), generated content (`node_modules/`, `dist/`, `target/`), and per-day journals (index by month, not per file).

**Sanity check before moving on:** count extracted entities. If a project has rich subfolders (e.g. `contacts/` with 7 files) but you ended up with <5 entities, you scanned too shallow — go back. A typical content-bearing folder yields ≥1 entity per file.

Capture per project:
- What is this project about? (root README, Home.md, top file)
- Major topic areas? (folders + document sections inside files)
- Key people/organizations/products/places? (extract from contact files, vendor lists, comparison tables)
- Decisions with rationale? (look for `Decision:`, `Вибрано:`, `Рішення:`, ✅, winner rows in comparison tables)
- Open questions? (`TODO`, `Питання:`, `?`, `Потрібно з'ясувати`)
- Reference materials? (URLs, PDFs, external standards, datasheets)

### Step 4: Create the project entity

```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name} predicate=rdf:type object=https://hippocamp.dev/ontology#Entity
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name} predicate=rdfs:label object="{Project Name}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name} predicate=hippo:summary object="{one-sentence description}" object_type=literal
```

### Step 5: Extract topics

For each major area/theme in the project, create a Topic:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/topic/{topic-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Topic
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/topic/{topic-slug} predicate=rdfs:label object="{Topic Name}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/topic/{topic-slug} predicate=hippo:summary object="{description}" object_type=literal
```

### Step 6: Extract entities

#### Extraction patterns (apply per file body)

| Signal in file body | Extract as |
|---|---|
| `Телефон:` / `Phone:` / `Tel:` / `+380...` | `hippo:Entity` (contact); include phone in `hippo:summary` |
| `Сайт:` / `Site:` / `Web:` / bare URL | `hippo:url` on the entity |
| `#contractor`, `#supplier`, `#vendor`, `#постачальник`, `#підрядник` | `hippo:hasTag` to corresponding tag resource |
| `tags: [...]` in YAML frontmatter | One `hippo:hasTag` per tag |
| H1 of a contact/supplier file | The entity name (use original-language label, ASCII slug) |
| Address / `Адреса:` / `вул.` / street | Include in `hippo:summary` |
| Comparison table (companies × criteria, "Порівняння", "vs") | One `hippo:Entity` per row + one `hippo:Decision` if a winner is marked (✅, **bold**, "Вибрано", "Decided") with `hippo:references` to all candidates |
| Price / `Ціна:` / `грн` / `UAH` / `USD` in a quote context | Include the figure in `hippo:summary` of the quoting entity |
| `Рішення:` / `Decision:` / "Вибрано X тому що Y" | `hippo:Decision` with `hippo:rationale=Y` and `hippo:references` to entities involved |
| Bullet starting with `?`, `TODO`, `Питання:` | `hippo:Question` with `hippo:status="open"` |
| Inline `[[wiki-link]]` to another note in the project | `hippo:references` from this resource to the linked resource |

For each person, organization, product, place, account, or identifiable thing:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Entity
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=rdfs:label object="{Entity Name}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=hippo:summary object="{role or description}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=hippo:hasTopic object=https://hippocamp.dev/project/{name}/topic/{relevant-topic}
```

For entities inferred (not explicitly stated in source documents), add provenance:
```
triple action=add graph=project:{name} subject=.../{entity-slug} predicate=hippo:confidence object="0.7" object_type=literal datatype=http://www.w3.org/2001/XMLSchema#float
triple action=add graph=project:{name} subject=.../{entity-slug} predicate=hippo:provenance object="inferred" object_type=literal
```

### Step 7: Capture notes

For important observations, specifications, instructions, or summaries:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/note/{note-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Note
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/note/{note-slug} predicate=rdfs:label object="{Note Title}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/note/{note-slug} predicate=hippo:content object="{full text}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/note/{note-slug} predicate=hippo:hasTopic object=https://hippocamp.dev/project/{name}/topic/{topic}
```

### Step 8: Record decisions

For each decision found in the documents:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/decision/{decision-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Decision
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/decision/{decision-slug} predicate=rdfs:label object="{Decision}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/decision/{decision-slug} predicate=hippo:rationale object="{why}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/decision/{decision-slug} predicate=hippo:hasTopic object=https://hippocamp.dev/project/{name}/topic/{topic}
```

### Step 9: Log questions

For open questions or areas of uncertainty:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/question/{q-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Question
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/question/{q-slug} predicate=rdfs:label object="{question}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/question/{q-slug} predicate=hippo:status object="open" object_type=literal
```

### Step 10: Index sources

For reference materials, standards, URLs:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/source/{source-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Source
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/source/{source-slug} predicate=rdfs:label object="{Source Name}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/source/{source-slug} predicate=hippo:summary object="{what it covers}" object_type=literal
```

### Step 11: Link relationships

Connect resources to each other:
- `hippo:references` — one resource cites or points to another
- `hippo:partOf` — hierarchy (subtopic of topic, entity in organization)
- `hippo:relatedTo` — general association between concepts
- `hippo:hasTopic` — categorize anything under a topic
- `hippo:hasTag` — lightweight tagging

### Step 12: Validate and fix

Run `validate` to check for issues. The tool now provides:
- **Fuzzy type suggestions**: if you accidentally used a wrong type (e.g. `hippo:Component`), validate suggests the closest match (e.g. "did you mean hippo:Concept?") with fix commands — apply them.
- **Dangling references**: links to non-existent resources, with `triple action=remove` fix commands.
- **Orphan resources**: typed resources with no relationships — add `hippo:hasTopic` or `hippo:references` to connect them.
- **Missing aliases**: failed search queries that suggest resources need `hippo:alias` — add the suggested aliases.

### Step 13: Consolidate

Run `analyze action=consolidate` to find resources that need enrichment. The tool returns resources with:
- `missing_summary` — add `hippo:summary` using the provided context (references, topics, related decisions)
- `sparse_summary` — expand the summary with more detail
- `no_topic` — add `hippo:hasTopic` to connect the resource to the topic structure

Each suggestion includes a `suggested_prompt` with context. Use it to add the missing data.

### Step 14: Persist

```
graph action=dump file=./data/default.trig
```

## URI conventions

- Project: `https://hippocamp.dev/project/{name}`
- Topic: `https://hippocamp.dev/project/{name}/topic/{slug}`
- Entity: `https://hippocamp.dev/project/{name}/entity/{slug}`
- Note: `https://hippocamp.dev/project/{name}/note/{slug}`
- Decision: `https://hippocamp.dev/project/{name}/decision/{slug}`
- Question: `https://hippocamp.dev/project/{name}/question/{slug}`
- Source: `https://hippocamp.dev/project/{name}/source/{slug}`
- Tag: `https://hippocamp.dev/project/{name}/tag/{slug}`

Use lowercase kebab-case slugs derived from the label.

## Guidelines

- Always set `rdfs:label` and `rdf:type` for every resource
- If project documents use a different language than English, add `hippo:alias` with common terms in that language so search works in both languages
- Add `hippo:summary` wherever a brief description is useful
- Use `hippo:content` for longer text (notes, decision rationale)
- Link entities to topics with `hippo:hasTopic`
- Keep summaries concise (one sentence)
- For large projects, prioritize the most important 50-100 entities
- Use SPARQL INSERT DATA for bulk operations when adding many triples at once
- Run `validate` after: initial graph population, bulk triple additions (10+), removing resources, or when search returns zero results unexpectedly. Validate now suggests closest type matches for typos (e.g. `hippo:Entiy` → "did you mean hippo:Entity?") — apply the fix commands.
- Run `analyze action=consolidate` after initial population to find and fill gaps (missing summaries, orphaned resources). Use the suggested prompts.
- Use `analyze action=export_html` to get a visualization URL — open it in the browser to review the graph visually.
- Use `analyze action=god_nodes` to identify the most connected resources (hubs) in the graph.
- Use `analyze action=surprising` to find cross-topic connections that may reveal non-obvious relationships.
- Add `hippo:confidence` and `hippo:provenance` when the certainty of extracted facts varies — this helps downstream consumers filter by reliability.
- After analysis, report a summary: number of topics, entities, notes, decisions, questions, and sources indexed

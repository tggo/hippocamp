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
- `rdfs:label` — display name (always set)
- `rdf:type` — classification
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

## Procedure

### Step 1: Setup prefixes and named graph

```
graph action=prefix_add prefix=hippo uri=https://hippocamp.dev/ontology#
graph action=prefix_add prefix=rdfs uri=http://www.w3.org/2000/01/rdf-schema#
graph action=prefix_add prefix=rdf uri=http://www.w3.org/1999/02/22-rdf-syntax-ns#
graph action=prefix_add prefix=proj uri=https://hippocamp.dev/project/
```

Derive the project name from the directory name. Create a named graph:
```
graph action=create name=project:{name}
```

### Step 2: Check for incremental mode

Check if `.claude/.hippocamp-stale` exists. If it does:
- Read the file — it contains paths of files that changed since last analysis
- Only analyze those files (skip full scan)
- For each stale file, remove its old triples: `triple action=list` with the file's URI prefix, then remove matching triples
- Re-analyze only the changed files
- After re-indexing, delete `.claude/.hippocamp-stale`
- Skip to Step 11 (persist)

If `.claude/.hippocamp-stale` does NOT exist, proceed with full analysis below.

### Step 3: Scan the project

Read the directory structure and key files (README, any `.md`, `.txt`, `.csv`, `.json`, `.yaml` files). Identify:
- What is this project about?
- What are the major topic areas? (folders, document sections)
- Who are the key people, organizations, entities?
- What decisions have been made?
- What questions are open?
- What reference materials exist?

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

For each person, organization, product, place, account, or identifiable thing:
```
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=rdf:type object=https://hippocamp.dev/ontology#Entity
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=rdfs:label object="{Entity Name}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=hippo:summary object="{role or description}" object_type=literal
triple action=add graph=project:{name} subject=https://hippocamp.dev/project/{name}/entity/{entity-slug} predicate=hippo:hasTopic object=https://hippocamp.dev/project/{name}/topic/{relevant-topic}
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

### Step 12: Persist

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
- Add `hippo:summary` wherever a brief description is useful
- Use `hippo:content` for longer text (notes, decision rationale)
- Link entities to topics with `hippo:hasTopic`
- Keep summaries concise (one sentence)
- For large projects, prioritize the most important 50-100 entities
- Use SPARQL INSERT DATA for bulk operations when adding many triples at once
- After populating the graph, run `validate` to check for non-standard types, missing labels, and decisions without rationale. Fix any warnings before finishing.
- After analysis, report a summary: number of topics, entities, notes, decisions, questions, and sources indexed

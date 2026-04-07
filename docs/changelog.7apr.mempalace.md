# Changelog — 7 April 2026 (mempalace-inspired)

## Summary

Three features inspired by [mempalace](https://github.com/milla-jovovich/mempalace) — a local-first AI memory system with temporal knowledge graph, progressive context loading, and duplicate detection. Plus a schema migration system added by linter.

---

## 1. Temporal validity (`hippo:validFrom` / `hippo:validTo`)

**Files:** `ontology/hippo.ttl`, `internal/tools/triple.go`, `internal/tools/analyze.go`

**Before:** Facts in the graph were either present or removed. No way to say "this was true from X until Y" without deleting the triple.

**After:** Two new ontology properties and a new `triple invalidate` action:

```
triple action=invalidate subject=http://ex.org/Alice
```

Sets `hippo:validTo=now` and `hippo:status=invalidated` on the subject. The original triples stay for historical queries — nothing is deleted.

**Example flow:**
```
# Alice works at OldCo
triple action=add subject=http://ex.org/Alice predicate=http://ex.org/worksAt object=http://ex.org/OldCo

# She leaves — invalidate, don't delete
triple action=invalidate subject=http://ex.org/Alice

# Later, add new fact
triple action=add subject=http://ex.org/Alice predicate=http://ex.org/worksAt object=http://ex.org/NewCo
```

**Inspired by:** mempalace's `KnowledgeGraph` with `valid_from`/`valid_to` on every triple and explicit `invalidate()` method.

---

## 2. Graph summary action (wake-up context)

**File:** `internal/tools/graph.go`

**New action:** `graph action=summary` — returns a compact JSON overview (~500 tokens) for LLM "wake-up" context:

```json
{
  "graphs": 3,
  "total_triples": 247,
  "invalidated": 2,
  "type_counts": {"Topic": 5, "Entity": 12, "Decision": 3, "Note": 8},
  "topics": ["Authentication", "Database", "Frontend"],
  "top_entities": [
    {"uri": "http://ex.org/auth-service", "label": "Auth Service", "degree": 8},
    {"uri": "http://ex.org/user-db", "label": "User Database", "degree": 6}
  ],
  "decisions": [
    {"uri": "http://ex.org/dec/jwt", "label": "Use JWT tokens"}
  ]
}
```

**Purpose:** An LLM can call this once at the start of a conversation to understand what's already in the graph before asking questions or adding data. Replaces the need to run multiple `graph stats` + `search` + `sparql` queries.

**Inspired by:** mempalace's 4-layer memory stack where L0+L1 (~600 tokens) loads automatically on wake-up, leaving 95%+ context free.

---

## 3. Duplicate detection in triple add

**File:** `internal/tools/triple.go`

**Before:** Adding the same S/P/O triple twice silently created a duplicate (or was a no-op depending on the store, but the tool returned "ok" either way — no feedback to the LLM).

**After:** Before adding, checks if exact S/P/O exists in the target graph. Returns `"duplicate: triple already exists in graph"` instead of silently proceeding.

- Different objects on the same S/P are allowed (multi-valued properties)
- Works per-graph (duplicates in different graphs are fine)

**Inspired by:** mempalace's `mempalace_check_duplicate` tool that checks before filing.

---

## 4. Schema migration system (linter-contributed)

**File:** `internal/tools/migrate.go`

A versioned migration system was added automatically during development:

- `graph action=migrate` applies pending schema migrations
- `validate` warns when migrations are available with a fix command
- V2 migration: adds `hippo:provenance="extracted"` and `hippo:confidence=1.0` to typed resources without them

---

## 5. Skill update

**Files:** `.claude/skills/project-analyze.md`, `internal/setup/embedded/claude/skills/project-analyze.md`

Updated the project-analyze skill with:
- Temporal validity properties documentation
- Step 1.5: "Check existing graph state" via `graph summary` before scanning
- `triple invalidate` usage guidance

---

## Files changed

| File | Change |
|------|--------|
| `ontology/hippo.ttl` | `hippo:validFrom`, `hippo:validTo` properties |
| `internal/tools/triple.go` | `invalidate` action, duplicate detection in `add`, `hippoValidFrom`/`hippoValidTo` constants |
| `internal/tools/graph.go` | `summary` action, `isMetaPredicate()` helper |
| `internal/tools/analyze.go` | `validFrom`/`validTo` added to metadata predicates |
| `internal/tools/migrate.go` | Schema migration system (linter) |
| `internal/tools/temporal_test.go` | 8 tests: invalidate, duplicate detection, graph summary |
| `internal/tools/migrate_test.go` | 4 tests: fresh graph, apply v2, idempotent, validate warning |
| `.claude/skills/project-analyze.md` | Temporal validity, graph summary, step 1.5 |
| `internal/setup/embedded/claude/skills/project-analyze.md` | Synced with above |
| `CLAUDE.md` | Temporal validity, duplicate detection, graph summary sections |

## How to verify

```bash
go test ./... -count=1                    # all packages pass
go test ./internal/tools/ -run TestTripleInvalidate -v   # temporal validity
go test ./internal/tools/ -run TestTripleAdd_Duplicate -v # duplicate detection
go test ./internal/tools/ -run TestGraphSummary -v        # graph summary
go test ./internal/tools/ -run TestMigrate -v             # migrations
```

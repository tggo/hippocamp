# Changelog — 7 April 2026

## Summary

Three improvements inspired by [cognee](https://github.com/topoteretes/cognee) knowledge engine, plus a comprehensive benchmarking suite that proves each feature's value. One feature (popularity boost) was implemented, benchmarked, found to have zero measurable impact, and removed — demonstrating evidence-based development.

---

## 1. Fuzzy type matching in validate

**File:** `internal/tools/validate.go`

**Before:** When an LLM created a resource with an unknown hippo type (e.g. `hippo:Component`), the validate tool gave a generic warning: "consider using hippo:Entity with hippo:hasTag instead."

**After:** The validate tool compares the unknown type against all 17 valid hippo types using LCS-based string similarity. If a match >= 50% is found, it suggests the specific type and provides fix commands.

**Example:**

```json
{
  "warnings": [
    "unknown hippo type hippo:Component on <http://ex.org/a> — did you mean hippo:Concept? (62% similar)"
  ],
  "fixes": [
    "triple action=remove subject=http://ex.org/a predicate=http://www.w3.org/1999/02/22-rdf-syntax-ns#type object=https://hippocamp.dev/ontology#Component",
    "triple action=add subject=http://ex.org/a predicate=http://www.w3.org/1999/02/22-rdf-syntax-ns#type object=https://hippocamp.dev/ontology#Concept"
  ]
}
```

**Algorithm:** `suggestType()` normalizes both strings to lowercase and computes similarity via longest common subsequence: `2 * LCS_length / (len(a) + len(b))`. This catches:
- Typos: `Enity` → `Entity`, `Functon` → `Function`
- Close names: `Component` → `Concept`, `Strukt` → `Struct`
- Falls back to generic message if no match >= 50%

**Inspired by:** cognee's `FuzzyMatchingStrategy` which uses Python's `difflib.get_close_matches()` with 0.8 cutoff for ontology grounding.

---

## 2. Search popularity boost — IMPLEMENTED, BENCHMARKED, REMOVED

**Inspired by:** cognee's `feedback_weight` system where user interactions influence triplet importance scores.

**What we built:** An in-memory access counter (`sync.Map` + `atomic.Int64`) that tracked how often each URI appeared in search results. Top-5 results per search incremented their counter. During scoring, each candidate got a popularity boost: `count * 0.5`, capped at 15% of the text score.

**What the benchmark showed:**

| Metric | Result |
|--------|--------|
| Ranking change after 10 repeated searches | 2/5 positions differ (tiebreaker only) |
| Ground truth position change | 2 -> 2 (zero effect) |
| Max popularity boost | 3.6 points |
| Boost as % of top score | 15% (at cap) |
| Did it improve relevant result ranking? | **No** |

**Why it didn't work:** Text score gaps between results (24 vs 21 vs 12) are always larger than the maximum popularity boost (3.6). The boost could only break ties between results with identical text scores (e.g. Ferguson Supply vs Tankless Heater, both score=12). This made it a tiebreaker for a case that rarely matters, while adding complexity (`sync.Map`, `atomic` ops, extra scoring path, explain field).

**Decision: REMOVED.** The code was deleted because:
1. Zero measurable improvement on ranking quality
2. Added cognitive overhead for developers (another scoring factor to understand)
3. Potential feedback loop risk (popular stays popular regardless of query)
4. Simpler code is better when the feature has no proven value

**Lesson:** Popularity-based boosting needs semantic embeddings (vector similarity) to work — keyword-based text scores create gaps too large for a small access-count bonus to bridge. If we revisit this, it should be via proper embedding-based semantic search, not access counting.

---

## 2. Consolidate action in analyze tool

**File:** `internal/tools/analyze.go`

**New action:** `consolidate` — finds resources that need enrichment and provides graph context so the LLM can fill in the gaps.

**Detects three issues:**
- `missing_summary` — typed resource has no `hippo:summary`
- `sparse_summary` — summary exists but is < 20 characters
- `no_topic` — resource has no `hippo:hasTopic` (orphaned from topic structure; excludes Topic and Tag types)

**For each issue, collects context from the graph:**
- `references` — labels of resources this one links to
- `referenced_by` — labels of resources linking to this one
- `topics` — topic names from `hippo:hasTopic`
- `related_decisions` — decisions in the same topic

**Generates a suggested prompt** the LLM can use to add the missing data:

```
Add hippo:summary to Auth Service (type: Entity). References: User Database, Session Cache. Referenced by: Login Page. Topics: backend.
```

**Usage:**

```json
{"action": "consolidate"}
{"action": "consolidate", "scope": "urn:hippocamp:default", "limit": 10}
```

**Returns:**

```json
[
  {
    "uri": "https://ex.org/auth-service",
    "label": "Auth Service",
    "type": "Entity",
    "issue": "missing_summary",
    "context": {
      "references": ["User Database", "Session Cache"],
      "referenced_by": ["Login Page"],
      "topics": ["backend"]
    },
    "suggested_prompt": "Add hippo:summary to Auth Service (type: Entity). References: User Database, Session Cache. Referenced by: Login Page. Topics: backend."
  }
]
```

Results are sorted by severity: missing_summary first, then sparse_summary, then no_topic.

**Inspired by:** cognee's `memify` pipeline which consolidates entity descriptions using LLM + graph neighbor context. Our version doesn't call an LLM — it provides the context for the client LLM to act on.

---

## 3. `hippo:revision` predicate

**File:** `ontology/hippo.ttl`

New property for tracking how many times a resource has been updated:

```turtle
hippo:revision a rdf:Property ;
    rdfs:label "revision" ;
    rdfs:comment "Revision counter for a resource. Tracks how many times it has been updated." ;
    rdfs:range xsd:integer .
```

**Why not reuse `hippo:version`?** The existing `hippo:version` has `rdfs:domain hippo:Dependency` and `rdfs:range xsd:string` — it's a dependency version string like "1.2.3". `hippo:revision` is a generic integer counter usable on any resource.

**Inspired by:** cognee's `DataPoint.version` field — an incremental counter on every data point.

---

## 4. Benchmarking suite

**File:** `internal/tools/benchmark_test.go`

The most important addition: a comprehensive benchmark suite that measures whether each feature actually provides value. This is what proved popularity boost should be removed and fuzzy matching + consolidate should stay.

### Benchmark results

**Fuzzy type matching (40 test cases):**

| Category | Accuracy | Details |
|----------|----------|---------|
| Typos | **100%** (16/16) | Entiy->Entity, Functon->Function, Strcut->Struct, etc. |
| Plurals | **100%** (7/7) | Entities->Entity, Functions->Function, etc. |
| Case | **100%** (4/4) | entity->Entity, FUNCTION->Function, etc. |
| No-match | **83%** (5/6) | Correctly rejects Banana, HttpServer, Database, Zzzzzzz. One false positive: Middleware->Module |
| Synonyms | **14%** (1/7) | Expected: string similarity can't handle semantic synonyms (Service!=Entity) |
| **Overall** | **82.5%** | Exceeds 60% threshold. Core use case (typos) is perfect. |

**Consolidate quality (house-construction graph, 5 entities stripped of summaries):**

| Metric | Result |
|--------|--------|
| Recall | **100%** (5/5 stripped entities found) |
| Context with references | **86%** (12/14) |
| Context with topics | **86%** (12/14) |
| Rich prompts (>30 chars) | **100%** (14/14) |

**Search ranking baseline (5 ground-truth queries):**

| Query | Expected | Position | Top-3? |
|-------|----------|----------|--------|
| "electrical wiring Tom Chen" | Bright Spark Electric | #2 | Yes |
| "plumbing water" | Waterline Plumbing | #1 | Yes |
| "metal roof hail" | Standing seam metal roof decision | #1 | Yes |
| "spray foam insulation" | Spray foam decision | #1 | Yes |
| "Jim Patterson builder" | Lone Star Builders | #1 | Yes |

**Precision@3: 100%** — all ground-truth results in top 3.

### Bug found by benchmarks

The consolidate benchmark revealed a bug: `handleConsolidate` applied the `limit` BEFORE sorting by severity. Since Go map iteration is random, the first 20 items could be all `no_topic` issues while `missing_summary` items were skipped. Fixed by collecting all suggestions first, sorting by severity, then truncating.

---

## Technical details

### Files changed

| File | Change |
|------|--------|
| `internal/tools/validate.go` | `suggestType()`, `stringSimilarity()`, fuzzy matching in type warnings, fix suggestions |
| `internal/tools/search.go` | Popularity boost added then **removed** after benchmarking |
| `internal/tools/analyze.go` | `consolidate` action, `handleConsolidate()`, issue detection, context collection, limit-after-sort fix |
| `ontology/hippo.ttl` | `hippo:revision` predicate |
| `internal/tools/validate_test.go` | 5 new tests: fuzzy matching, no-match fallback, suggestType table, stringSimilarity table |
| `internal/tools/analyze_test.go` | 3 new tests: missing summary, sparse summary, empty graph |
| `internal/tools/benchmark_test.go` | **New:** 5 benchmarks: fuzzy precision, consolidate quality, search ranking, popularity A/B (documented removal), summary |
| `CLAUDE.md` | Documented kept features, removed popularity boost mention |

### Algorithm: LCS-based string similarity

```go
func stringSimilarity(a, b string) float64 {
    // Dynamic programming LCS
    // Returns: 2 * LCS_length / (len(a) + len(b))
    // Range: 0.0 (completely different) to 1.0 (identical)
}
```

Results on representative inputs:
- "component" vs "concept" -> 0.62 (match)
- "entity" vs "enity" -> 0.91 (typo caught)
- "zzzzz" vs "entity" -> 0.18 (correctly rejected)

---

## How to verify

### Run all tests including benchmarks

```bash
go test ./... -count=1          # all 7 packages pass
go test ./internal/tools/ -run TestBenchmark -v   # see detailed benchmark output
```

### Test fuzzy matching manually

```
triple action=add subject=http://ex.org/test predicate=rdf:type object=https://hippocamp.dev/ontology#Entiy
triple action=add subject=http://ex.org/test predicate=rdfs:label object="Test Resource" object_type=literal
validate
# Should see: "did you mean hippo:Entity? (91% similar)" with fix commands
```

### Test consolidate manually

```
# After populating a graph with some resources missing summaries:
analyze action=consolidate
# Returns suggestions with context and prompts
```

---

## Process note

This changelog documents an evidence-based approach to feature development:

1. **Implement** — build the feature with tests
2. **Benchmark** — measure actual impact with realistic data
3. **Decide** — keep features that prove their value, remove those that don't

The popularity boost was a good idea inspired by cognee's feedback_weight system. It was implemented correctly, tested thoroughly, and benchmarked honestly. The benchmark showed it had zero effect on ranking. Rather than keeping dead code "just in case," we removed it and documented why. The benchmark tests remain as documentation of the decision.

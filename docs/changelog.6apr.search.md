# Changelog — 6 April 2026 (Search Enhancements)

## Summary

Three new capabilities for the `search` tool: confidence normalization (0-100%), temporal search with natural-language date parsing, and an explain mode for score transparency. Inspired by [engraph](https://github.com/devwhodevs/engraph)'s 5-lane hybrid search with RRF fusion — adapted the best ideas for Hippocamp's RDF architecture.

---

## New: Confidence normalization

Every search result now includes two new fields:

- **`score`** (int) — raw text-matching score (sum of field weights, word-boundary bonuses, prefix matches)
- **`confidence`** (float, 0-100%) — normalized score where the top result is 100%, rest proportional

Before:
```json
[{"uri": "...", "label": "Auth Service", "type": "Entity"}]
```

After:
```json
[{"uri": "...", "score": 12, "confidence": 100.0, "label": "Auth Service", "type": "Entity"},
 {"uri": "...", "score": 8, "confidence": 66.7, "label": "Login Page", "type": "Entity"}]
```

**Why it matters:** LLMs can now compare result relevance without understanding our internal scoring system. "100% vs 66.7%" is immediately interpretable; "12 vs 8" is not.

---

## New: Temporal search

Queries with temporal keywords are automatically parsed into date ranges. Resources whose `hippo:createdAt` or `hippo:updatedAt` timestamps fall within the range get a scoring boost.

### Supported temporal patterns

| Pattern | Example query | Parsed range |
|---------|--------------|-------------|
| today | `"decisions today"` | today 00:00–23:59 |
| yesterday | `"notes yesterday"` | yesterday |
| last week | `"changes last week"` | prev Mon–Sun |
| this week | `"work this week"` | current Mon–Sun |
| last month | `"last month summary"` | prev month 1st–last |
| this month | `"this month"` | current month |
| recent/recently | `"recent decisions"` | last 7 days |
| ISO date | `"notes from 2026-03-25"` | that day |
| Month name | `"march notes"` | March (current year) |
| Month + year | `"february 2026"` | Feb 2026 |

### How it works

1. **Parse** — `parseTemporalRange()` scans query for temporal keywords and returns a date range
2. **Strip** — temporal keywords are removed from text matching (so "decisions last week" searches for "decisions")
3. **Score** — resources with timestamps are scored by proximity: 1.0 inside range, smooth decay outside (`1/(1 + days_away * 0.1)`)
4. **Boost** — temporal score adds up to 50% bonus on top of the text score
5. **Fallback** — if few text matches exist, Phase 4 surfaces temporally-matching resources even without text overlap

### Temporal scoring formula

```
Inside range:  1.0
Outside range: 1.0 / (1.0 + days_away * 0.1)

1 day away:    0.909
7 days away:   0.588
30 days away:  0.250
365 days away: 0.027
```

Best of `createdAt` and `updatedAt` is used — a note created in February but updated today will score well for "today" queries.

---

## New: Explain mode

Set `explain=true` to get a per-field score breakdown for each result.

```json
{"query": "authentication", "explain": true}
```

Returns:
```json
[{
  "uri": "...",
  "score": 12,
  "confidence": 100.0,
  "label": "Auth Service",
  "explain": {
    "field_scores": {
      "rdfs:label": 8,
      "hippo:summary": 3,
      "uri": 1
    },
    "temporal_score": 0.91,
    "temporal_range": "today"
  }
}]
```

For related results (via graph traversal):
```json
{
  "explain": {
    "field_scores": {"related": 1},
    "related_from": "https://ex.org/auth-service"
  }
}
```

**Fields:**
- `field_scores` — per-predicate contribution (shortened keys: `rdfs:label`, `hippo:summary`, `uri`, `related`)
- `temporal_score` — 0.0-1.0 temporal proximity (omitted when no temporal range in query)
- `temporal_range` — human description of parsed range (e.g. "last week", "March 2026")
- `related_from` — URI of the direct match this result was linked from (only for graph-traversal results)

When `explain=false` (default), the `explain` field is omitted entirely — no JSON bloat.

---

## Technical details

### Files changed

| File | Change |
|------|--------|
| `internal/tools/search.go` | +temporal parsing, +confidence normalization, +explain mode, +temporal scoring |
| `internal/tools/search_temporal_test.go` | New: 23 unit tests (temporal parsing, scoring, confidence, explain) |
| `internal/tools/testdata_house_test.go` | +7 timestamp triples, +3 temporal query flows |
| `internal/tools/testprojects_test.go` | +tripleCreatedAt, +tripleUpdatedAt helpers |
| `CLAUDE.md` | Documented temporal search, confidence, explain |

### Design choices

- **No SPARQL for temporal queries:** Consistent with existing search — all matching runs in Go for reliability and speed
- **Temporal as a scoring boost, not a filter:** Temporal proximity adds up to 50% bonus rather than filtering out non-matching resources. This preserves recall while improving ranking
- **Smooth decay scoring:** `1/(1 + days * 0.1)` gives gentle preference to nearby dates without cliff edges. Borrowed from engraph's `temporal_score()` which uses the same formula
- **Keyword stripping:** Temporal keywords are removed from text matching so "decisions last week" searches for "decisions" rather than matching on the literal word "last"
- **Phase 4 temporal fallback:** When text matching finds few results, temporal-only matches (score >= 0.5) are surfaced. This handles pure temporal queries like "what happened this month"

### What we took from engraph

| engraph concept | hippocamp adaptation |
|----------------|---------------------|
| Confidence % normalization | Direct port: top = 100%, rest proportional |
| Temporal search lane | Adapted: heuristic date parsing + proximity scoring, same decay formula |
| `--explain` per-lane breakdown | Adapted: `explain=true` parameter with per-field scores |
| 5-lane RRF fusion | Not adopted: our scoring is additive, not RRF. May add later if search quality needs it |
| Embedding/vector search | Not adopted: different paradigm from RDF |

---

## How to verify

### 1. Run tests
```bash
go test ./... -count=1
```
All 7 packages pass, including 23 new search tests and 3 new temporal integration test flows.

### 2. Test confidence normalization
```
Use the search tool: {"query": "authentication"}
```
Results now include `"score": N, "confidence": 100.0` on top result, lower on subsequent.

### 3. Test temporal search
```
Use the search tool: {"query": "decisions today"}
Use the search tool: {"query": "notes last week"}
Use the search tool: {"query": "march 2026"}
```
Resources with `hippo:createdAt`/`hippo:updatedAt` near the date range rank higher.

### 4. Test explain mode
```
Use the search tool: {"query": "auth", "explain": true}
```
Each result includes `explain.field_scores` breakdown. Temporal queries also show `temporal_score` and `temporal_range`.

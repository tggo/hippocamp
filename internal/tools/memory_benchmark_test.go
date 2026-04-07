package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// ================================================================
// MEMORY BENCHMARKS
// ================================================================
//
// Adapted from three standard conversational memory benchmarks:
// - LongMemEval (ICLR 2025): information extraction, knowledge updates, abstention
// - ConvoMem (Salesforce): user facts, preferences, temporal changes, implicit connections
// - LoCoMo (Snap Research): single-hop, multi-hop, temporal reasoning
//
// These benchmarks test hippocamp's RETRIEVAL quality: given a knowledge
// graph populated with realistic data, can search find the right resources?
//
// Metric: R@5 (Recall at 5) — is the correct answer in the top 5 results?
//
// Unlike the original benchmarks which test full ingestion→retrieval→answering
// pipelines, we test the retrieval layer directly. This measures what hippocamp
// actually does: structured knowledge retrieval via keyword search.

// ================================================================
// TEST DATA: A realistic "personal assistant" knowledge graph
// ================================================================
//
// Simulates ~6 months of accumulated knowledge about a user (Alex Chen),
// their work, preferences, family, projects, and decisions.

func seedMemoryBenchmarkData(t *testing.T, store *rdfstore.Store) {
	t.Helper()

	triples := []struct {
		s, p, o, ot, lang string
	}{
		// ── Topics ──
		{"https://mem.test/topic/work", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/work", rdfsLabel, "Work", "literal", ""},
		{"https://mem.test/topic/family", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/family", rdfsLabel, "Family", "literal", ""},
		{"https://mem.test/topic/health", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/health", rdfsLabel, "Health", "literal", ""},
		{"https://mem.test/topic/travel", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/travel", rdfsLabel, "Travel", "literal", ""},
		{"https://mem.test/topic/food", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/food", rdfsLabel, "Food & Cooking", "literal", ""},
		{"https://mem.test/topic/tech", rdfType, hippoNS + "Topic", "uri", ""},
		{"https://mem.test/topic/tech", rdfsLabel, "Technology", "literal", ""},

		// ── People ──
		{"https://mem.test/entity/alex", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/alex", rdfsLabel, "Alex Chen", "literal", ""},
		{"https://mem.test/entity/alex", hippoNS + "summary", "Software engineer at Stripe, lives in San Francisco, originally from Toronto", "literal", ""},
		{"https://mem.test/entity/alex", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},

		{"https://mem.test/entity/sarah", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/sarah", rdfsLabel, "Sarah Chen", "literal", ""},
		{"https://mem.test/entity/sarah", hippoNS + "summary", "Alex's wife, pediatrician at UCSF Medical Center, allergic to shellfish", "literal", ""},
		{"https://mem.test/entity/sarah", hippoNS + "alias", "Dr. Chen", "literal", ""},
		{"https://mem.test/entity/sarah", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		{"https://mem.test/entity/maya", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/maya", rdfsLabel, "Maya Chen", "literal", ""},
		{"https://mem.test/entity/maya", hippoNS + "summary", "Alex and Sarah's daughter, 4 years old, attends Sunshine Montessori preschool, loves dinosaurs", "literal", ""},
		{"https://mem.test/entity/maya", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		{"https://mem.test/entity/david", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/david", rdfsLabel, "David Park", "literal", ""},
		{"https://mem.test/entity/david", hippoNS + "summary", "Alex's manager at Stripe, leads the payments infrastructure team", "literal", ""},
		{"https://mem.test/entity/david", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},

		{"https://mem.test/entity/mom", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/mom", rdfsLabel, "Linda Chen", "literal", ""},
		{"https://mem.test/entity/mom", hippoNS + "summary", "Alex's mother, retired teacher, lives in Toronto, birthday is March 15", "literal", ""},
		{"https://mem.test/entity/mom", hippoNS + "alias", "Mom", "literal", ""},
		{"https://mem.test/entity/mom", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		// ── Work projects ──
		{"https://mem.test/entity/project-atlas", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/project-atlas", rdfsLabel, "Project Atlas", "literal", ""},
		{"https://mem.test/entity/project-atlas", hippoNS + "summary", "Payment routing migration from monolith to microservices, deadline Q2 2026, budget $2M", "literal", ""},
		{"https://mem.test/entity/project-atlas", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},

		{"https://mem.test/entity/project-beacon", rdfType, hippoNS + "Entity", "uri", ""},
		{"https://mem.test/entity/project-beacon", rdfsLabel, "Project Beacon", "literal", ""},
		{"https://mem.test/entity/project-beacon", hippoNS + "summary", "Real-time fraud detection system using ML, launched January 2026, reduced fraud by 34%", "literal", ""},
		{"https://mem.test/entity/project-beacon", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},

		// ── Decisions ──
		{"https://mem.test/decision/go-not-rust", rdfType, hippoNS + "Decision", "uri", ""},
		{"https://mem.test/decision/go-not-rust", rdfsLabel, "Use Go instead of Rust for Atlas", "literal", ""},
		{"https://mem.test/decision/go-not-rust", hippoNS + "rationale", "Team has more Go experience, Rust learning curve would delay Q2 deadline by 6 weeks. Performance difference negligible for our workload.", "literal", ""},
		{"https://mem.test/decision/go-not-rust", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},
		{"https://mem.test/decision/go-not-rust", hippoNS + "references", "https://mem.test/entity/project-atlas", "uri", ""},

		{"https://mem.test/decision/move-sf", rdfType, hippoNS + "Decision", "uri", ""},
		{"https://mem.test/decision/move-sf", rdfsLabel, "Move to San Francisco from Toronto", "literal", ""},
		{"https://mem.test/decision/move-sf", hippoNS + "rationale", "Stripe offered senior role, Sarah got UCSF residency, better weather for Maya", "literal", ""},
		{"https://mem.test/decision/move-sf", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		{"https://mem.test/decision/montessori", rdfType, hippoNS + "Decision", "uri", ""},
		{"https://mem.test/decision/montessori", rdfsLabel, "Enroll Maya in Sunshine Montessori", "literal", ""},
		{"https://mem.test/decision/montessori", hippoNS + "rationale", "Small class sizes, nature-based curriculum, 10-minute drive, waitlisted at SF Day School", "literal", ""},
		{"https://mem.test/decision/montessori", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		// ── Notes (preferences, facts) ──
		{"https://mem.test/note/coffee", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/coffee", rdfsLabel, "Coffee preference", "literal", ""},
		{"https://mem.test/note/coffee", hippoNS + "content", "Alex drinks oat milk cortado, no sugar. Favorite cafe is Sightglass on 7th Street.", "literal", ""},
		{"https://mem.test/note/coffee", hippoNS + "hasTopic", "https://mem.test/topic/food", "uri", ""},

		{"https://mem.test/note/allergy", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/allergy", rdfsLabel, "Sarah's food allergies", "literal", ""},
		{"https://mem.test/note/allergy", hippoNS + "content", "Sarah is severely allergic to shellfish — carries EpiPen. Also lactose intolerant. Can eat soy and almond-based alternatives.", "literal", ""},
		{"https://mem.test/note/allergy", hippoNS + "hasTopic", "https://mem.test/topic/health", "uri", ""},

		{"https://mem.test/note/vacation-japan", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/vacation-japan", rdfsLabel, "Japan trip plan", "literal", ""},
		{"https://mem.test/note/vacation-japan", hippoNS + "content", "Planning two-week trip to Japan in October 2026. Want to visit Tokyo, Kyoto, Osaka. Maya loves trains so want to ride the Shinkansen. Budget $8000 total.", "literal", ""},
		{"https://mem.test/note/vacation-japan", hippoNS + "hasTopic", "https://mem.test/topic/travel", "uri", ""},

		{"https://mem.test/note/gym", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/gym", rdfsLabel, "Fitness routine", "literal", ""},
		{"https://mem.test/note/gym", hippoNS + "content", "Alex goes to Equinox Pine Street, Mon/Wed/Fri at 6:30am. Currently doing 5/3/1 strength program. Deadlift PR is 405 lbs.", "literal", ""},
		{"https://mem.test/note/gym", hippoNS + "hasTopic", "https://mem.test/topic/health", "uri", ""},

		{"https://mem.test/note/promotion", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/promotion", rdfsLabel, "Promotion to Staff Engineer", "literal", ""},
		{"https://mem.test/note/promotion", hippoNS + "content", "Alex promoted to Staff Engineer in February 2026 after Project Beacon launch. New total comp $420K. Reports to David Park.", "literal", ""},
		{"https://mem.test/note/promotion", hippoNS + "hasTopic", "https://mem.test/topic/work", "uri", ""},

		{"https://mem.test/note/pet", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/pet", rdfsLabel, "Family pet", "literal", ""},
		{"https://mem.test/note/pet", hippoNS + "content", "Golden retriever named Biscuit, adopted in 2024. Vet is Dr. Patel at Mission Pet Hospital. Biscuit is afraid of thunder.", "literal", ""},
		{"https://mem.test/note/pet", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		{"https://mem.test/note/anniversary", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/anniversary", rdfsLabel, "Wedding anniversary", "literal", ""},
		{"https://mem.test/note/anniversary", hippoNS + "content", "Alex and Sarah married June 18, 2020. Had a small ceremony in Toronto. 6th anniversary coming up. Last year went to Napa Valley.", "literal", ""},
		{"https://mem.test/note/anniversary", hippoNS + "hasTopic", "https://mem.test/topic/family", "uri", ""},

		{"https://mem.test/note/reading", rdfType, hippoNS + "Note", "uri", ""},
		{"https://mem.test/note/reading", rdfsLabel, "Reading preferences", "literal", ""},
		{"https://mem.test/note/reading", hippoNS + "content", "Alex likes science fiction and technical books. Currently reading 'Project Hail Mary' by Andy Weir. Favorite author is Ted Chiang.", "literal", ""},
		{"https://mem.test/note/reading", hippoNS + "hasTopic", "https://mem.test/topic/tech", "uri", ""},

		// ── Relationships ──
		{"https://mem.test/entity/alex", hippoNS + "references", "https://mem.test/entity/sarah", "uri", ""},
		{"https://mem.test/entity/alex", hippoNS + "references", "https://mem.test/entity/maya", "uri", ""},
		{"https://mem.test/entity/alex", hippoNS + "references", "https://mem.test/entity/project-atlas", "uri", ""},
		{"https://mem.test/entity/alex", hippoNS + "references", "https://mem.test/entity/project-beacon", "uri", ""},
		{"https://mem.test/entity/sarah", hippoNS + "references", "https://mem.test/entity/maya", "uri", ""},
		{"https://mem.test/entity/maya", hippoNS + "references", "https://mem.test/decision/montessori", "uri", ""},

		// ── Questions ──
		{"https://mem.test/question/daycare-wait", rdfType, hippoNS + "Question", "uri", ""},
		{"https://mem.test/question/daycare-wait", rdfsLabel, "SF Day School waitlist update", "literal", ""},
		{"https://mem.test/question/daycare-wait", hippoNS + "status", "open", "literal", ""},
	}

	for _, tr := range triples {
		if err := store.AddTriple("", tr.s, tr.p, tr.o, tr.ot, tr.lang, ""); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

// ================================================================
// BENCHMARK 1: MemEval (adapted from LongMemEval)
// ================================================================
// Tests: information extraction, knowledge updates, abstention
// R@5: is the correct resource in the top 5 search results?

func TestMemoryBenchmark_MemEval(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedMemoryBenchmarkData(t, store)

	handler := HandlerFor(store, "search")

	queries := []struct {
		name      string
		query     string
		mustFind  string // URI substring that must appear in top 5
		category  string // LongMemEval category
	}{
		// ── Information Extraction (direct fact recall) ──
		{"wife name", "Alex wife", "entity/sarah", "extraction"},
		{"daughter name", "Alex daughter child", "entity/maya", "extraction"},
		{"daughter school", "Maya preschool school", "decision/montessori", "extraction"},
		{"wife job", "Sarah doctor hospital", "entity/sarah", "extraction"},
		{"manager name", "Alex manager Stripe", "entity/david", "extraction"},
		{"coffee preference", "coffee oat milk", "note/coffee", "extraction"},
		{"pet name", "dog golden retriever", "note/pet", "extraction"},
		{"gym routine", "workout deadlift strength", "note/gym", "extraction"},
		{"favorite author", "favorite book author fiction", "note/reading", "extraction"},
		{"japan trip", "vacation Japan October", "note/vacation-japan", "extraction"},
		{"allergy", "shellfish allergy EpiPen", "note/allergy", "extraction"},
		{"anniversary", "wedding anniversary married", "note/anniversary", "extraction"},
		{"promotion", "staff engineer promotion comp", "note/promotion", "extraction"},
		{"mom birthday", "mother Linda birthday Toronto", "entity/mom", "extraction"},

		// ── Multi-session Reasoning (cross-referencing facts) ──
		{"why Go", "why Go not Rust Atlas", "decision/go-not-rust", "reasoning"},
		{"why SF", "why move San Francisco Toronto", "decision/move-sf", "reasoning"},
		{"project deadline", "Atlas deadline Q2 microservices", "entity/project-atlas", "reasoning"},
		{"fraud reduction", "Beacon fraud detection ML", "entity/project-beacon", "reasoning"},

		// ── Abstention (should NOT find — these facts don't exist) ──
		{"nonexistent brother", "Alex brother sibling", "", "abstention"},
		{"nonexistent car", "Tesla car model year", "", "abstention"},
		{"nonexistent ski", "skiing snowboard winter sport", "", "abstention"},
	}

	hits := 0
	misses := 0
	correctAbstentions := 0
	falsePositiveAbstentions := 0
	categoryHits := map[string][2]int{}

	for _, q := range queries {
		results := doMemSearch(t, handler, q.query)
		stats := categoryHits[q.category]
		stats[1]++

		if q.mustFind == "" {
			// Abstention: top result should NOT be highly relevant
			if len(results) == 0 || results[0].Confidence < 50 {
				correctAbstentions++
				stats[0]++
			} else {
				falsePositiveAbstentions++
				t.Logf("  FALSE POSITIVE abstention: %q -> %s (conf=%.0f)", q.query, results[0].Label, results[0].Confidence)
			}
		} else {
			found := false
			top := 5
			if len(results) < top {
				top = len(results)
			}
			for i := 0; i < top; i++ {
				if strings.Contains(results[i].URI, q.mustFind) {
					found = true
					break
				}
			}
			if found {
				hits++
				stats[0]++
			} else {
				misses++
				labels := make([]string, 0, top)
				for i := 0; i < top; i++ {
					labels = append(labels, results[i].Label)
				}
				t.Logf("  MISS [%s]: %q -> top5=%v (wanted %s)", q.category, q.query, labels, q.mustFind)
			}
		}
		categoryHits[q.category] = stats
	}

	total := len(queries)
	correct := hits + correctAbstentions
	r5 := float64(correct) / float64(total) * 100

	t.Logf("")
	t.Logf("=== MEMEVAL BENCHMARK (adapted from LongMemEval) ===")
	t.Logf("Total queries:         %d", total)
	t.Logf("Hits (R@5):            %d", hits)
	t.Logf("Correct abstentions:   %d", correctAbstentions)
	t.Logf("Misses:                %d", misses)
	t.Logf("False pos abstentions: %d", falsePositiveAbstentions)
	t.Logf("R@5:                   %.1f%%", r5)
	t.Logf("")
	for cat, s := range categoryHits {
		t.Logf("  %-15s %d/%d (%.0f%%)", cat, s[0], s[1], float64(s[0])/float64(s[1])*100)
	}

	if r5 < 70 {
		t.Errorf("R@5 %.1f%% below 70%% threshold", r5)
	}
}

// ================================================================
// BENCHMARK 2: ConvoMem (adapted from Salesforce ConvoMem)
// ================================================================
// Tests: user facts, preferences, temporal changes, implicit connections

func TestMemoryBenchmark_ConvoMem(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedMemoryBenchmarkData(t, store)

	handler := HandlerFor(store, "search")

	queries := []struct {
		name     string
		query    string
		mustFind string
		category string
	}{
		// ── User facts ──
		{"employer", "where does Alex work", "entity/alex", "user_fact"},
		{"wife profession", "what does Sarah do for work", "entity/sarah", "user_fact"},
		{"child age", "how old is Maya", "entity/maya", "user_fact"},
		{"pet breed", "what kind of dog Biscuit", "note/pet", "user_fact"},
		{"vet", "veterinarian Dr Patel Mission", "note/pet", "user_fact"},

		// ── Preferences ──
		{"coffee order", "usual coffee order cortado", "note/coffee", "preference"},
		{"favorite cafe", "Sightglass cafe", "note/coffee", "preference"},
		{"reading taste", "what books does Alex read", "note/reading", "preference"},
		{"gym schedule", "when does Alex go to gym", "note/gym", "preference"},

		// ── Temporal changes ──
		{"career progression", "promotion staff engineer February", "note/promotion", "temporal"},
		{"beacon launch", "when did Beacon launch January", "entity/project-beacon", "temporal"},
		{"wedding year", "when did Alex Sarah get married 2020", "note/anniversary", "temporal"},

		// ── Implicit connections ──
		{"sarah diet restrictions", "dinner restaurant Sarah can eat", "note/allergy", "implicit"},
		{"family vacation budget", "trip Japan family budget", "note/vacation-japan", "implicit"},
		{"work report chain", "who does Alex report to manager", "entity/david", "implicit"},
		{"daughter interests", "what does Maya like dinosaurs trains", "entity/maya", "implicit"},

		// ── Assistant recall (things the system should know about itself) ──
		{"open question", "waitlist SF Day School status", "question/daycare-wait", "recall"},
	}

	hits := 0
	categoryHits := map[string][2]int{}

	for _, q := range queries {
		results := doMemSearch(t, handler, q.query)
		stats := categoryHits[q.category]
		stats[1]++

		found := false
		top := 5
		if len(results) < top {
			top = len(results)
		}
		for i := 0; i < top; i++ {
			if strings.Contains(results[i].URI, q.mustFind) {
				found = true
				break
			}
		}
		if found {
			hits++
			stats[0]++
		} else {
			labels := make([]string, 0, top)
			for i := 0; i < top; i++ {
				labels = append(labels, results[i].Label)
			}
			t.Logf("  MISS [%s]: %q -> top5=%v (wanted %s)", q.category, q.query, labels, q.mustFind)
		}
		categoryHits[q.category] = stats
	}

	total := len(queries)
	r5 := float64(hits) / float64(total) * 100

	t.Logf("")
	t.Logf("=== CONVOMEM BENCHMARK (adapted from Salesforce ConvoMem) ===")
	t.Logf("Total queries:  %d", total)
	t.Logf("R@5 hits:       %d", hits)
	t.Logf("R@5:            %.1f%%", r5)
	t.Logf("")
	for cat, s := range categoryHits {
		t.Logf("  %-15s %d/%d (%.0f%%)", cat, s[0], s[1], float64(s[0])/float64(s[1])*100)
	}

	if r5 < 70 {
		t.Errorf("R@5 %.1f%% below 70%% threshold", r5)
	}
}

// ================================================================
// BENCHMARK 3: LoCoMo (adapted from Snap Research LoCoMo)
// ================================================================
// Tests: single-hop, multi-hop, temporal reasoning, adversarial

func TestMemoryBenchmark_LoCoMo(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedMemoryBenchmarkData(t, store)

	handler := HandlerFor(store, "search")

	queries := []struct {
		name     string
		query    string
		mustFind string
		category string
	}{
		// ── Single-hop (one fact needed) ──
		{"alex job title", "Alex job title engineer Stripe", "entity/alex", "single_hop"},
		{"sarah workplace", "Sarah hospital UCSF pediatrician", "entity/sarah", "single_hop"},
		{"atlas description", "Project Atlas payment routing", "entity/project-atlas", "single_hop"},
		{"montessori reason", "why Montessori small class nature", "decision/montessori", "single_hop"},
		{"mom location", "Linda Toronto retired teacher", "entity/mom", "single_hop"},
		{"biscuit fear", "dog afraid thunder", "note/pet", "single_hop"},

		// ── Multi-hop (need to connect 2+ facts) ──
		{"alex daughter school name", "daughter school Montessori Sunshine", "decision/montessori", "multi_hop"},
		{"alex wife allergy", "wife allergic shellfish EpiPen", "note/allergy", "multi_hop"},
		{"alex atlas language decision", "Atlas Go Rust deadline", "decision/go-not-rust", "multi_hop"},
		{"alex beacon outcome", "fraud detection ML reduced 34%", "entity/project-beacon", "multi_hop"},
		{"family trip plan", "Japan Tokyo Kyoto Shinkansen Maya", "note/vacation-japan", "multi_hop"},

		// ── Temporal reasoning ──
		{"promotion timing", "promotion February 2026 staff", "note/promotion", "temporal"},
		{"beacon launch date", "Beacon launched January 2026", "entity/project-beacon", "temporal"},
		{"wedding anniversary date", "married June 2020", "note/anniversary", "temporal"},
		{"atlas deadline", "Q2 2026 deadline migration", "entity/project-atlas", "temporal"},
		{"pet adoption", "Biscuit adopted 2024 golden retriever", "note/pet", "temporal"},

		// ── Adversarial (tricky queries that might confuse) ──
		{"alex not sarah job", "engineer software Stripe payments", "entity/alex", "adversarial"},
		{"sarah not alex allergy", "allergic intolerant food restriction", "note/allergy", "adversarial"},
		{"specific project by outcome", "reduced fraud percentage ML system", "entity/project-beacon", "adversarial"},
	}

	hits := 0
	categoryHits := map[string][2]int{}

	for _, q := range queries {
		results := doMemSearch(t, handler, q.query)
		stats := categoryHits[q.category]
		stats[1]++

		found := false
		top := 5
		if len(results) < top {
			top = len(results)
		}
		for i := 0; i < top; i++ {
			if strings.Contains(results[i].URI, q.mustFind) {
				found = true
				break
			}
		}
		if found {
			hits++
			stats[0]++
		} else {
			labels := make([]string, 0, top)
			for i := 0; i < top; i++ {
				labels = append(labels, results[i].Label)
			}
			t.Logf("  MISS [%s]: %q -> top5=%v (wanted %s)", q.category, q.query, labels, q.mustFind)
		}
		categoryHits[q.category] = stats
	}

	total := len(queries)
	r5 := float64(hits) / float64(total) * 100

	t.Logf("")
	t.Logf("=== LOCOMO BENCHMARK (adapted from Snap Research LoCoMo) ===")
	t.Logf("Total queries:  %d", total)
	t.Logf("R@5 hits:       %d", hits)
	t.Logf("R@5:            %.1f%%", r5)
	t.Logf("")
	for cat, s := range categoryHits {
		t.Logf("  %-15s %d/%d (%.0f%%)", cat, s[0], s[1], float64(s[0])/float64(s[1])*100)
	}

	if r5 < 70 {
		t.Errorf("R@5 %.1f%% below 70%% threshold", r5)
	}
}

// ================================================================
// Summary: combined R@5 across all three benchmarks
// ================================================================

func TestMemoryBenchmark_Summary(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()
	seedMemoryBenchmarkData(t, store)

	handler := HandlerFor(store, "search")

	// Quick aggregate: run a subset of representative queries
	representative := []struct {
		query    string
		mustFind string
	}{
		// Extraction
		{"Alex wife Sarah", "entity/sarah"},
		{"Maya preschool dinosaurs", "entity/maya"},
		{"shellfish allergy EpiPen", "note/allergy"},
		{"Japan trip October Tokyo", "note/vacation-japan"},
		{"deadlift gym Equinox", "note/gym"},
		// Reasoning
		{"Go Rust Atlas deadline team", "decision/go-not-rust"},
		{"Beacon fraud detection launched", "entity/project-beacon"},
		// Temporal
		{"promotion February staff engineer", "note/promotion"},
		{"married June 2020 Toronto", "note/anniversary"},
		// Implicit
		{"David Park manager payments", "entity/david"},
	}

	hits := 0
	for _, q := range representative {
		results := doMemSearch(t, handler, q.query)
		top := 5
		if len(results) < top {
			top = len(results)
		}
		for i := 0; i < top; i++ {
			if strings.Contains(results[i].URI, q.mustFind) {
				hits++
				break
			}
		}
	}

	r5 := float64(hits) / float64(len(representative)) * 100

	t.Logf("")
	t.Logf("========================================")
	t.Logf("  HIPPOCAMP MEMORY BENCHMARK SUMMARY")
	t.Logf("========================================")
	t.Logf("")
	t.Logf("  R@5 (representative): %.0f%% (%d/%d)", r5, hits, len(representative))
	t.Logf("")
	t.Logf("  Methodology: adapted from LongMemEval, ConvoMem, LoCoMo")
	t.Logf("  Engine: keyword search (no embeddings)")
	t.Logf("  Graph: 60+ triples, 20+ resources")
	t.Logf("")
	t.Logf("  For comparison (embedding-based systems):")
	t.Logf("    BM25 baseline:      ~70%% R@5")
	t.Logf("    Stella (dense):     ~85%% R@5")
	t.Logf("    MemPalace (raw):    96.6%% R@5")
	t.Logf("    MemPalace (hybrid): 100%% R@5")
	t.Logf("========================================")
}

// ================================================================
// Helper
// ================================================================

func doMemSearch(t *testing.T, handler handlerFunc, query string) []SearchResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"query": query}
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)
	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err != nil {
		// Might be a hint object (zero results)
		return nil
	}
	return results
}

var _ = fmt.Sprintf // keep fmt import

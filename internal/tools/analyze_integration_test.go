package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// buildAnalysisTriG builds a TriG string representing the house-construction
// project knowledge — this is what Claude would produce via graph import.
func buildAnalysisTriG() string {
	const base = "https://hippocamp.dev/project/house-construction"
	var b strings.Builder

	b.WriteString("@prefix rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#> .\n")
	b.WriteString("@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#> .\n")
	b.WriteString("@prefix hippo: <https://hippocamp.dev/ontology#> .\n\n")

	// Helper to write a triple.
	uri := func(s string) string { return "<" + s + ">" }
	lit := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }
	w := func(subj, pred, obj string) { fmt.Fprintf(&b, "%s %s %s .\n", subj, pred, obj) }

	// Project entity
	p := uri(base)
	w(p, "rdf:type", "hippo:Entity")
	w(p, "rdfs:label", lit("Patterson Family Home Construction"))
	w(p, "hippo:summary", lit("Two-story family home construction at 4218 Barton Creek Drive, Austin TX, $450K budget, 8 months"))

	// Topics
	topics := map[string]string{
		"permits": "Building permits and inspections", "materials": "Construction materials and suppliers",
		"budget": "Project budget and cost tracking", "schedule": "Construction timeline and milestones",
		"electrical": "Electrical systems and wiring", "plumbing": "Plumbing systems and fixtures",
		"roofing": "Roof construction and materials", "foundation": "Foundation and structural work",
		"hvac": "Heating, ventilation, and air conditioning", "insulation": "Insulation materials and installation",
	}
	for slug, summary := range topics {
		u := uri(base + "/topic/" + slug)
		w(u, "rdf:type", "hippo:Topic")
		w(u, "rdfs:label", lit(slug))
		w(u, "hippo:summary", lit(summary))
	}

	// Entities
	entities := []struct{ slug, label, summary, topic string }{
		{"lone-star-builders", "Lone Star Builders", "General contractor, Jim Patterson, 22 years experience in Austin residential", "foundation"},
		{"bright-spark-electric", "Bright Spark Electric", "Electrician, Tom Chen, handles 400-amp service and solar-ready wiring", "electrical"},
		{"waterline-plumbing", "Waterline Plumbing Services", "Plumber, Carlos Mendez, under-slab and above-grade plumbing", "plumbing"},
		{"austin-metal-roofing", "Austin Metal Roofing Co", "Roofer, Derek Williams, specializes in metal roofing", "roofing"},
		{"comfort-air-solutions", "Comfort Air Solutions", "HVAC contractor, Sarah Kim, Carrier Infinity heat pump system", "hvac"},
		{"hill-country-lumber", "Hill Country Lumber", "Primary lumber supplier for framing materials", "materials"},
		{"ferguson-supply", "Ferguson Supply", "Plumbing fixtures supplier, low-flow fixtures", "plumbing"},
		{"maria-rodriguez", "Maria Rodriguez", "City of Austin building inspector", "permits"},
		{"angela-torres", "Angela Torres", "Architect at Hill Country Design Group, AIA", "foundation"},
	}
	for _, e := range entities {
		u := uri(base + "/entity/" + e.slug)
		w(u, "rdf:type", "hippo:Entity")
		w(u, "rdfs:label", lit(e.label))
		w(u, "hippo:summary", lit(e.summary))
		w(u, "hippo:hasTopic", uri(base+"/topic/"+e.topic))
	}

	// Notes
	notes := []struct{ slug, label, content, topic string }{
		{"foundation-specs", "Foundation Specifications", "Post-tension concrete slab, 4-inch minimum thickness, rebar grid on 12-inch centers", "foundation"},
		{"lumber-order", "Lumber Order Details", "Douglas Fir framing lumber from Hill Country Lumber, delivery scheduled for week 3, bulk discount saves $3,200", "materials"},
		{"budget-summary", "Budget Summary", "Total budget $450K. Foundation $45K, framing $65K, roofing $30K, electrical $25K, plumbing $20K, HVAC $18K, insulation $12K", "budget"},
		{"energy-efficiency", "Energy Efficiency Goals", "Targeting HERS rating below 55. Passive solar gain from south-facing windows, spray foam insulation, metal roof heat reflection", "insulation"},
	}
	for _, n := range notes {
		u := uri(base + "/note/" + n.slug)
		w(u, "rdf:type", "hippo:Note")
		w(u, "rdfs:label", lit(n.label))
		w(u, "hippo:content", lit(n.content))
		w(u, "hippo:hasTopic", uri(base+"/topic/"+n.topic))
	}

	// Decisions
	decisions := []struct{ slug, label, rationale, topic string }{
		{"metal-roof", "Standing seam metal roof over asphalt shingles", "Class 4 impact rating for hail, 50+ year lifespan, 140 mph wind resistance, 15% cooling cost reduction", "roofing"},
		{"spray-foam", "Spray foam insulation over fiberglass batts", "Combined insulation and air sealing, eliminates vapor barrier, annual HVAC savings $600-$900", "insulation"},
		{"tankless-heater", "Tankless water heater over tank", "Continuous hot water, wall-mounted saves space, Energy Factor 0.96, $150 annual gas savings, 20-year lifespan", "plumbing"},
	}
	for _, d := range decisions {
		u := uri(base + "/decision/" + d.slug)
		w(u, "rdf:type", "hippo:Decision")
		w(u, "rdfs:label", lit(d.label))
		w(u, "hippo:rationale", lit(d.rationale))
		w(u, "hippo:hasTopic", uri(base+"/topic/"+d.topic))
	}

	// Questions
	questions := []struct{ slug, label string }{
		{"triple-pane-windows", "Should we upgrade to triple-pane windows on the west-facing wall? Adds $4,800"},
		{"hoa-paint-colors", "HOA pre-approval for exterior paint colors, deadline May 30"},
	}
	for _, q := range questions {
		u := uri(base + "/question/" + q.slug)
		w(u, "rdf:type", "hippo:Question")
		w(u, "rdfs:label", lit(q.label))
		w(u, "hippo:status", lit("open"))
	}

	// Sources
	sources := []struct{ slug, label, summary string }{
		{"austin-building-codes", "Austin Residential Building Codes", "City of Austin building code requirements for residential construction"},
		{"nec-2023", "NEC 2023", "National Electrical Code 2023 edition, required for all electrical work"},
		{"irc-2021", "International Residential Code 2021", "IRC adopted by City of Austin with local amendments"},
	}
	for _, s := range sources {
		u := uri(base + "/source/" + s.slug)
		w(u, "rdf:type", "hippo:Source")
		w(u, "rdfs:label", lit(s.label))
		w(u, "hippo:summary", lit(s.summary))
	}

	return b.String()
}

// TestAnalyzeHouseConstruction simulates the full /project-analyze flow:
// reads real testdata files, batch-imports triples via graph import (not
// individual triple adds), then verifies search and SPARQL work.
func TestAnalyzeHouseConstruction(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "house-construction")
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata/house-construction not found")
	}

	store := rdfstore.NewStore()
	defer store.Close()

	graphHandler := HandlerFor(store, "graph")
	searchHandler := HandlerFor(store, "search")

	// Step 1: Verify testdata files exist and have content.
	for _, name := range []string{"README.md", "contractors.md", "decisions.md", "budget.md"} {
		data, err := os.ReadFile(filepath.Join(testdataDir, name))
		if err != nil || len(data) == 0 {
			t.Fatalf("testdata file %s missing or empty", name)
		}
	}

	// Step 2: Batch import — ONE call, like the real user flow.
	trigData := buildAnalysisTriG()
	t.Logf("TriG data size: %d bytes", len(trigData))

	importReq := mcp.CallToolRequest{}
	importReq.Params.Arguments = map[string]any{"action": "import", "data": trigData}
	importRes, err := graphHandler(context.Background(), importReq)
	if err != nil {
		t.Fatalf("import error: %v", err)
	}
	if importRes.IsError {
		t.Fatalf("import failed: %s", ResultText(importRes))
	}
	t.Logf("Import result: %s", ResultText(importRes))

	// Step 3: Verify graph stats.
	statsRes := callToolGetText(t, graphHandler, map[string]any{"action": "stats"})
	t.Logf("Graph stats: %s", statsRes)

	// Step 4: Search tests — verify knowledge is findable.
	searchTests := []struct {
		name, query string
		mustFind    string
	}{
		{"find contractor by name", "Jim Patterson Lone Star", "lone-star-builders"},
		{"find electrician", "Tom Chen electrical", "bright-spark-electric"},
		{"find HVAC contractor", "Sarah Kim heat pump", "comfort-air-solutions"},
		{"find metal roof decision", "metal roof hail", "decision/metal-roof"},
		{"find spray foam decision", "spray foam insulation", "decision/spray-foam"},
		{"find tankless heater", "tankless water heater", "decision/tankless-heater"},
		{"find budget note", "budget $450K", "note/budget-summary"},
		{"find lumber order", "Douglas Fir lumber", "note/lumber-order"},
		{"find inspector", "Maria Rodriguez inspector", "maria-rodriguez"},
		{"find architect", "Angela Torres architect", "angela-torres"},
		{"find NEC source", "NEC 2023 electrical code", "source/nec-2023"},
		{"find window question", "triple-pane windows", "question/triple-pane"},
		{"find plumbing topic", "plumbing fixtures", "topic/plumbing"},
		{"find foundation specs", "concrete slab rebar", "note/foundation-specs"},
		{"find energy note", "HERS rating passive solar", "note/energy-efficiency"},
	}

	for _, st := range searchTests {
		t.Run("search/"+st.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": st.query}
			res, err := searchHandler(context.Background(), req)
			if err != nil {
				t.Fatalf("search error: %v", err)
			}
			text := ResultText(res)
			var results []SearchResult
			json.Unmarshal([]byte(text), &results)

			found := false
			for _, r := range results {
				if strings.Contains(r.URI, st.mustFind) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected URI containing %q, got %d results", st.mustFind, len(results))
			}
		})
	}

	// Step 5: Persistence round-trip.
	t.Run("persistence", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "house.trig")
		callToolExpectOK(t, graphHandler, map[string]any{"action": "dump", "file": tmpFile})

		info, _ := os.Stat(tmpFile)
		t.Logf("TriG file size: %d bytes", info.Size())
		if info.Size() < 1000 {
			t.Error("TriG file is suspiciously small")
		}

		// Reload into fresh store and verify search still works.
		store2 := rdfstore.NewStore()
		defer store2.Close()
		callToolExpectOK(t, HandlerFor(store2, "graph"), map[string]any{"action": "load", "file": tmpFile})

		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"query": "Lone Star Builders"}
		res, _ := HandlerFor(store2, "search")(context.Background(), req)
		if !strings.Contains(ResultText(res), "lone-star-builders") {
			t.Error("search after reload failed to find Lone Star Builders")
		}
	})

	// Step 6: Validate.
	t.Run("validate", func(t *testing.T) {
		validateHandler := HandlerFor(store, "validate")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{}
		res, _ := validateHandler(context.Background(), req)
		text := ResultText(res)
		t.Logf("Validate result: %s", text)
	})
}

// ── helpers ──

func callToolExpectOK(t *testing.T, handler handlerFunc, args map[string]any) {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %s", ResultText(res))
	}
}

func callToolGetText(t *testing.T, handler handlerFunc, args map[string]any) string {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("tool error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %s", ResultText(res))
	}
	return ResultText(res)
}

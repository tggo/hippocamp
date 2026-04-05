package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// TestAnalyzeHouseConstruction simulates the full /project-analyze flow
// against the house-construction testdata: reads real markdown files,
// populates the graph via the MCP tools, then verifies search results.
func TestAnalyzeHouseConstruction(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata", "house-construction")
	if _, err := os.Stat(testdataDir); os.IsNotExist(err) {
		t.Skip("testdata/house-construction not found")
	}

	store := rdfstore.NewStore()
	defer store.Close()

	graphHandler := HandlerFor(store, "graph")
	tripleHandler := HandlerFor(store, "triple")
	searchHandler := HandlerFor(store, "search")
	sparqlHandler := HandlerFor(store, "sparql")

	// Step 1: Setup prefixes and named graph (as the skill instructs).
	callToolExpectOK(t, graphHandler, map[string]any{"action": "prefix_add", "prefix": "hippo", "uri": "https://hippocamp.dev/ontology#"})
	callToolExpectOK(t, graphHandler, map[string]any{"action": "prefix_add", "prefix": "rdfs", "uri": "http://www.w3.org/2000/01/rdf-schema#"})
	callToolExpectOK(t, graphHandler, map[string]any{"action": "prefix_add", "prefix": "rdf", "uri": "http://www.w3.org/1999/02/22-rdf-syntax-ns#"})
	callToolExpectOK(t, graphHandler, map[string]any{"action": "create", "name": "project:house-construction"})

	const graph = "project:house-construction"
	const base = "https://hippocamp.dev/project/house-construction"

	// Step 2: Read testdata files to understand the project.
	readme, _ := os.ReadFile(filepath.Join(testdataDir, "README.md"))
	contractors, _ := os.ReadFile(filepath.Join(testdataDir, "contractors.md"))
	decisions, _ := os.ReadFile(filepath.Join(testdataDir, "decisions.md"))
	budget, _ := os.ReadFile(filepath.Join(testdataDir, "budget.md"))

	// Verify files have content.
	for name, content := range map[string][]byte{"README.md": readme, "contractors.md": contractors, "decisions.md": decisions, "budget.md": budget} {
		if len(content) == 0 {
			t.Fatalf("testdata file %s is empty", name)
		}
	}

	// Step 3: Create project entity.
	addTriple(t, tripleHandler, graph, base, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri")
	addTriple(t, tripleHandler, graph, base, "http://www.w3.org/2000/01/rdf-schema#label", "Patterson Family Home Construction", "literal")
	addTriple(t, tripleHandler, graph, base, "https://hippocamp.dev/ontology#summary",
		"Two-story family home construction at 4218 Barton Creek Drive, Austin TX, $450K budget, 8 months", "literal")

	// Step 4: Extract topics.
	topics := map[string]string{
		"permits":    "Building permits and inspections",
		"materials":  "Construction materials and suppliers",
		"budget":     "Project budget and cost tracking",
		"schedule":   "Construction timeline and milestones",
		"electrical": "Electrical systems and wiring",
		"plumbing":   "Plumbing systems and fixtures",
		"roofing":    "Roof construction and materials",
		"foundation": "Foundation and structural work",
		"hvac":       "Heating, ventilation, and air conditioning",
		"insulation": "Insulation materials and installation",
	}
	for slug, summary := range topics {
		uri := base + "/topic/" + slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Topic", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", slug, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#summary", summary, "literal")
	}

	// Step 5: Extract entities from contractors.md.
	entities := []struct {
		slug, label, summary, topic string
	}{
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
		uri := base + "/entity/" + e.slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", e.label, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#summary", e.summary, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#hasTopic", base+"/topic/"+e.topic, "uri")
	}

	// Step 6: Notes from budget.md and README.md.
	notes := []struct {
		slug, label, content, topic string
	}{
		{"foundation-specs", "Foundation Specifications",
			"Post-tension concrete slab, 4-inch minimum thickness, rebar grid on 12-inch centers, must pass city inspection before framing",
			"foundation"},
		{"lumber-order", "Lumber Order Details",
			"Douglas Fir framing lumber from Hill Country Lumber, delivery scheduled for week 3, bulk discount saves $3,200",
			"materials"},
		{"budget-summary", "Budget Summary",
			"Total budget $450K. Foundation $45K, framing $65K, roofing $30K, electrical $25K, plumbing $20K, HVAC $18K, insulation $12K",
			"budget"},
		{"energy-efficiency", "Energy Efficiency Goals",
			"Targeting HERS rating below 55. Passive solar gain from south-facing windows, spray foam insulation, metal roof heat reflection",
			"insulation"},
	}
	for _, n := range notes {
		uri := base + "/note/" + n.slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Note", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", n.label, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#content", n.content, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#hasTopic", base+"/topic/"+n.topic, "uri")
	}

	// Step 7: Decisions from decisions.md.
	decisionsList := []struct {
		slug, label, rationale, topic string
	}{
		{"metal-roof", "Standing seam metal roof over asphalt shingles",
			"Class 4 impact rating for hail, 50+ year lifespan, 140 mph wind resistance, 15% cooling cost reduction, $800/yr insurance discount",
			"roofing"},
		{"spray-foam", "Spray foam insulation over fiberglass batts",
			"Combined insulation and air sealing, eliminates vapor barrier, annual HVAC savings $600-$900, passes blower door test at 5 ACH50",
			"insulation"},
		{"tankless-heater", "Tankless water heater over tank",
			"Continuous hot water, wall-mounted saves space, Energy Factor 0.96, $150 annual gas savings, 20-year lifespan",
			"plumbing"},
	}
	for _, d := range decisionsList {
		uri := base + "/decision/" + d.slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Decision", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", d.label, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#rationale", d.rationale, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#hasTopic", base+"/topic/"+d.topic, "uri")
	}

	// Step 8: Questions from README.md.
	questions := []struct {
		slug, label string
	}{
		{"triple-pane-windows", "Should we upgrade to triple-pane windows on the west-facing wall? Adds $4,800"},
		{"hoa-paint-colors", "HOA pre-approval for exterior paint colors, deadline May 30"},
	}
	for _, q := range questions {
		uri := base + "/question/" + q.slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Question", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", q.label, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#status", "open", "literal")
	}

	// Step 9: Sources.
	sources := []struct {
		slug, label, summary string
	}{
		{"austin-building-codes", "Austin Residential Building Codes", "City of Austin building code requirements for residential construction"},
		{"nec-2023", "NEC 2023", "National Electrical Code 2023 edition, required for all electrical work"},
		{"irc-2021", "International Residential Code 2021", "IRC adopted by City of Austin with local amendments"},
	}
	for _, s := range sources {
		uri := base + "/source/" + s.slug
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Source", "uri")
		addTriple(t, tripleHandler, graph, uri, "http://www.w3.org/2000/01/rdf-schema#label", s.label, "literal")
		addTriple(t, tripleHandler, graph, uri, "https://hippocamp.dev/ontology#summary", s.summary, "literal")
	}

	// ──────────────────────────────────────────────
	// VERIFY: Graph stats.
	// ──────────────────────────────────────────────
	statsRes := callToolGetText(t, graphHandler, map[string]any{"action": "stats", "name": graph})
	t.Logf("Graph stats: %s", statsRes)

	// ──────────────────────────────────────────────
	// VERIFY: SPARQL queries work against the populated graph.
	// ──────────────────────────────────────────────
	// SPARQL query against named graph — should work after the fix.
	sparqlRes := callToolGetText(t, sparqlHandler, map[string]any{
		"query": "SELECT ?s ?label WHERE { ?s <http://www.w3.org/2000/01/rdf-schema#label> ?label } LIMIT 5",
		"graph": graph,
	})
	t.Logf("Sample labels via SPARQL: %s", sparqlRes)
	if sparqlRes == "[]" || sparqlRes == "" {
		t.Error("SPARQL: expected non-empty results for named graph label query")
	}

	// ──────────────────────────────────────────────
	// VERIFY: Search tool finds knowledge.
	// ──────────────────────────────────────────────
	searchTests := []struct {
		name, query string
		mustFind    string // substring of URI that must appear
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

	passCount := 0
	for _, st := range searchTests {
		t.Run("search/"+st.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": st.query, "scope": graph}
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
				t.Errorf("expected result containing %q in URI, got %d results: %s", st.mustFind, len(results), text)
			} else {
				passCount++
			}
			t.Logf("query=%q results=%d found=%v", st.query, len(results), found)
		})
	}

	// ──────────────────────────────────────────────
	// VERIFY: Cross-domain queries (things an LLM would ask).
	// ──────────────────────────────────────────────
	crossDomainTests := []struct {
		name, query string
		minResults  int
	}{
		{"how much does roofing cost", "roofing cost $30,000", 1},
		{"who handles electrical", "electrical contractor", 1},
		{"what decisions were made about insulation", "insulation decision", 1},
		{"energy savings", "energy savings annual", 1},
		{"all contractors", "contractor builder plumber electrician", 1},
	}

	for _, ct := range crossDomainTests {
		t.Run("cross/"+ct.name, func(t *testing.T) {
			req := mcp.CallToolRequest{}
			req.Params.Arguments = map[string]any{"query": ct.query, "scope": graph}
			res, _ := searchHandler(context.Background(), req)
			text := ResultText(res)
			var results []SearchResult
			json.Unmarshal([]byte(text), &results)
			if len(results) < ct.minResults {
				t.Errorf("expected >= %d results, got %d", ct.minResults, len(results))
			}
			t.Logf("query=%q results=%d", ct.query, len(results))
		})
	}

	t.Logf("=== ANALYSIS SUMMARY ===")
	t.Logf("Testdata files read: README.md, contractors.md, decisions.md, budget.md")
	t.Logf("Topics created: %d", len(topics))
	t.Logf("Entities created: %d", len(entities))
	t.Logf("Notes created: %d", len(notes))
	t.Logf("Decisions created: %d", len(decisionsList))
	t.Logf("Questions created: %d", len(questions))
	t.Logf("Sources created: %d", len(sources))
	t.Logf("Search tests passed: %d/%d", passCount, len(searchTests))

	// Verify persistence: dump to TriG and reload.
	t.Run("persistence", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "house.trig")
		callToolExpectOK(t, graphHandler, map[string]any{"action": "dump", "file": tmpFile})
		t.Logf("Dumped graph to %s", tmpFile)

		info, _ := os.Stat(tmpFile)
		t.Logf("TriG file size: %d bytes", info.Size())
		if info.Size() < 1000 {
			t.Error("TriG file is suspiciously small")
		}

		// Reload into fresh store and verify search still works.
		store2 := rdfstore.NewStore()
		defer store2.Close()
		callToolExpectOK(t, HandlerFor(store2, "graph"), map[string]any{"action": "load", "file": tmpFile})

		handler2 := HandlerFor(store2, "search")
		req := mcp.CallToolRequest{}
		req.Params.Arguments = map[string]any{"query": "Lone Star Builders", "scope": graph}
		res, _ := handler2(context.Background(), req)
		text := ResultText(res)
		if !strings.Contains(text, "lone-star-builders") {
			t.Error("search after reload failed to find Lone Star Builders")
		}
		t.Log("Persistence round-trip: OK")
	})
}

// ── helpers ──

func addTriple(t *testing.T, handler handlerFunc, graph, subj, pred, obj, objType string) {
	t.Helper()
	args := map[string]any{
		"action":    "add",
		"graph":     graph,
		"subject":   subj,
		"predicate": pred,
		"object":    obj,
	}
	if objType != "" {
		args["object_type"] = objType
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("addTriple error: %v", err)
	}
	if res.IsError {
		t.Fatalf("addTriple tool error: %s", ResultText(res))
	}
}

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

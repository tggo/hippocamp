package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// testProject defines a complete test scenario for one project domain.
type testProject struct {
	Name    string
	Graph   string        // named graph URI
	Triples []testTriple  // seed data
	Queries []searchQuery // test queries with expected results
}

// testTriple is a single RDF triple to seed.
type testTriple struct {
	Subject   string
	Predicate string
	Object    string
	ObjType   string // "uri" or "literal"
}

// searchQuery defines one search test case with expectations.
type searchQuery struct {
	Name        string         // descriptive test name
	Args        map[string]any // search tool arguments
	MinResults  int            // at least this many results
	MaxResults  int            // at most this many (0 = no upper bound)
	MustFind    []string       // URIs that MUST appear in results
	MustNotFind []string       // URIs that must NOT appear
}

// Ontology constants used in test data.
const (
	testRdfType      = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"
	testRdfsLabel    = "http://www.w3.org/2000/01/rdf-schema#label"
	testHippoNS      = "https://hippocamp.dev/ontology#"
	testHippoTopic   = testHippoNS + "Topic"
	testHippoEntity  = testHippoNS + "Entity"
	testHippoNote    = testHippoNS + "Note"
	testHippoSource  = testHippoNS + "Source"
	testHippoDecision = testHippoNS + "Decision"
	testHippoQuestion = testHippoNS + "Question"
	testHippoTag     = testHippoNS + "Tag"
	testHippoSummary = testHippoNS + "summary"
	testHippoContent = testHippoNS + "content"
	testHippoRationale = testHippoNS + "rationale"
	testHippoHasTopic  = testHippoNS + "hasTopic"
	testHippoHasTag    = testHippoNS + "hasTag"
	testHippoReferences = testHippoNS + "references"
	testHippoPartOf    = testHippoNS + "partOf"
	testHippoStatus    = testHippoNS + "status"
	testHippoURL       = testHippoNS + "url"
)

// Triple builder helpers to reduce boilerplate.

func tripleType(subj, rdfType string) testTriple {
	return testTriple{subj, testRdfType, rdfType, "uri"}
}

func tripleLabel(subj, label string) testTriple {
	return testTriple{subj, testRdfsLabel, label, "literal"}
}

func tripleSummary(subj, summary string) testTriple {
	return testTriple{subj, testHippoSummary, summary, "literal"}
}

func tripleContent(subj, content string) testTriple {
	return testTriple{subj, testHippoContent, content, "literal"}
}

func tripleHasTopic(subj, topicURI string) testTriple {
	return testTriple{subj, testHippoHasTopic, topicURI, "uri"}
}

func tripleRationale(subj, rationale string) testTriple {
	return testTriple{subj, testHippoRationale, rationale, "literal"}
}

func tripleStatus(subj, status string) testTriple {
	return testTriple{subj, testHippoStatus, status, "literal"}
}

func tripleRef(subj, obj string) testTriple {
	return testTriple{subj, testHippoReferences, obj, "uri"}
}

func triplePartOf(subj, parent string) testTriple {
	return testTriple{subj, testHippoPartOf, parent, "uri"}
}

func tripleURL(subj, url string) testTriple {
	return testTriple{subj, testHippoURL, url, "literal"}
}

func tripleTag(subj, tagURI string) testTriple {
	return testTriple{subj, testHippoHasTag, tagURI, "uri"}
}

func tripleCreatedAt(subj, iso8601 string) testTriple {
	return testTriple{subj, testHippoNS + "createdAt", iso8601, "literal"}
}

func tripleUpdatedAt(subj, iso8601 string) testTriple {
	return testTriple{subj, testHippoNS + "updatedAt", iso8601, "literal"}
}

// entity returns a set of triples for an Entity resource.
func entity(uri, label, summary string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoEntity),
		tripleLabel(uri, label),
		tripleSummary(uri, summary),
	}
}

// topic returns a set of triples for a Topic resource.
func topic(uri, label, summary string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoTopic),
		tripleLabel(uri, label),
		tripleSummary(uri, summary),
	}
}

// note returns a set of triples for a Note resource.
func note(uri, label, content string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoNote),
		tripleLabel(uri, label),
		tripleContent(uri, content),
	}
}

// decision returns a set of triples for a Decision resource.
func decision(uri, label, rationale string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoDecision),
		tripleLabel(uri, label),
		tripleRationale(uri, rationale),
	}
}

// question returns a set of triples for a Question resource.
func question(uri, label string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoQuestion),
		tripleLabel(uri, label),
		tripleStatus(uri, "open"),
	}
}

// source returns a set of triples for a Source resource.
func source(uri, label, summary string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoSource),
		tripleLabel(uri, label),
		tripleSummary(uri, summary),
	}
}

// tag returns a set of triples for a Tag resource.
func tag(uri, label string) []testTriple {
	return []testTriple{
		tripleType(uri, testHippoTag),
		tripleLabel(uri, label),
	}
}

// countFound returns how many of the expected URIs were found.
func countFound(found map[string]bool, expected []string) int {
	n := 0
	for _, uri := range expected {
		if found[uri] {
			n++
		}
	}
	return n
}

func TestSearchProjects(t *testing.T) {
	projects := []testProject{
		houseProject(),
		gardenProject(),
		salesProject(),
		accountingProject(),
		recipesProject(),
	}

	totalQueries := 0
	for _, proj := range projects {
		totalQueries += len(proj.Queries)
	}
	t.Logf("Running %d queries across %d projects (%d total test flows)", totalQueries, len(projects), totalQueries)

	for _, proj := range projects {
		t.Run(proj.Name, func(t *testing.T) {
			store := rdfstore.NewStore()
			defer store.Close()

			if proj.Graph != "" {
				store.CreateGraph(proj.Graph)
			}
			for _, tr := range proj.Triples {
				graphName := proj.Graph
				if err := store.AddTriple(graphName, tr.Subject, tr.Predicate, tr.Object, tr.ObjType, "", ""); err != nil {
					t.Fatalf("seed triple (%s, %s, %s): %v", tr.Subject, tr.Predicate, tr.Object, err)
				}
			}

			handler := HandlerFor(store, "search")
			t.Logf("Seeded %d triples, running %d queries", len(proj.Triples), len(proj.Queries))

			for _, q := range proj.Queries {
				t.Run(q.Name, func(t *testing.T) {
					// Add scope to args if graph is set and not already specified.
					args := make(map[string]any, len(q.Args))
					for k, v := range q.Args {
						args[k] = v
					}
					if _, hasScope := args["scope"]; !hasScope && proj.Graph != "" {
						args["scope"] = proj.Graph
					}

					results := callSearchProject(t, handler, args)

					// Check result count bounds.
					if q.MinResults > 0 && len(results) < q.MinResults {
						t.Errorf("expected >= %d results, got %d", q.MinResults, len(results))
					}
					if q.MaxResults > 0 && len(results) > q.MaxResults {
						t.Errorf("expected <= %d results, got %d", q.MaxResults, len(results))
					}

					// Collect found URIs.
					found := map[string]bool{}
					for _, r := range results {
						found[r.URI] = true
					}

					// Check MustFind (recall).
					for _, uri := range q.MustFind {
						if !found[uri] {
							t.Errorf("RECALL MISS: expected to find %s in results", uri)
						}
					}

					// Check MustNotFind (precision).
					for _, uri := range q.MustNotFind {
						if found[uri] {
							t.Errorf("PRECISION MISS: expected NOT to find %s in results", uri)
						}
					}

					// Log metrics.
					if len(q.MustFind) > 0 {
						hits := countFound(found, q.MustFind)
						recall := float64(hits) / float64(len(q.MustFind))
						t.Logf("recall=%.2f (%d/%d)", recall, hits, len(q.MustFind))
					}
					if len(q.MustNotFind) > 0 && len(results) > 0 {
						falsePositives := countFound(found, q.MustNotFind)
						precision := 1.0 - float64(falsePositives)/float64(len(results))
						t.Logf("precision=%.2f (false_positives=%d/%d)", precision, falsePositives, len(results))
					}
				})
			}
		})
	}
}

// callSearchProject invokes the search handler and returns parsed results.
func callSearchProject(t *testing.T, handler handlerFunc, args map[string]any) []SearchResult {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args

	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	text := ResultText(res)
	if res.IsError {
		t.Fatalf("search returned error: %s", text)
	}

	// Try array first (normal results).
	var results []SearchResult
	if err := json.Unmarshal([]byte(text), &results); err == nil {
		return results
	}

	// Try hint object (zero-result response).
	var hint struct {
		Results []SearchResult `json:"results"`
		Hint    string         `json:"hint"`
	}
	if err := json.Unmarshal([]byte(text), &hint); err != nil {
		t.Fatalf("unmarshal: %v (text: %s)", err, text)
	}
	return hint.Results
}

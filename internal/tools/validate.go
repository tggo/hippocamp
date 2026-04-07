package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// validHippoTypeNames is the list of local names for fuzzy matching.
var validHippoTypeNames []string

func init() {
	for t := range validHippoTypes {
		validHippoTypeNames = append(validHippoTypeNames, strings.TrimPrefix(t, hippoNS))
	}
}

// validHippoTypes are the accepted rdf:type values from the hippo: ontology.
var validHippoTypes = map[string]bool{
	hippoNS + "Topic":      true,
	hippoNS + "Entity":     true,
	hippoNS + "Note":       true,
	hippoNS + "Source":     true,
	hippoNS + "Decision":  true,
	hippoNS + "Question":  true,
	hippoNS + "Tag":       true,
	hippoNS + "Project":   true,
	hippoNS + "Module":    true,
	hippoNS + "File":      true,
	hippoNS + "Symbol":    true,
	hippoNS + "Function":  true,
	hippoNS + "Struct":    true,
	hippoNS + "Interface": true,
	hippoNS + "Class":     true,
	hippoNS + "Dependency": true,
	hippoNS + "Concept":   true,
}

type validateOutput struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Fixes    []string `json:"fixes,omitempty"`
	Stats    struct {
		Resources   int `json:"resources"`
		WithType    int `json:"with_type"`
		WithLabel   int `json:"with_label"`
		NonStandard int `json:"non_standard_types"`
	} `json:"stats"`
}

func validateTool() mcp.Tool {
	return mcp.NewTool("validate",
		mcp.WithDescription(`Validate the knowledge graph for ontology compliance and health. Checks that all resources use standard hippo: types, have labels, decisions have rationale, and detects dangling references, orphan resources, and failed search queries that suggest missing aliases.

Run after: bulk triple additions, removing resources, or when search returns unexpected zero results.

Results include actionable fixes — e.g. triple remove commands for dangling references, and alias suggestions from zero-result search analytics.

Examples:
  {"scope": "project:house-construction"}
  {}

Returns JSON with valid (bool), warnings (array), fixes (array of suggested commands), and stats.`),
		mcp.WithString("scope",
			mcp.Description("Named graph to validate (omit to validate all graphs)"),
		),
	)
}

func validateHandlerFactory(store *rdfstore.Store) handlerFunc {
	return func(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		scope := req.GetString("scope", "")

		graphNames := []string{}
		if scope != "" {
			graphNames = append(graphNames, scope)
		} else {
			graphNames = store.ListGraphs()
		}

		var out validateOutput
		out.Errors = []string{}
		out.Warnings = []string{}

		// Collect all subjects and their properties.
		type resourceInfo struct {
			uri       string
			rdfType   string
			hasLabel  bool
			hasRationale bool
		}
		resources := map[string]*resourceInfo{}

		for _, gn := range graphNames {
			triples, err := store.ListTriples(gn, "", "", "")
			if err != nil {
				continue
			}
			for _, t := range triples {
				ri, ok := resources[t.Subject]
				if !ok {
					ri = &resourceInfo{uri: t.Subject}
					resources[t.Subject] = ri
				}
				switch t.Predicate {
				case rdfType:
					ri.rdfType = t.Object
				case rdfsLabel:
					ri.hasLabel = true
				case hippoNS + "rationale":
					ri.hasRationale = true
				}
			}
		}

		out.Stats.Resources = len(resources)

		for uri, ri := range resources {
			if ri.rdfType != "" {
				out.Stats.WithType++

				if !validHippoTypes[ri.rdfType] {
					out.Stats.NonStandard++
					shortType := ri.rdfType
					if strings.HasPrefix(shortType, hippoNS) {
						localName := strings.TrimPrefix(shortType, hippoNS)
						shortType = "hippo:" + localName
						// Unknown hippo: type — suggest closest match via fuzzy matching.
						if match, score := suggestType(localName); match != "" {
							out.Warnings = append(out.Warnings,
								fmt.Sprintf("unknown hippo type %s on <%s> — did you mean hippo:%s? (%.0f%% similar)", shortType, uri, match, score*100))
							out.Fixes = append(out.Fixes,
								fmt.Sprintf("triple action=remove subject=%s predicate=%s object=%s", uri, rdfType, ri.rdfType))
							out.Fixes = append(out.Fixes,
								fmt.Sprintf("triple action=add subject=%s predicate=%s object=%s%s", uri, rdfType, hippoNS, match))
						} else {
							out.Warnings = append(out.Warnings,
								fmt.Sprintf("unknown hippo type %s on <%s> — consider using hippo:Entity with hippo:hasTag instead", shortType, uri))
						}
					} else {
						// Non-hippo namespace — error.
						out.Errors = append(out.Errors,
							fmt.Sprintf("non-standard type %q on <%s> — use hippo:Entity, hippo:Topic, etc.", shortType, uri))
					}
				}
			}

			if ri.hasLabel {
				out.Stats.WithLabel++
			} else if ri.rdfType != "" {
				out.Errors = append(out.Errors,
					fmt.Sprintf("missing rdfs:label on <%s> (type: %s)", uri, ri.rdfType))
			}

			// Decisions should have rationale.
			if ri.rdfType == hippoNS+"Decision" && !ri.hasRationale {
				out.Errors = append(out.Errors,
					fmt.Sprintf("Decision <%s> is missing hippo:rationale", uri))
			}
		}

		out.Valid = len(out.Errors) == 0

		// Check schema version — warn if migration is available.
		if pending := PendingMigrations(store); len(pending) > 0 {
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("schema update available — run graph action=migrate to apply %d pending migration(s): %s",
					len(pending), strings.Join(pending, "; ")))
			out.Fixes = append(out.Fixes, `graph action=migrate`)
		}

		// Append background health check results if available.
		if globalChecker != nil {
			if report := globalChecker.Report(); report != nil {
				for _, dr := range report.DanglingRefs {
					shortPred := dr.Predicate
					if strings.HasPrefix(shortPred, hippoNS) {
						shortPred = "hippo:" + strings.TrimPrefix(shortPred, hippoNS)
					}
					out.Warnings = append(out.Warnings,
						fmt.Sprintf("dangling reference: <%s> %s <%s> — target does not exist", dr.Subject, shortPred, dr.Object))
					out.Fixes = append(out.Fixes,
						fmt.Sprintf(`triple action=remove subject=%s predicate=%s object=%s`, dr.Subject, dr.Predicate, dr.Object))
				}
				for _, orphan := range report.OrphanResources {
					out.Warnings = append(out.Warnings,
						fmt.Sprintf("orphan resource: <%s> has no incoming or outgoing relationships", orphan))
				}
				for _, alias := range report.MissingAliases {
					out.Warnings = append(out.Warnings,
						fmt.Sprintf("zero-result search: %q — consider adding hippo:alias to matching resources", alias.Query))
				}
			}
		}

		data, _ := json.Marshal(out)
		return mcp.NewToolResultText(string(data)), nil
	}
}

// suggestType finds the closest valid hippo type name using string similarity.
// Returns the best match and similarity score (0.0–1.0). Returns ("", 0) if no
// match above the 0.5 threshold.
func suggestType(unknown string) (string, float64) {
	unknown = strings.ToLower(unknown)
	bestMatch := ""
	bestScore := 0.0

	for _, name := range validHippoTypeNames {
		score := stringSimilarity(unknown, strings.ToLower(name))
		if score > bestScore {
			bestScore = score
			bestMatch = name
		}
	}

	if bestScore < 0.5 {
		return "", 0
	}
	return bestMatch, bestScore
}

// stringSimilarity computes a similarity ratio between two strings using the
// longest common subsequence: 2 * LCS / (len(a) + len(b)).
func stringSimilarity(a, b string) float64 {
	if a == b {
		return 1.0
	}
	la, lb := len(a), len(b)
	if la == 0 || lb == 0 {
		return 0.0
	}

	// LCS via dynamic programming.
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	lcs := dp[la][lb]
	return 2.0 * float64(lcs) / float64(la+lb)
}

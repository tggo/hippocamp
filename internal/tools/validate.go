package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

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
	Stats    struct {
		Resources   int `json:"resources"`
		WithType    int `json:"with_type"`
		WithLabel   int `json:"with_label"`
		NonStandard int `json:"non_standard_types"`
	} `json:"stats"`
}

func validateTool() mcp.Tool {
	return mcp.NewTool("validate",
		mcp.WithDescription(`Validate the knowledge graph for ontology compliance. Checks that all resources use standard hippo: types, have labels, and decisions have rationale.

Examples:
  {"scope": "project:house-construction"}
  {}

Returns JSON with valid (bool), warnings (array), and stats.`),
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
						shortType = "hippo:" + strings.TrimPrefix(shortType, hippoNS)
						// Unknown hippo: type — warning only (user may extend the ontology).
						out.Warnings = append(out.Warnings,
							fmt.Sprintf("unknown hippo type %s on <%s> — consider using hippo:Entity with hippo:hasTag instead", shortType, uri))
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

		data, _ := json.Marshal(out)
		return mcp.NewToolResultText(string(data)), nil
	}
}

package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// CurrentSchemaVersion is bumped whenever the ontology or tool behavior
// changes in a way that existing graphs benefit from a data migration.
// Each version has a corresponding migration function in schemaMigrations.
const CurrentSchemaVersion = 2

const (
	metaURI         = "urn:hippocamp:meta"
	schemaVersionP  = hippoNS + "schemaVersion"
)

// schemaMigrations maps version numbers to the migration that upgrades
// FROM (version-1) TO that version. Version 1 is the baseline (no migration needed).
var schemaMigrations = map[int]migration{
	2: {
		Description: "Add provenance defaults and temporal validity support",
		Steps: []string{
			"Set hippo:provenance='extracted' on typed resources without provenance",
			"Set hippo:confidence=1.0 on typed resources without confidence",
		},
		Apply: migrateToV2,
	},
}

type migration struct {
	Description string
	Steps       []string
	Apply       func(store *rdfstore.Store) (int, error) // returns number of triples added
}

// GetSchemaVersion reads the current schema version from the graph.
// Returns 0 if no version is set (pre-versioning graph).
func GetSchemaVersion(store *rdfstore.Store) int {
	triples, _ := store.ListTriples("", metaURI, schemaVersionP, "")
	if len(triples) == 0 {
		return 0
	}
	v, _ := strconv.Atoi(triples[0].Object)
	return v
}

// setSchemaVersion writes the schema version to the graph.
func setSchemaVersion(store *rdfstore.Store, version int) {
	// Remove old version triple(s).
	triples, _ := store.ListTriples("", metaURI, schemaVersionP, "")
	for _, t := range triples {
		store.RemoveTriple("", t.Subject, t.Predicate, t.Object)
	}
	store.AddTriple("", metaURI, schemaVersionP, strconv.Itoa(version), "literal", "", "")
}

// PendingMigrations returns the list of migrations that need to be applied.
func PendingMigrations(store *rdfstore.Store) []string {
	current := GetSchemaVersion(store)
	var pending []string
	for v := current + 1; v <= CurrentSchemaVersion; v++ {
		if m, ok := schemaMigrations[v]; ok {
			pending = append(pending, fmt.Sprintf("v%d: %s", v, m.Description))
		}
	}
	return pending
}

// handleGraphMigrate applies all pending schema migrations.
func handleGraphMigrate(store *rdfstore.Store) (*mcp.CallToolResult, error) {
	current := GetSchemaVersion(store)

	if current >= CurrentSchemaVersion {
		return mcp.NewToolResultText(fmt.Sprintf("schema is up to date (v%d)", current)), nil
	}

	type migrationResult struct {
		Version     int      `json:"version"`
		Description string   `json:"description"`
		Steps       []string `json:"steps"`
		Added       int      `json:"triples_added"`
	}

	var results []migrationResult

	for v := current + 1; v <= CurrentSchemaVersion; v++ {
		m, ok := schemaMigrations[v]
		if !ok {
			continue
		}

		added, err := m.Apply(store)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("migration v%d failed: %v", v, err)), nil
		}

		setSchemaVersion(store, v)

		results = append(results, migrationResult{
			Version:     v,
			Description: m.Description,
			Steps:       m.Steps,
			Added:       added,
		})
	}

	data, _ := json.MarshalIndent(map[string]any{
		"migrated_from": current,
		"migrated_to":   CurrentSchemaVersion,
		"migrations":    results,
	}, "", "  ")
	return mcp.NewToolResultText(string(data)), nil
}

// ── Migration functions ─────────────────────────────────────

// migrateToV2 adds provenance defaults to all typed resources that lack them.
func migrateToV2(store *rdfstore.Store) (int, error) {
	added := 0

	graphs := store.ListGraphs()
	for _, g := range graphs {
		// Skip analytics graph.
		if strings.Contains(g, "analytics") {
			continue
		}

		triples, err := store.ListTriples(g, "", "", "")
		if err != nil {
			continue
		}

		// Collect typed resources and what they already have.
		type info struct {
			hasProvenance  bool
			hasConfidence  bool
		}
		resources := map[string]*info{}

		for _, t := range triples {
			if t.Predicate == rdfType && strings.HasPrefix(t.Object, hippoNS) {
				if resources[t.Subject] == nil {
					resources[t.Subject] = &info{}
				}
			}
			if t.Predicate == hippoNS+"provenance" {
				if resources[t.Subject] == nil {
					resources[t.Subject] = &info{}
				}
				resources[t.Subject].hasProvenance = true
			}
			if t.Predicate == hippoNS+"confidence" {
				if resources[t.Subject] == nil {
					resources[t.Subject] = &info{}
				}
				resources[t.Subject].hasConfidence = true
			}
		}

		for uri, ri := range resources {
			if !ri.hasProvenance {
				store.AddTriple(g, uri, hippoNS+"provenance", "extracted", "literal", "", "")
				added++
			}
			if !ri.hasConfidence {
				store.AddTriple(g, uri, hippoNS+"confidence", "1.0", "literal", "", "http://www.w3.org/2001/XMLSchema#float")
				added++
			}
		}
	}

	return added, nil
}

package tools

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

func TestMigrate_FreshGraph(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Fresh graph has no schema version.
	v := GetSchemaVersion(store)
	if v != 0 {
		t.Errorf("expected version 0 on fresh graph, got %d", v)
	}

	// Should have pending migrations.
	pending := PendingMigrations(store)
	if len(pending) == 0 {
		t.Fatal("expected pending migrations on fresh graph")
	}
	t.Logf("Pending: %v", pending)
}

func TestMigrate_ApplyToV2(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Seed some resources WITHOUT provenance (simulating old data).
	store.AddTriple("", "https://ex.org/alice", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "https://ex.org/alice", rdfsLabel, "Alice", "literal", "", "")
	store.AddTriple("", "https://ex.org/bob", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "https://ex.org/bob", rdfsLabel, "Bob", "literal", "", "")
	// Bob already has provenance.
	store.AddTriple("", "https://ex.org/bob", hippoNS+"provenance", "inferred", "literal", "", "")
	store.AddTriple("", "https://ex.org/bob", hippoNS+"confidence", "0.8", "literal", "", "")

	// Run migrate.
	handler := HandlerFor(store, "graph")
	text := callToolGetText(t, handler, map[string]any{"action": "migrate"})
	t.Logf("Migrate result: %s", text)

	// Verify version bumped.
	v := GetSchemaVersion(store)
	if v != CurrentSchemaVersion {
		t.Errorf("expected version %d, got %d", CurrentSchemaVersion, v)
	}

	// Alice should now have provenance="extracted" and confidence=1.0.
	aliceProvenance, _ := store.ListTriples("", "https://ex.org/alice", hippoNS+"provenance", "")
	if len(aliceProvenance) != 1 || aliceProvenance[0].Object != "extracted" {
		t.Errorf("expected alice provenance=extracted, got %v", aliceProvenance)
	}
	aliceConfidence, _ := store.ListTriples("", "https://ex.org/alice", hippoNS+"confidence", "")
	if len(aliceConfidence) != 1 || aliceConfidence[0].Object != "1.0" {
		t.Errorf("expected alice confidence=1.0, got %v", aliceConfidence)
	}

	// Bob should keep his existing provenance (not overwritten).
	bobProvenance, _ := store.ListTriples("", "https://ex.org/bob", hippoNS+"provenance", "")
	if len(bobProvenance) != 1 || bobProvenance[0].Object != "inferred" {
		t.Errorf("expected bob provenance=inferred (preserved), got %v", bobProvenance)
	}
	bobConfidence, _ := store.ListTriples("", "https://ex.org/bob", hippoNS+"confidence", "")
	if len(bobConfidence) != 1 || bobConfidence[0].Object != "0.8" {
		t.Errorf("expected bob confidence=0.8 (preserved), got %v", bobConfidence)
	}

	// Parse result JSON.
	var res struct {
		MigratedFrom int `json:"migrated_from"`
		MigratedTo   int `json:"migrated_to"`
		Migrations   []struct {
			Version int `json:"version"`
			Added   int `json:"triples_added"`
		} `json:"migrations"`
	}
	json.Unmarshal([]byte(text), &res)

	if res.MigratedFrom != 0 {
		t.Errorf("expected migrated_from=0, got %d", res.MigratedFrom)
	}
	if res.MigratedTo != CurrentSchemaVersion {
		t.Errorf("expected migrated_to=%d, got %d", CurrentSchemaVersion, res.MigratedTo)
	}
	// Alice needed 2 triples (provenance + confidence), Bob needed 0.
	if len(res.Migrations) > 0 && res.Migrations[0].Added != 2 {
		t.Errorf("expected 2 triples added for alice, got %d", res.Migrations[0].Added)
	}
}

func TestMigrate_AlreadyUpToDate(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Run migrate twice.
	callToolGetText(t, HandlerFor(store, "graph"), map[string]any{"action": "migrate"})
	text := callToolGetText(t, HandlerFor(store, "graph"), map[string]any{"action": "migrate"})

	if !strings.Contains(text, "up to date") {
		t.Errorf("expected 'up to date' message, got: %s", text)
	}
}

func TestMigrate_ValidateWarning(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Add a resource (old schema, no version).
	store.AddTriple("", "https://ex.org/x", rdfType, hippoNS+"Entity", "uri", "", "")
	store.AddTriple("", "https://ex.org/x", rdfsLabel, "Test", "literal", "", "")

	// Validate should warn about pending migration.
	vr := callValidate(t, store, map[string]any{})

	foundWarning := false
	foundFix := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "schema update available") {
			foundWarning = true
		}
	}
	for _, f := range vr.Fixes {
		if strings.Contains(f, "graph action=migrate") {
			foundFix = true
		}
	}

	if !foundWarning {
		t.Error("expected schema migration warning in validate")
	}
	if !foundFix {
		t.Error("expected 'graph action=migrate' in fixes")
	}

	// Run migrate, then validate again — warning should be gone.
	callToolGetText(t, HandlerFor(store, "graph"), map[string]any{"action": "migrate"})
	vr2 := callValidate(t, store, map[string]any{})

	for _, w := range vr2.Warnings {
		if strings.Contains(w, "schema update available") {
			t.Error("migration warning should be gone after migrate")
		}
	}
}

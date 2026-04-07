package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/analytics"
	"github.com/ruslanmv/hippocamp/internal/healthcheck"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

// validateHealthResult extends validateResult with the fixes field.
type validateHealthResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Fixes    []string `json:"fixes"`
	Stats    struct {
		Resources   int `json:"resources"`
		WithType    int `json:"with_type"`
		WithLabel   int `json:"with_label"`
		NonStandard int `json:"non_standard_types"`
	} `json:"stats"`
}

func callValidateHealth(t *testing.T, store *rdfstore.Store, args map[string]any) validateHealthResult {
	t.Helper()
	handler := HandlerFor(store, "validate")
	if handler == nil {
		t.Fatal("validate tool not registered")
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	res, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("validate error: %v", err)
	}
	text := ResultText(res)
	var vr validateHealthResult
	if err := json.Unmarshal([]byte(text), &vr); err != nil {
		t.Fatalf("unmarshal validate result: %v (text: %s)", err, text)
	}
	return vr
}

func TestValidateWithDanglingRef(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Entity A references non-existent entity B.
	store.AddTriple("", "http://ex.org/A", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://ex.org/A", rdfsLabel, "Entity A", "literal", "", "")
	store.AddTriple("", "http://ex.org/A", testHippoReferences, "http://ex.org/B", "uri", "", "")

	checker := healthcheck.New(store, time.Hour)
	defer SetHealthChecker(nil)
	time.Sleep(100 * time.Millisecond)

	SetHealthChecker(checker)

	vr := callValidateHealth(t, store, map[string]any{})

	// Should be valid from ontology perspective (A has type + label).
	if !vr.Valid {
		t.Errorf("expected valid (ontology-wise), got errors: %v", vr.Errors)
	}

	// Should have a dangling reference warning.
	foundDangling := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "dangling reference") && strings.Contains(w, "http://ex.org/B") {
			foundDangling = true
		}
	}
	if !foundDangling {
		t.Errorf("expected dangling reference warning about B, got warnings: %v", vr.Warnings)
	}

	// Should have a fix command suggesting triple action=remove.
	foundFix := false
	for _, f := range vr.Fixes {
		if strings.Contains(f, "triple action=remove") {
			foundFix = true
		}
	}
	if !foundFix {
		t.Errorf("expected fix command with 'triple action=remove', got fixes: %v", vr.Fixes)
	}
}

func TestValidateWithOrphan(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Typed entity with no relationships — orphan.
	store.AddTriple("", "http://ex.org/orphan", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://ex.org/orphan", rdfsLabel, "Lonely Entity", "literal", "", "")

	checker := healthcheck.New(store, time.Hour)
	defer SetHealthChecker(nil)
	time.Sleep(100 * time.Millisecond)

	SetHealthChecker(checker)

	vr := callValidateHealth(t, store, map[string]any{})

	foundOrphan := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "orphan resource") && strings.Contains(w, "http://ex.org/orphan") {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Errorf("expected orphan warning about http://ex.org/orphan, got warnings: %v", vr.Warnings)
	}
}

func TestValidateWithZeroResultSuggestion(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Seed the analytics graph with a zero-result search call.
	_ = analytics.New(store) // ensures the analytics graph exists
	callSubject := "urn:hippocamp:analytics:call:manual"
	store.AddTriple(analytics.GraphURI, callSubject,
		"http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
		"http://purl.org/hippocamp/analytics#ToolCall", "uri", "", "")
	store.AddTriple(analytics.GraphURI, callSubject,
		"http://purl.org/hippocamp/analytics#tool", "search", "literal", "", "")
	store.AddTriple(analytics.GraphURI, callSubject,
		"http://purl.org/hippocamp/analytics#input", "nonexistent-term", "literal", "", "")
	store.AddTriple(analytics.GraphURI, callSubject,
		"http://purl.org/hippocamp/analytics#resultCount", "0", "literal", "", "http://www.w3.org/2001/XMLSchema#integer")

	checker := healthcheck.New(store, time.Hour)
	defer SetHealthChecker(nil)
	time.Sleep(100 * time.Millisecond)

	SetHealthChecker(checker)

	vr := callValidateHealth(t, store, map[string]any{})

	foundAlias := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "zero-result search") && strings.Contains(w, "nonexistent-term") {
			foundAlias = true
		}
	}
	if !foundAlias {
		t.Errorf("expected alias suggestion warning for 'nonexistent-term', got warnings: %v", vr.Warnings)
	}
}

func TestValidateWithoutHealthChecker(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Ensure no health checker is set.
	SetHealthChecker(nil)
	defer SetHealthChecker(nil)

	store.AddTriple("", "http://ex.org/clean", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://ex.org/clean", rdfsLabel, "Clean Entity", "literal", "", "")

	// Set schema version to current to avoid migration warnings.
	setSchemaVersion(store, CurrentSchemaVersion)

	vr := callValidateHealth(t, store, map[string]any{})

	if !vr.Valid {
		t.Errorf("expected valid result without health checker, got errors: %v", vr.Errors)
	}
	if len(vr.Warnings) != 0 {
		t.Errorf("expected no warnings without health checker, got: %v", vr.Warnings)
	}
}

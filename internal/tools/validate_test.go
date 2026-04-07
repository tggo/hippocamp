package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

type validateResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	Fixes    []string `json:"fixes"`
	Stats    struct {
		Resources    int `json:"resources"`
		WithType     int `json:"with_type"`
		WithLabel    int `json:"with_label"`
		NonStandard  int `json:"non_standard_types"`
	} `json:"stats"`
}

func callValidate(t *testing.T, store *rdfstore.Store, args map[string]any) validateResult {
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
	var vr validateResult
	if err := json.Unmarshal([]byte(text), &vr); err != nil {
		t.Fatalf("unmarshal validate result: %v (text: %s)", err, text)
	}
	return vr
}

func TestValidate_AllTypesFromOntology(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Valid entity.
	store.AddTriple("", "http://test.org/a", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/a", rdfsLabel, "Valid Entity", "literal", "", "")

	// Invalid: custom type not from hippo: namespace.
	store.AddTriple("", "http://test.org/b", rdfType, "http://example.org/TomatoVariety", "uri", "", "")
	store.AddTriple("", "http://test.org/b", rdfsLabel, "San Marzano", "literal", "", "")

	vr := callValidate(t, store, map[string]any{})
	if vr.Valid {
		t.Error("expected invalid due to non-standard type")
	}
	if vr.Stats.NonStandard != 1 {
		t.Errorf("expected 1 non-standard type, got %d", vr.Stats.NonStandard)
	}
	if len(vr.Errors) == 0 {
		t.Error("expected errors about non-standard type from non-hippo namespace")
	}
}

func TestValidate_MissingLabel(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// Resource with type but no label.
	store.AddTriple("", "http://test.org/a", rdfType, testHippoEntity, "uri", "", "")
	// No rdfs:label added.

	vr := callValidate(t, store, map[string]any{})
	if vr.Valid {
		t.Error("expected invalid due to missing label")
	}
	if vr.Stats.WithLabel != 0 {
		t.Errorf("expected 0 with label, got %d", vr.Stats.WithLabel)
	}
}

func TestValidate_Clean(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.AddTriple("", "http://test.org/a", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("", "http://test.org/a", rdfsLabel, "Clean Entity", "literal", "", "")
	store.AddTriple("", "http://test.org/b", rdfType, testHippoTopic, "uri", "", "")
	store.AddTriple("", "http://test.org/b", rdfsLabel, "Clean Topic", "literal", "", "")

	vr := callValidate(t, store, map[string]any{})
	if !vr.Valid {
		t.Errorf("expected valid, got warnings: %v", vr.Warnings)
	}
	if vr.Stats.Resources != 2 {
		t.Errorf("expected 2 resources, got %d", vr.Stats.Resources)
	}
}

func TestValidate_Scoped(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.CreateGraph("http://test.org/g1")
	store.AddTriple("http://test.org/g1", "http://test.org/a", rdfType, testHippoEntity, "uri", "", "")
	store.AddTriple("http://test.org/g1", "http://test.org/a", rdfsLabel, "Valid", "literal", "", "")

	// Invalid in default graph — should NOT affect g1 validation.
	store.AddTriple("", "http://test.org/b", rdfType, "http://custom/Type", "uri", "", "")

	vr := callValidate(t, store, map[string]any{"scope": "http://test.org/g1"})
	if !vr.Valid {
		t.Errorf("scoped validation should be valid, got: %v", vr.Warnings)
	}
}

func TestValidate_HippoUnknownType_WarningNotError(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// hippo:Component is not in the standard list but IS in hippo: namespace.
	store.AddTriple("", "http://test.org/a", rdfType, testHippoNS+"Component", "uri", "", "")
	store.AddTriple("", "http://test.org/a", rdfsLabel, "Foundation", "literal", "", "")

	vr := callValidate(t, store, map[string]any{})
	if !vr.Valid {
		t.Error("hippo: unknown type should still be valid (warning only)")
	}
	if len(vr.Warnings) == 0 {
		t.Error("expected warning about unknown hippo type")
	}
	if len(vr.Errors) != 0 {
		t.Errorf("expected no errors for hippo: type, got: %v", vr.Errors)
	}
}

func TestValidate_DecisionWithoutRationale(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	store.AddTriple("", "http://test.org/d", rdfType, testHippoDecision, "uri", "", "")
	store.AddTriple("", "http://test.org/d", rdfsLabel, "Some Decision", "literal", "", "")
	// No hippo:rationale added.

	vr := callValidate(t, store, map[string]any{})
	if vr.Valid {
		t.Error("expected invalid: Decision without rationale")
	}
}

func TestValidate_FuzzyTypeMatch(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// hippo:Component should suggest hippo:Concept (high similarity).
	store.AddTriple("", "http://test.org/a", rdfType, testHippoNS+"Component", "uri", "", "")
	store.AddTriple("", "http://test.org/a", rdfsLabel, "Auth Module", "literal", "", "")

	vr := callValidate(t, store, map[string]any{})

	if !vr.Valid {
		t.Error("unknown hippo type should still be valid (warning only)")
	}

	// Should have a warning with "did you mean"
	found := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "did you mean") {
			found = true
			t.Logf("warning: %s", w)
		}
	}
	if !found {
		t.Errorf("expected 'did you mean' warning, got: %v", vr.Warnings)
	}

	// Should have fix suggestions
	if len(vr.Fixes) < 2 {
		t.Errorf("expected at least 2 fixes (remove + add), got %d: %v", len(vr.Fixes), vr.Fixes)
	}
}

func TestValidate_FuzzyTypeMatch_NoMatch(t *testing.T) {
	store := rdfstore.NewStore()
	defer store.Close()

	// hippo:Zzzzz has no close match — should fall back to generic message.
	store.AddTriple("", "http://test.org/a", rdfType, testHippoNS+"Zzzzz", "uri", "", "")
	store.AddTriple("", "http://test.org/a", rdfsLabel, "Something", "literal", "", "")

	vr := callValidate(t, store, map[string]any{})

	found := false
	for _, w := range vr.Warnings {
		if strings.Contains(w, "consider using hippo:Entity") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected generic fallback warning, got: %v", vr.Warnings)
	}
}

func TestSuggestType(t *testing.T) {
	tests := []struct {
		input    string
		wantMatch string
		wantMin  float64
	}{
		{"Component", "Concept", 0.5},
		{"Enity", "Entity", 0.5},     // typo
		{"Functon", "Function", 0.5},  // typo
		{"Strukt", "Struct", 0.5},     // close
		{"Zzzzzzz", "", 0},            // no match
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			match, score := suggestType(tt.input)
			if tt.wantMatch == "" {
				if match != "" {
					t.Errorf("expected no match, got %q (%.2f)", match, score)
				}
			} else {
				if match != tt.wantMatch {
					t.Errorf("expected match %q, got %q (%.2f)", tt.wantMatch, match, score)
				}
				if score < tt.wantMin {
					t.Errorf("expected score >= %.2f, got %.2f", tt.wantMin, score)
				}
			}
		})
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		a, b    string
		wantMin float64
	}{
		{"concept", "concept", 1.0},
		{"component", "concept", 0.5},
		{"entity", "enity", 0.8},
		{"", "abc", 0.0},
		{"abc", "", 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			score := stringSimilarity(tt.a, tt.b)
			if score < tt.wantMin {
				t.Errorf("stringSimilarity(%q, %q) = %.2f, want >= %.2f", tt.a, tt.b, score, tt.wantMin)
			}
		})
	}
}

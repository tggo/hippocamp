package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/ruslanmv/hippocamp/internal/rdfstore"
)

type validateResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
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
	if len(vr.Warnings) == 0 {
		t.Error("expected warnings about non-standard type")
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

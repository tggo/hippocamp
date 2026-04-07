package tools_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/rdfstore"
	"github.com/ruslanmv/hippocamp/internal/tools"
)

// ── helpers ─────────────────────────────────────────────────

func seedAnalyzeGraph(t *testing.T, s *rdfstore.Store) {
	t.Helper()
	// Build a small graph with two topics, cross-topic edges, and varying degrees.
	//
	// Topic: backend (auth-service, user-db, session-cache)
	// Topic: frontend (login-page, dashboard)
	// Cross-topic: login-page → references → auth-service
	//
	// auth-service is the "god node" — connected to user-db, session-cache, login-page

	triples := []struct {
		s, p, o, ot string
	}{
		// Types
		{"https://ex.org/topic/backend", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Topic", "uri"},
		{"https://ex.org/topic/frontend", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Topic", "uri"},
		{"https://ex.org/auth-service", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri"},
		{"https://ex.org/user-db", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri"},
		{"https://ex.org/session-cache", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri"},
		{"https://ex.org/login-page", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri"},
		{"https://ex.org/dashboard", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri"},

		// Labels
		{"https://ex.org/topic/backend", "http://www.w3.org/2000/01/rdf-schema#label", "Backend", "literal"},
		{"https://ex.org/topic/frontend", "http://www.w3.org/2000/01/rdf-schema#label", "Frontend", "literal"},
		{"https://ex.org/auth-service", "http://www.w3.org/2000/01/rdf-schema#label", "Auth Service", "literal"},
		{"https://ex.org/user-db", "http://www.w3.org/2000/01/rdf-schema#label", "User Database", "literal"},
		{"https://ex.org/session-cache", "http://www.w3.org/2000/01/rdf-schema#label", "Session Cache", "literal"},
		{"https://ex.org/login-page", "http://www.w3.org/2000/01/rdf-schema#label", "Login Page", "literal"},
		{"https://ex.org/dashboard", "http://www.w3.org/2000/01/rdf-schema#label", "Dashboard", "literal"},

		// Topics
		{"https://ex.org/auth-service", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/backend", "uri"},
		{"https://ex.org/user-db", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/backend", "uri"},
		{"https://ex.org/session-cache", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/backend", "uri"},
		{"https://ex.org/login-page", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/frontend", "uri"},
		{"https://ex.org/dashboard", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/frontend", "uri"},

		// Relationships (backend internal)
		{"https://ex.org/auth-service", "https://hippocamp.dev/ontology#references", "https://ex.org/user-db", "uri"},
		{"https://ex.org/auth-service", "https://hippocamp.dev/ontology#references", "https://ex.org/session-cache", "uri"},

		// Relationships (frontend internal)
		{"https://ex.org/dashboard", "https://hippocamp.dev/ontology#references", "https://ex.org/login-page", "uri"},

		// Cross-topic edge: login-page → auth-service
		{"https://ex.org/login-page", "https://hippocamp.dev/ontology#references", "https://ex.org/auth-service", "uri"},
	}

	for _, tr := range triples {
		if err := s.AddTriple("", tr.s, tr.p, tr.o, tr.ot, "", ""); err != nil {
			t.Fatalf("seed triple (%s %s %s): %v", tr.s, tr.p, tr.o, err)
		}
	}
}

// ── god_nodes ───────────────────────────────────────────────

func TestAnalyze_GodNodes_Empty(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	result := callTool(t, s, "analyze", map[string]any{"action": "god_nodes"})
	text := tools.ResultText(result)

	var nodes []json.RawMessage
	if err := json.Unmarshal([]byte(text), &nodes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("expected 0 god nodes on empty graph, got %d", len(nodes))
	}
}

func TestAnalyze_GodNodes_StarTopology(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()
	seedAnalyzeGraph(t, s)

	result := callTool(t, s, "analyze", map[string]any{"action": "god_nodes", "limit": float64(3)})
	text := tools.ResultText(result)
	t.Logf("god_nodes: %s", text)

	var nodes []struct {
		URI    string `json:"uri"`
		Label  string `json:"label"`
		Degree int    `json:"degree"`
	}
	if err := json.Unmarshal([]byte(text), &nodes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected at least 1 god node")
	}

	// auth-service should be #1 (connected to user-db, session-cache, login-page)
	if !strings.Contains(nodes[0].URI, "auth-service") {
		t.Errorf("expected auth-service as top god node, got %s", nodes[0].URI)
	}

	if len(nodes) > 3 {
		t.Errorf("limit=3 but got %d results", len(nodes))
	}
}

func TestAnalyze_GodNodes_ExcludesHubTypes(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()
	seedAnalyzeGraph(t, s)

	result := callTool(t, s, "analyze", map[string]any{"action": "god_nodes", "limit": float64(20)})
	text := tools.ResultText(result)

	var nodes []struct {
		URI  string `json:"uri"`
		Type string `json:"type"`
	}
	json.Unmarshal([]byte(text), &nodes)

	for _, n := range nodes {
		if n.Type == "Topic" || n.Type == "Tag" || n.Type == "Project" {
			t.Errorf("hub type %s should be excluded, found %s", n.Type, n.URI)
		}
	}
}

// ── components ──────────────────────────────────────────────

func TestAnalyze_Components_Empty(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	result := callTool(t, s, "analyze", map[string]any{"action": "components"})
	text := tools.ResultText(result)

	var comps []json.RawMessage
	json.Unmarshal([]byte(text), &comps)
	if len(comps) != 0 {
		t.Errorf("expected 0 components on empty graph, got %d", len(comps))
	}
}

func TestAnalyze_Components_ConnectedGraph(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()
	seedAnalyzeGraph(t, s)

	result := callTool(t, s, "analyze", map[string]any{"action": "components"})
	text := tools.ResultText(result)
	t.Logf("components: %s", text)

	var comps []struct {
		ID   int `json:"id"`
		Size int `json:"size"`
	}
	json.Unmarshal([]byte(text), &comps)

	// Everything is connected via hasTopic and references
	// so we expect one large component
	if len(comps) == 0 {
		t.Fatal("expected at least 1 component")
	}

	// The main component should have all 7 nodes (5 entities + 2 topics)
	if comps[0].Size < 5 {
		t.Errorf("expected main component to have >=5 nodes, got %d", comps[0].Size)
	}
}

func TestAnalyze_Components_Disconnected(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	// Two isolated clusters with no connection
	s.AddTriple("", "https://ex.org/a", "https://hippocamp.dev/ontology#references", "https://ex.org/b", "uri", "", "")
	s.AddTriple("", "https://ex.org/c", "https://hippocamp.dev/ontology#references", "https://ex.org/d", "uri", "", "")

	result := callTool(t, s, "analyze", map[string]any{"action": "components"})
	text := tools.ResultText(result)

	var comps []struct {
		Size int `json:"size"`
	}
	json.Unmarshal([]byte(text), &comps)

	if len(comps) != 2 {
		t.Errorf("expected 2 disconnected components, got %d", len(comps))
	}
}

// ── surprising ──────────────────────────────────────────────

func TestAnalyze_Surprising_NoCrossTopic(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	// Two entities with the SAME topic
	s.AddTriple("", "https://ex.org/a", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri", "", "")
	s.AddTriple("", "https://ex.org/b", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri", "", "")
	s.AddTriple("", "https://ex.org/a", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/x", "uri", "", "")
	s.AddTriple("", "https://ex.org/b", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/x", "uri", "", "")
	s.AddTriple("", "https://ex.org/a", "https://hippocamp.dev/ontology#references", "https://ex.org/b", "uri", "", "")

	result := callTool(t, s, "analyze", map[string]any{"action": "surprising"})
	text := tools.ResultText(result)

	var edges []json.RawMessage
	json.Unmarshal([]byte(text), &edges)
	if len(edges) != 0 {
		t.Errorf("expected 0 surprising edges (same topic), got %d", len(edges))
	}
}

func TestAnalyze_Surprising_CrossTopic(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()
	seedAnalyzeGraph(t, s)

	result := callTool(t, s, "analyze", map[string]any{"action": "surprising"})
	text := tools.ResultText(result)
	t.Logf("surprising: %s", text)

	var edges []struct {
		Subject string `json:"subject"`
		Object  string `json:"object"`
		Reason  string `json:"reason"`
	}
	json.Unmarshal([]byte(text), &edges)

	// login-page (frontend) → auth-service (backend) should be surprising
	found := false
	for _, e := range edges {
		if strings.Contains(e.Subject, "login-page") && strings.Contains(e.Object, "auth-service") {
			found = true
			if !strings.Contains(e.Reason, "cross-topic") {
				t.Errorf("expected cross-topic reason, got %s", e.Reason)
			}
		}
	}
	if !found {
		t.Error("expected login-page → auth-service as surprising edge")
	}
}

// ── export_html ─────────────────────────────────────────────

func TestAnalyze_ExportHTML(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()
	seedAnalyzeGraph(t, s)

	// Start the visualization server (as main.go would do on startup)
	port, err := tools.StartVisualizationServer(s)
	if err != nil {
		t.Fatalf("start viz server: %v", err)
	}
	t.Logf("viz server on port %d", port)

	result := callTool(t, s, "analyze", map[string]any{"action": "export_html"})
	text := tools.ResultText(result)
	t.Logf("export_html: %s", text)

	var resp struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.URL == "" {
		t.Fatal("expected a URL in response")
	}
	if resp.Status != "running" {
		t.Errorf("expected status=running, got %s", resp.Status)
	}

	// Fetch the page and verify it has vis.js content
	httpResp, err := http.Get(resp.URL)
	if err != nil {
		t.Fatalf("GET %s: %v", resp.URL, err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", httpResp.StatusCode)
	}

	buf := make([]byte, 8192)
	n, _ := httpResp.Body.Read(buf)
	body := string(buf[:n])

	if !strings.Contains(body, "vis-network") {
		t.Error("HTML should contain vis-network reference")
	}
	if !strings.Contains(body, "Hippocamp") {
		t.Error("HTML should contain Hippocamp title")
	}
	if !strings.Contains(body, "auth-service") || !strings.Contains(body, "Auth Service") {
		t.Error("HTML should contain node data")
	}
}

// ── consolidate ─────────────────────────────────────────────

func TestAnalyze_Consolidate_MissingSummary(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	// Entity with NO summary
	s.AddTriple("", "https://ex.org/svc", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri", "", "")
	s.AddTriple("", "https://ex.org/svc", "http://www.w3.org/2000/01/rdf-schema#label", "My Service", "literal", "", "")
	// Entity WITH summary (should not appear)
	s.AddTriple("", "https://ex.org/db", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri", "", "")
	s.AddTriple("", "https://ex.org/db", "http://www.w3.org/2000/01/rdf-schema#label", "Database", "literal", "", "")
	s.AddTriple("", "https://ex.org/db", "https://hippocamp.dev/ontology#summary", "Main PostgreSQL database for user data", "literal", "", "")
	s.AddTriple("", "https://ex.org/db", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/backend", "uri", "", "")
	// Relationship for context
	s.AddTriple("", "https://ex.org/svc", "https://hippocamp.dev/ontology#references", "https://ex.org/db", "uri", "", "")

	result := callTool(t, s, "analyze", map[string]any{"action": "consolidate"})
	text := tools.ResultText(result)
	t.Logf("consolidate: %s", text)

	var suggestions []struct {
		URI             string `json:"uri"`
		Issue           string `json:"issue"`
		SuggestedPrompt string `json:"suggested_prompt"`
	}
	if err := json.Unmarshal([]byte(text), &suggestions); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Should find svc (missing summary) but not db (has summary)
	found := false
	for _, s := range suggestions {
		if strings.Contains(s.URI, "svc") {
			found = true
			if s.Issue != "missing_summary" {
				t.Errorf("expected issue=missing_summary, got %s", s.Issue)
			}
			if !strings.Contains(s.SuggestedPrompt, "Database") {
				t.Errorf("expected prompt to mention referenced Database, got: %s", s.SuggestedPrompt)
			}
		}
		if strings.Contains(s.URI, "db") {
			t.Error("db has a summary, should not appear in consolidate results")
		}
	}
	if !found {
		t.Error("expected svc in consolidate results (missing summary)")
	}
}

func TestAnalyze_Consolidate_SparseSummary(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	s.AddTriple("", "https://ex.org/x", "http://www.w3.org/1999/02/22-rdf-syntax-ns#type", "https://hippocamp.dev/ontology#Entity", "uri", "", "")
	s.AddTriple("", "https://ex.org/x", "http://www.w3.org/2000/01/rdf-schema#label", "Widget", "literal", "", "")
	s.AddTriple("", "https://ex.org/x", "https://hippocamp.dev/ontology#summary", "A thing", "literal", "", "")
	s.AddTriple("", "https://ex.org/x", "https://hippocamp.dev/ontology#hasTopic", "https://ex.org/topic/main", "uri", "", "")

	result := callTool(t, s, "analyze", map[string]any{"action": "consolidate"})
	text := tools.ResultText(result)

	var suggestions []struct {
		Issue string `json:"issue"`
	}
	json.Unmarshal([]byte(text), &suggestions)

	found := false
	for _, s := range suggestions {
		if s.Issue == "sparse_summary" {
			found = true
		}
	}
	if !found {
		t.Error("expected sparse_summary issue for very short summary")
	}
}

func TestAnalyze_Consolidate_Empty(t *testing.T) {
	s := rdfstore.NewStore()
	defer s.Close()

	result := callTool(t, s, "analyze", map[string]any{"action": "consolidate"})
	text := tools.ResultText(result)

	var suggestions []json.RawMessage
	json.Unmarshal([]byte(text), &suggestions)
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions on empty graph, got %d", len(suggestions))
	}
}

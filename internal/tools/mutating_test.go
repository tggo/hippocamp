package tools

import "testing"

func TestIsMutatingCall(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		args     map[string]any
		expected bool
	}{
		// triple tool
		{"triple add", "triple", map[string]any{"action": "add"}, true},
		{"triple remove", "triple", map[string]any{"action": "remove"}, true},
		{"triple list", "triple", map[string]any{"action": "list"}, false},
		{"triple no action", "triple", map[string]any{}, false},

		// sparql tool
		{"sparql SELECT", "sparql", map[string]any{"query": "SELECT ?s ?p ?o WHERE { ?s ?p ?o }"}, false},
		{"sparql ASK", "sparql", map[string]any{"query": "ASK { ?s ?p ?o }"}, false},
		{"sparql INSERT DATA", "sparql", map[string]any{"query": "INSERT DATA { <s> <p> <o> }"}, true},
		{"sparql DELETE DATA", "sparql", map[string]any{"query": "DELETE DATA { <s> <p> <o> }"}, true},
		{"sparql PREFIX then INSERT", "sparql", map[string]any{"query": "PREFIX ex: <http://example.org/>\nINSERT DATA { ex:a ex:b ex:c }"}, true},
		{"sparql PREFIX then SELECT", "sparql", map[string]any{"query": "PREFIX ex: <http://example.org/>\nSELECT ?s WHERE { ?s ?p ?o }"}, false},
		{"sparql LOAD", "sparql", map[string]any{"query": "LOAD <http://example.org/data>"}, true},
		{"sparql CLEAR", "sparql", map[string]any{"query": "CLEAR DEFAULT"}, true},
		{"sparql DROP", "sparql", map[string]any{"query": "DROP GRAPH <g>"}, true},
		{"sparql CREATE", "sparql", map[string]any{"query": "CREATE GRAPH <g>"}, true},
		{"sparql COPY", "sparql", map[string]any{"query": "COPY DEFAULT TO <g>"}, true},
		{"sparql MOVE", "sparql", map[string]any{"query": "MOVE <g1> TO <g2>"}, true},
		{"sparql ADD", "sparql", map[string]any{"query": "ADD DEFAULT TO <g>"}, true},
		{"sparql empty query", "sparql", map[string]any{"query": ""}, false},
		{"sparql no query", "sparql", map[string]any{}, false},

		// graph tool
		{"graph create", "graph", map[string]any{"action": "create"}, true},
		{"graph delete", "graph", map[string]any{"action": "delete"}, true},
		{"graph clear", "graph", map[string]any{"action": "clear"}, true},
		{"graph load", "graph", map[string]any{"action": "load"}, true},
		{"graph import", "graph", map[string]any{"action": "import"}, true},
		{"graph list", "graph", map[string]any{"action": "list"}, false},
		{"graph stats", "graph", map[string]any{"action": "stats"}, false},
		{"graph dump", "graph", map[string]any{"action": "dump"}, false},
		{"graph no action", "graph", map[string]any{}, false},

		// non-mutating tools
		{"search", "search", map[string]any{"query": "foo"}, false},
		{"validate", "validate", map[string]any{}, false},
		{"unknown tool", "unknown", map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsMutatingCall(tt.tool, tt.args)
			if got != tt.expected {
				t.Errorf("IsMutatingCall(%q, %v) = %v, want %v", tt.tool, tt.args, got, tt.expected)
			}
		})
	}
}

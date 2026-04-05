package rdfstore

import (
	"fmt"
	"io"
	"strings"
	"sync"

	rdf "github.com/tggo/goRDFlib"
	"github.com/tggo/goRDFlib/graph"
	"github.com/tggo/goRDFlib/sparql"
	"github.com/tggo/goRDFlib/store/badgerstore"
	"github.com/tggo/goRDFlib/term"
	"github.com/tggo/goRDFlib/trig"
)

// DefaultGraphURI is the URI of the implicit default graph.
const DefaultGraphURI = "urn:hippocamp:default"

// Triple is a plain struct returned from queries (no goRDFlib types exposed).
type Triple struct {
	Subject   string
	Predicate string
	Object    string
	ObjType   string // "uri", "literal", "bnode"
	Lang      string
	Datatype  string
}

// Store wraps a context-aware goRDFlib Dataset (backed by BadgerDB in-memory)
// with dirty tracking and a write mutex.
type Store struct {
	mu       sync.RWMutex
	ds       *graph.Dataset
	bg       *badgerstore.BadgerStore // kept for Close()
	dirty    bool
	prefixes map[string]string // prefix → namespace URI
}

// NewStore creates an empty, context-aware Store with a default graph pre-created.
// Each call produces an independent in-memory store; no shared state between instances.
func NewStore() *Store {
	bg, err := badgerstore.New(badgerstore.WithInMemory())
	if err != nil {
		panic(fmt.Sprintf("hippocamp: cannot create in-memory Badger store: %v", err))
	}
	ds := rdf.NewDataset(rdf.WithStore(bg))
	s := &Store{
		ds:       ds,
		bg:       bg,
		prefixes: make(map[string]string),
	}
	// Ensure default graph exists.
	s.ds.Graph(rdf.NewURIRefUnsafe(DefaultGraphURI))
	return s
}

// Close releases the underlying Badger database. Call when the store is no longer needed.
func (s *Store) Close() error {
	return s.bg.Close()
}

// IsDirty reports whether any mutations occurred since the last ClearDirty.
func (s *Store) IsDirty() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dirty
}

// ClearDirty resets the dirty flag (called after a successful save).
func (s *Store) ClearDirty() {
	s.mu.Lock()
	s.dirty = false
	s.mu.Unlock()
}

// Dataset returns the underlying goRDFlib Dataset (for serialization).
func (s *Store) Dataset() *graph.Dataset {
	return s.ds
}

// resolveGraphID returns the URIRef for the given graph name.
// Empty name resolves to the default graph.
func resolveGraphID(name string) term.URIRef {
	if name == "" {
		return rdf.NewURIRefUnsafe(DefaultGraphURI)
	}
	return rdf.NewURIRefUnsafe(name)
}

// getGraph returns the named graph (creating it if absent).
func (s *Store) getGraph(name string) *graph.Graph {
	return s.ds.Graph(resolveGraphID(name))
}

// AddTriple adds a triple to the named graph (empty name = default graph).
// objectType: "uri", "literal", "bnode"
// lang: language tag (for literals)
// datatype: XSD datatype URI (for typed literals)
func (s *Store) AddTriple(graphName, subject, predicate, object, objectType, lang, datatype string) error {
	subj, err := buildSubject(subject)
	if err != nil {
		return fmt.Errorf("subject: %w", err)
	}
	pred, err := rdf.NewURIRef(predicate)
	if err != nil {
		return fmt.Errorf("predicate: %w", err)
	}
	obj, err := buildObject(object, objectType, lang, datatype)
	if err != nil {
		return fmt.Errorf("object: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.getGraph(graphName).Add(subj, pred, obj)
	s.dirty = true
	return nil
}

// RemoveTriple removes a specific triple from the named graph.
// objectType defaults to "uri" when empty.
func (s *Store) RemoveTriple(graphName, subject, predicate, object string) error {
	subj, err := buildSubject(subject)
	if err != nil {
		return fmt.Errorf("subject: %w", err)
	}
	pred, err := rdf.NewURIRef(predicate)
	if err != nil {
		return fmt.Errorf("predicate: %w", err)
	}
	obj, err := rdf.NewURIRef(object)
	if err != nil {
		return fmt.Errorf("object: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.getGraph(graphName).Remove(subj, &pred, obj)
	s.dirty = true
	return nil
}

// ListTriples returns triples from the named graph matching the pattern.
// Empty string means wildcard.
func (s *Store) ListTriples(graphName, subject, predicate, object string) ([]Triple, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g := s.getGraph(graphName)

	var subj term.Subject
	if subject != "" {
		u, err := rdf.NewURIRef(subject)
		if err != nil {
			return nil, fmt.Errorf("subject: %w", err)
		}
		subj = u
	}

	var pred *term.URIRef
	if predicate != "" {
		u, err := rdf.NewURIRef(predicate)
		if err != nil {
			return nil, fmt.Errorf("predicate: %w", err)
		}
		pred = &u
	}

	var obj term.Term
	if object != "" {
		u, err := rdf.NewURIRef(object)
		if err != nil {
			// treat as literal if not a valid URI
			obj = rdf.NewLiteral(object)
		} else {
			obj = u
		}
	}

	var results []Triple
	g.Triples(subj, pred, obj)(func(t term.Triple) bool {
		results = append(results, termToTriple(t))
		return true
	})
	return results, nil
}

// CreateGraph creates a named graph (no-op if already exists).
func (s *Store) CreateGraph(name string) error {
	if name == "" {
		return fmt.Errorf("graph name cannot be empty (use a URI)")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ds.Graph(resolveGraphID(name))
	s.dirty = true
	return nil
}

// DeleteGraph removes a named graph. Cannot delete the default graph.
func (s *Store) DeleteGraph(name string) error {
	if name == "" || name == DefaultGraphURI {
		return fmt.Errorf("cannot delete the default graph")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ds.RemoveGraph(resolveGraphID(name))
	s.dirty = true
	return nil
}

// ListGraphs returns the URIs of all named graphs.
// Filters out internal BNode graphs created by goRDFlib's ConjunctiveGraph.
func (s *Store) ListGraphs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var names []string
	s.ds.Graphs()(func(g *graph.Graph) bool {
		if id := g.Identifier(); id != nil {
			uri := id.String()
			// Skip BNode identifiers (internal goRDFlib default context).
			if _, ok := id.(term.BNode); ok {
				return true
			}
			names = append(names, uri)
		}
		return true
	})
	return names
}

// ClearGraph removes all triples from the named graph (graph itself remains).
func (s *Store) ClearGraph(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getGraph(name).Remove(nil, nil, nil)
	s.dirty = true
	return nil
}

// Stats returns basic statistics for the named graph.
func (s *Store) Stats(name string) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]int{"triples": s.getGraph(name).Len()}
}

// BindPrefix registers a namespace prefix.
func (s *Store) BindPrefix(prefix, uri string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefixes[prefix] = uri
	s.ds.ConjunctiveGraph.Bind(prefix, rdf.NewURIRefUnsafe(uri))
}

// RemovePrefix removes a namespace prefix.
func (s *Store) RemovePrefix(prefix string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.prefixes, prefix)
}

// ListPrefixes returns all registered prefixes.
func (s *Store) ListPrefixes() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.prefixes))
	for k, v := range s.prefixes {
		out[k] = v
	}
	return out
}

// SPARQLQuery executes a SPARQL SELECT/ASK/CONSTRUCT against the named graph.
// Empty graphName queries the default graph. All named graphs in the store
// are made available for GRAPH { } clauses in the query.
func (s *Store) SPARQLQuery(graphName, queryStr string) (*sparql.Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g := s.getGraph(graphName)

	// Parse the query so we can inject named graphs for GRAPH clause support.
	parsed, err := sparql.Parse(queryStr)
	if err != nil {
		return nil, fmt.Errorf("sparql parse: %w", err)
	}

	// Populate NamedGraphs from all graphs in the store so GRAPH <uri> { }
	// clauses can resolve against them.
	if parsed.NamedGraphs == nil {
		parsed.NamedGraphs = make(map[string]*graph.Graph)
	}
	s.ds.Graphs()(func(ng *graph.Graph) bool {
		if id := ng.Identifier(); id != nil {
			parsed.NamedGraphs[id.String()] = ng
		}
		return true
	})

	return sparql.EvalQuery(g, parsed, nil)
}

// SPARQLUpdate executes a SPARQL Update against the store.
// defaultGraph is used as the default graph; all named graphs are exposed.
func (s *Store) SPARQLUpdate(defaultGraphName, update string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultG := s.getGraph(defaultGraphName)
	named := make(map[string]*graph.Graph)

	s.ds.Graphs()(func(g *graph.Graph) bool {
		id := g.Identifier()
		if id != nil {
			named[id.String()] = g
		}
		return true
	})

	ds := &sparql.Dataset{
		Default:     defaultG,
		NamedGraphs: named,
	}

	if err := sparql.Update(ds, update); err != nil {
		return err
	}
	s.dirty = true
	return nil
}

// Import parses a TriG/Turtle string and adds all triples to the store.
// Triples without an explicit named graph go into the default graph.
func (s *Store) Import(data string) (int, error) {
	// Parse into a temporary dataset.
	tmpBg, err := badgerstore.New(badgerstore.WithInMemory())
	if err != nil {
		return 0, fmt.Errorf("import: create temp store: %w", err)
	}
	tmpDs := rdf.NewDataset(rdf.WithStore(tmpBg))

	if err := trig.ParseDataset(tmpDs, io.Reader(strings.NewReader(data))); err != nil {
		tmpBg.Close()
		return 0, fmt.Errorf("import: parse: %w", err)
	}

	// Copy all triples from the temp dataset into our store.
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	tmpDs.Graphs()(func(g *graph.Graph) bool {
		// Determine target graph: named graphs keep their URI, default goes to our default.
		var targetID string
		if id := g.Identifier(); id != nil {
			if _, ok := id.(term.BNode); ok {
				targetID = "" // BNode = default context → our default graph
			} else {
				targetID = id.String()
			}
		}
		target := s.getGraph(targetID)

		g.Triples(nil, nil, nil)(func(t term.Triple) bool {
			target.Add(t.Subject, t.Predicate, t.Object)
			count++
			return true
		})
		return true
	})

	if count > 0 {
		s.dirty = true
	}
	tmpBg.Close()
	return count, nil
}

// --- helpers ---

func buildSubject(s string) (term.Subject, error) {
	u, err := rdf.NewURIRef(s)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func buildObject(value, objType, lang, datatype string) (term.Term, error) {
	switch objType {
	case "uri", "":
		u, err := rdf.NewURIRef(value)
		if err != nil {
			return nil, err
		}
		return u, nil
	case "literal":
		var opts []rdf.LiteralOption
		if lang != "" {
			opts = append(opts, rdf.WithLang(lang))
		} else if datatype != "" {
			dt, err := rdf.NewURIRef(datatype)
			if err != nil {
				return nil, fmt.Errorf("datatype: %w", err)
			}
			opts = append(opts, rdf.WithDatatype(dt))
		}
		return rdf.NewLiteral(value, opts...), nil
	case "bnode":
		return rdf.NewBNode(value), nil
	default:
		return nil, fmt.Errorf("unknown object_type %q; use uri, literal, or bnode", objType)
	}
}

func termToTriple(t term.Triple) Triple {
	tr := Triple{
		Subject:   t.Subject.String(),
		Predicate: t.Predicate.String(),
	}
	switch o := t.Object.(type) {
	case term.URIRef:
		tr.Object = o.String()
		tr.ObjType = "uri"
	case term.Literal:
		tr.Object = o.Lexical()
		tr.ObjType = "literal"
		tr.Lang = o.Language()
		if o.Datatype() != (term.URIRef{}) {
			tr.Datatype = o.Datatype().String()
		}
	case term.BNode:
		tr.Object = o.String()
		tr.ObjType = "bnode"
	default:
		tr.Object = t.Object.String()
		tr.ObjType = "uri"
	}
	return tr
}

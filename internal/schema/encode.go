package schema

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// EncodeOptions controls how a graph is serialized.
type EncodeOptions struct {
	Project     Project
	GeneratedAt time.Time
	// Audit is an optional audit result embedded in the document.
	Audit *AuditDTO
}

// Encode converts g to a deterministic JSON Document.
// Nodes are sorted by ID; edges are sorted by (From, To).
func Encode(g *graph.Graph, opts EncodeOptions) ([]byte, error) {
	if opts.GeneratedAt.IsZero() {
		opts.GeneratedAt = time.Now().UTC()
	}

	nodes := g.Nodes() // already sorted by ID
	nodeDTOs := make([]NodeDTO, len(nodes))
	for i, n := range nodes {
		nodeDTOs[i] = nodeToDTO(n)
	}

	edges := g.Edges() // already sorted by (From, To)
	edgeDTOs := make([]EdgeDTO, len(edges))
	for i, e := range edges {
		edgeDTOs[i] = edgeToDTO(e)
	}

	s := g.Stats()
	doc := Document{
		SchemaVersion: Version,
		GeneratedAt:   opts.GeneratedAt.UTC(),
		Project:       opts.Project,
		Nodes:         nodeDTOs,
		Edges:         edgeDTOs,
		Stats: StatsDTO{
			NodeCount: s.NodeCount,
			EdgeCount: s.EdgeCount,
			MaxDepth:  s.MaxDepth,
			HasCycles: s.HasCycles,
		},
		Audit: opts.Audit,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}

	// json.Encoder appends a trailing newline; strip it for predictable bytes.
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func nodeToDTO(n graph.Node) NodeDTO {
	dto := NodeDTO{
		ID:       string(n.ID),
		Name:     n.Name,
		Version:  n.Version,
		Kind:     string(n.Kind),
		Indirect: n.Indirect,
		Metadata: n.Metadata,
	}
	if n.ReplacedBy != nil {
		dto.ReplacedBy = &ReplacementDTO{
			Path:    n.ReplacedBy.Path,
			Version: n.ReplacedBy.Version,
		}
	}
	return dto
}

func edgeToDTO(e graph.Edge) EdgeDTO {
	return EdgeDTO{
		From:     string(e.From),
		To:       string(e.To),
		Kind:     string(e.Kind),
		Metadata: e.Metadata,
	}
}

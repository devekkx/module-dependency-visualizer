package schema

import (
	"encoding/json"
	"fmt"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// Decode parses a JSON document and reconstructs a Graph and its Project metadata.
// Returns an error if the schema_version is not supported.
func Decode(data []byte) (*graph.Graph, Project, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, Project{}, fmt.Errorf("schema: unmarshal: %w", err)
	}

	if doc.SchemaVersion != Version {
		return nil, Project{}, fmt.Errorf("schema: unsupported version %q (want %q)", doc.SchemaVersion, Version)
	}

	b := graph.NewBuilder()

	for _, dto := range doc.Nodes {
		n := dtoToNode(dto)
		if err := b.AddNode(n); err != nil {
			return nil, Project{}, fmt.Errorf("schema: add node %q: %w", dto.ID, err)
		}
	}

	for _, dto := range doc.Edges {
		e := dtoToEdge(dto)
		if err := b.AddEdge(e); err != nil {
			return nil, Project{}, fmt.Errorf("schema: add edge %q->%q: %w", dto.From, dto.To, err)
		}
	}

	g, err := b.Build()
	if err != nil {
		return nil, Project{}, fmt.Errorf("schema: build graph: %w", err)
	}

	return g, doc.Project, nil
}

func dtoToNode(dto NodeDTO) graph.Node {
	n := graph.Node{
		ID:       graph.NodeID(dto.ID),
		Name:     dto.Name,
		Version:  dto.Version,
		Kind:     graph.NodeKind(dto.Kind),
		Indirect: dto.Indirect,
		Metadata: dto.Metadata,
	}
	if dto.ReplacedBy != nil {
		n.ReplacedBy = &graph.Replacement{
			Path:    dto.ReplacedBy.Path,
			Version: dto.ReplacedBy.Version,
		}
	}
	return n
}

func dtoToEdge(dto EdgeDTO) graph.Edge {
	return graph.Edge{
		From:     graph.NodeID(dto.From),
		To:       graph.NodeID(dto.To),
		Kind:     graph.EdgeKind(dto.Kind),
		Metadata: dto.Metadata,
	}
}

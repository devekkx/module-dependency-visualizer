package mermaid

import (
	"context"
	"fmt"
	"io"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// Exporter writes a graph as a Mermaid flowchart.
//
// Mermaid node IDs must match [A-Za-z0-9_]+, so module paths (which contain
// dots, slashes, and @) cannot be used directly. We assign stable aliases
// n0, n1, n2, ... in sorted node ID order and render labels separately.
type Exporter struct{}

// New returns a Mermaid Exporter.
func New() *Exporter { return &Exporter{} }

// Format returns "mermaid".
func (e *Exporter) Format() string { return "mermaid" }

// Write serializes g to w as a Mermaid graph TD diagram.
func (e *Exporter) Write(_ context.Context, w io.Writer, g *graph.Graph) error {
	nodes := g.Nodes() // sorted by ID for determinism
	alias := make(map[graph.NodeID]string, len(nodes))
	for i, n := range nodes {
		alias[n.ID] = fmt.Sprintf("n%d", i)
	}

	if _, err := fmt.Fprintln(w, "graph TD"); err != nil {
		return err
	}

	for _, n := range nodes {
		label := fmt.Sprintf("%s@%s", n.Name, n.Version)
		line := fmt.Sprintf("  %s[\"%s\"]", alias[n.ID], sanitizeLabel(label))
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	for _, edge := range g.Edges() {
		from := alias[edge.From]
		to := alias[edge.To]

		toNode, _ := g.Node(edge.To)
		arrow := "-->"
		if toNode.Indirect {
			arrow = "-.->"
		}

		line := fmt.Sprintf("  %s %s %s", from, arrow, to)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return nil
}

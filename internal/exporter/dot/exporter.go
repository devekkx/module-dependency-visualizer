package dot

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// Exporter writes a graph as Graphviz DOT syntax.
type Exporter struct{}

// New returns a DOT Exporter.
func New() *Exporter { return &Exporter{} }

// Format returns "dot".
func (e *Exporter) Format() string { return "dot" }

// Write serializes g to w as a Graphviz digraph.
// Output is deterministic: nodes are sorted by ID, edges by (From, To).
func (e *Exporter) Write(_ context.Context, w io.Writer, g *graph.Graph) error {
	if _, err := fmt.Fprintln(w, `digraph deps {`); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, `  rankdir="LR";`); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, `  node [shape=box fontname="Helvetica"];`); err != nil {
		return err
	}

	for _, n := range g.Nodes() {
		attrs := nodeAttrs(n)
		line := fmt.Sprintf("  %s%s;", escapeID(string(n.ID)), attrs)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	for _, edge := range g.Edges() {
		style := ""
		if isIndirectEdge(g, edge) {
			style = ` [style="dashed"]`
		}
		line := fmt.Sprintf("  %s -> %s%s;",
			escapeID(string(edge.From)),
			escapeID(string(edge.To)),
			style,
		)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	_, err := fmt.Fprintln(w, "}")
	return err
}

func nodeAttrs(n graph.Node) string {
	label := fmt.Sprintf("%s\\n%s", n.Name, n.Version)
	attrs := []string{fmt.Sprintf(`label="%s"`, escapeLabel(label))}

	if n.Kind == graph.NodeKindMain {
		attrs = append(attrs, `style="bold"`)
	}
	if n.Indirect {
		attrs = append(attrs, `style="dashed"`)
	}

	return " [" + strings.Join(attrs, " ") + "]"
}

func isIndirectEdge(g *graph.Graph, e graph.Edge) bool {
	toNode, ok := g.Node(e.To)
	return ok && toNode.Indirect
}

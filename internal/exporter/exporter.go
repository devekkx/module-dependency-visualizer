package exporter

import (
	"context"
	"fmt"
	"io"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// Exporter serializes a graph to a specific text format.
type Exporter interface {
	// Format returns the short format name, e.g. "dot", "mermaid", "json".
	Format() string

	// Write serializes g to w.
	Write(ctx context.Context, w io.Writer, g *graph.Graph) error
}

// Registry maps format names to Exporters.
type Registry struct {
	exporters map[string]Exporter
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{exporters: make(map[string]Exporter)}
}

// Register adds e to the registry. Returns an error if Format() is already registered.
func (r *Registry) Register(e Exporter) error {
	if _, exists := r.exporters[e.Format()]; exists {
		return fmt.Errorf("exporter: %q is already registered", e.Format())
	}
	r.exporters[e.Format()] = e
	return nil
}

// Get returns the Exporter for the given format name, or an error if unknown.
func (r *Registry) Get(format string) (Exporter, error) {
	e, ok := r.exporters[format]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownFormat, format)
	}
	return e, nil
}

// ErrUnknownFormat is returned when the requested format has no registered exporter.
var ErrUnknownFormat = fmt.Errorf("exporter: unknown format")

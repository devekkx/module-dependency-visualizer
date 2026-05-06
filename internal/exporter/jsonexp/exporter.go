package jsonexp

import (
	"context"
	"fmt"
	"io"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/schema"
)

// Exporter writes a graph as JSON Schema v1.0.0.
type Exporter struct {
	opts schema.EncodeOptions
}

// New returns a JSON Exporter with the given project metadata.
func New(opts schema.EncodeOptions) *Exporter {
	return &Exporter{opts: opts}
}

// Format returns "json".
func (e *Exporter) Format() string { return "json" }

// Write serializes g to w as JSON Schema v1.0.0.
func (e *Exporter) Write(_ context.Context, w io.Writer, g *graph.Graph) error {
	data, err := schema.Encode(g, e.opts)
	if err != nil {
		return fmt.Errorf("json exporter: %w", err)
	}
	_, err = w.Write(data)
	return err
}

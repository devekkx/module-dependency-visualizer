package provider

import (
	"context"

	"module-dependency-visualizer/internal/graph"
)

// ParseOptions configures how a provider parses a project.
type ParseOptions struct {
	// MaxDepth limits traversal depth. -1 means unlimited.
	MaxDepth int
	// NoIndirect omits indirect dependencies.
	NoIndirect bool
}

// Project describes the analyzed project.
type Project struct {
	Name     string
	Language string
	RootPath string
	MainModule string
}

// Provider knows how to detect and parse a specific ecosystem's dependency manifest.
type Provider interface {
	// Name returns the short ecosystem name, e.g. "go", "npm", "python".
	Name() string

	// Detect reports whether the given rootPath contains a manifest this provider
	// can parse. It must not modify any files.
	Detect(ctx context.Context, rootPath string) (bool, error)

	// Parse reads the dependency manifest(s) at rootPath and returns an immutable
	// graph plus project metadata.
	Parse(ctx context.Context, rootPath string, opts ParseOptions) (*graph.Graph, Project, error)
}

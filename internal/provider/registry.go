package provider

import (
	"context"
	"fmt"
)

// Registry holds the set of registered providers and detects which one applies
// to a given project directory.
type Registry struct {
	providers []Provider
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds p to the registry. Returns an error if a provider with the same
// Name() is already registered.
func (r *Registry) Register(p Provider) error {
	for _, existing := range r.providers {
		if existing.Name() == p.Name() {
			return fmt.Errorf("provider: %q is already registered", p.Name())
		}
	}
	r.providers = append(r.providers, p)
	return nil
}

// Detect iterates over registered providers and returns the first one whose
// Detect() returns true. Returns ErrNoProvider if none match.
func (r *Registry) Detect(ctx context.Context, rootPath string) (Provider, error) {
	for _, p := range r.providers {
		ok, err := p.Detect(ctx, rootPath)
		if err != nil {
			return nil, fmt.Errorf("provider %q detect: %w", p.Name(), err)
		}
		if ok {
			return p, nil
		}
	}
	return nil, fmt.Errorf("%w: no provider recognizes %q", ErrNoProvider, rootPath)
}

// ErrNoProvider is returned when no registered provider supports the target path.
var ErrNoProvider = fmt.Errorf("provider: unsupported project")

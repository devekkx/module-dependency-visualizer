package provider_test

import (
	"context"
	"errors"
	"testing"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
)

// stubProvider is a test double that controls Detect behavior.
type stubProvider struct {
	name      string
	detectOK  bool
	detectErr error
}

func (s *stubProvider) Name() string { return s.name }

func (s *stubProvider) Detect(_ context.Context, _ string) (bool, error) {
	return s.detectOK, s.detectErr
}

func (s *stubProvider) Parse(_ context.Context, _ string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	return nil, provider.Project{}, nil
}

func TestRegistry_Register_RejectsDuplicate(t *testing.T) {
	r := provider.NewRegistry()

	if err := r.Register(&stubProvider{name: "go"}); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}
	if err := r.Register(&stubProvider{name: "go"}); err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}
}

func TestRegistry_Detect_ReturnsMatchingProvider(t *testing.T) {
	r := provider.NewRegistry()
	_ = r.Register(&stubProvider{name: "npm", detectOK: false})
	_ = r.Register(&stubProvider{name: "go", detectOK: true})

	p, err := r.Detect(context.Background(), "/some/path")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if p.Name() != "go" {
		t.Errorf("Detect() = %q; want %q", p.Name(), "go")
	}
}

func TestRegistry_Detect_NoProviderMatch(t *testing.T) {
	r := provider.NewRegistry()
	_ = r.Register(&stubProvider{name: "go", detectOK: false})

	_, err := r.Detect(context.Background(), "/no/match")
	if !errors.Is(err, provider.ErrNoProvider) {
		t.Errorf("expected ErrNoProvider, got %v", err)
	}
}

func TestRegistry_Detect_EmptyRegistry(t *testing.T) {
	r := provider.NewRegistry()
	_, err := r.Detect(context.Background(), "/any")
	if !errors.Is(err, provider.ErrNoProvider) {
		t.Errorf("empty registry: expected ErrNoProvider, got %v", err)
	}
}

func TestRegistry_Detect_PropagatesDetectError(t *testing.T) {
	r := provider.NewRegistry()
	detectErr := errors.New("filesystem unavailable")
	_ = r.Register(&stubProvider{name: "go", detectErr: detectErr})

	_, err := r.Detect(context.Background(), "/err/path")
	if err == nil {
		t.Fatal("expected error from Detect, got nil")
	}
	if !errors.Is(err, detectErr) {
		t.Errorf("expected wrapped detect error, got %v", err)
	}
}

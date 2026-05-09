package exporter_test

import (
	"errors"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/exporter"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/dot"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/mermaid"
)

func TestRegistry_Register_RejectsDuplicate(t *testing.T) {
	r := exporter.NewRegistry()
	if err := r.Register(dot.New()); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := r.Register(dot.New()); err == nil {
		t.Error("expected error for duplicate registration, got nil")
	}
}

func TestRegistry_Get_KnownFormat(t *testing.T) {
	r := exporter.NewRegistry()
	_ = r.Register(dot.New())
	_ = r.Register(mermaid.New())

	e, err := r.Get("dot")
	if err != nil {
		t.Fatalf("Get('dot'): %v", err)
	}
	if e.Format() != "dot" {
		t.Errorf("Format() = %q; want 'dot'", e.Format())
	}
}

func TestRegistry_Get_UnknownFormat(t *testing.T) {
	r := exporter.NewRegistry()
	_, err := r.Get("csv")
	if !errors.Is(err, exporter.ErrUnknownFormat) {
		t.Errorf("expected ErrUnknownFormat, got %v", err)
	}
}

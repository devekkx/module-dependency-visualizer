package golang

import (
	"testing"
)

func TestNormalizeRef(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"example.com/app", "example.com/app@"},
		{"example.com/app@v1.0.0", "example.com/app@v1.0.0"},
		{"github.com/x/y@v0.0.0-20240101-abc", "github.com/x/y@v0.0.0-20240101-abc"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeRef(tc.input)
			if got != tc.want {
				t.Errorf("normalizeRef(%q) = %q; want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestSplitRef(t *testing.T) {
	cases := []struct {
		input       string
		wantPath    string
		wantVersion string
	}{
		{"example.com/app@", "example.com/app", ""},
		{"example.com/app@v1.0.0", "example.com/app", "v1.0.0"},
		{"example.com/app", "example.com/app", ""},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			path, version := splitRef(tc.input)
			if path != tc.wantPath || version != tc.wantVersion {
				t.Errorf("splitRef(%q) = (%q, %q); want (%q, %q)",
					tc.input, path, version, tc.wantPath, tc.wantVersion)
			}
		})
	}
}

func TestBuildGraph_WithReplaceDirective(t *testing.T) {
	edges := []ModGraphEdge{
		{From: "myapp", To: "github.com/old/pkg@v1.0.0"},
	}
	infos := []ModuleInfo{
		{Path: "myapp", Main: true},
		{
			Path:    "github.com/old/pkg",
			Version: "v1.0.0",
			Replace: &ModuleInfo{Path: "github.com/new/pkg", Version: "v2.0.0"},
		},
	}

	g, mainModule, err := buildGraph(edges, infos)
	if err != nil {
		t.Fatalf("buildGraph: %v", err)
	}

	if mainModule != "myapp" {
		t.Errorf("mainModule = %q; want %q", mainModule, "myapp")
	}

	n, ok := g.Node("github.com/old/pkg@v1.0.0")
	if !ok {
		t.Fatal("expected node github.com/old/pkg@v1.0.0 not found")
	}
	if n.ReplacedBy == nil {
		t.Fatal("expected ReplacedBy to be set for replaced module")
	}
	if n.ReplacedBy.Path != "github.com/new/pkg" {
		t.Errorf("ReplacedBy.Path = %q; want %q", n.ReplacedBy.Path, "github.com/new/pkg")
	}
}

func TestFindGoBinary_GoBinaryPresent(t *testing.T) {
	// The test environment must have Go on PATH for this test to work.
	path, err := findGoBinary()
	if err != nil {
		t.Skipf("go binary not on PATH: %v", err)
	}
	if path == "" {
		t.Error("findGoBinary returned empty path")
	}
}

func TestFindGoBinary_BinaryNotFound(t *testing.T) {
	orig := lookPath
	defer func() { lookPath = orig }()

	lookPath = func(_ string) (string, error) {
		return "", &notFoundError{}
	}

	_, err := findGoBinary()
	if err == nil {
		t.Error("expected error when go binary is not found")
	}
}

type notFoundError struct{}

func (e *notFoundError) Error() string { return "not found" }

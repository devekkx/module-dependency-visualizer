package schema_test

import (
	"encoding/json"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"testing"
	"time"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/schema"
)

var update = flag.Bool("update", false, "Update golden files")

// fixedTime is used to make test output deterministic.
var fixedTime = time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

func buildTestGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()

	main := graph.Node{
		ID:      "example.com/app@",
		Name:    "example.com/app",
		Version: "",
		Kind:    graph.NodeKindMain,
	}
	cobra := graph.Node{
		ID:      "github.com/spf13/cobra@v1.8.0",
		Name:    "github.com/spf13/cobra",
		Version: "v1.8.0",
		Kind:    graph.NodeKindModule,
	}
	pflag := graph.Node{
		ID:       "github.com/spf13/pflag@v1.0.9",
		Name:     "github.com/spf13/pflag",
		Version:  "v1.0.9",
		Kind:     graph.NodeKindModule,
		Indirect: true,
	}

	for _, n := range []graph.Node{main, cobra, pflag} {
		if err := b.AddNode(n); err != nil {
			t.Fatal(err)
		}
	}
	for _, e := range []graph.Edge{
		{From: "example.com/app@", To: "github.com/spf13/cobra@v1.8.0", Kind: graph.EdgeKindDependsOn},
		{From: "github.com/spf13/cobra@v1.8.0", To: "github.com/spf13/pflag@v1.0.9", Kind: graph.EdgeKindDependsOn},
	} {
		if err := b.AddEdge(e); err != nil {
			t.Fatal(err)
		}
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func testOpts() schema.EncodeOptions {
	return schema.EncodeOptions{
		Project: schema.Project{
			Name:       "example.com/app",
			Language:   "go",
			RootPath:   "/project",
			MainModule: "example.com/app",
		},
		GeneratedAt: fixedTime,
	}
}

func TestEncode_Golden(t *testing.T) {
	g := buildTestGraph(t)
	got, err := schema.Encode(g, testOpts())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	goldenPath := filepath.Join("..", "..", "testdata", "fixtures", "golden", "simple.json")

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0644); err != nil {
			t.Fatalf("writing golden file: %v", err)
		}
		t.Logf("Updated golden file: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("Reading golden file %s: %v\nRun with -update to create it.", goldenPath, err)
	}

	if string(got) != string(want) {
		t.Errorf("Encode output does not match golden file.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestEncode_Deterministic(t *testing.T) {
	g := buildTestGraph(t)
	opts := testOpts()

	first, err := schema.Encode(g, opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := schema.Encode(g, opts)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Error("Encode() is not deterministic — two calls produced different output")
	}
}

func TestEncode_ValidJSON(t *testing.T) {
	g := buildTestGraph(t)
	data, err := schema.Encode(g, testOpts())
	if err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Encode produced invalid JSON: %v\n%s", err, data)
	}

	if out["schema_version"] != schema.Version {
		t.Errorf("schema_version = %v; want %q", out["schema_version"], schema.Version)
	}
}

func TestDecode_RoundTrip(t *testing.T) {
	g := buildTestGraph(t)
	opts := testOpts()

	encoded, err := schema.Encode(g, opts)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, proj, err := schema.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if proj.Name != opts.Project.Name {
		t.Errorf("Project.Name = %q; want %q", proj.Name, opts.Project.Name)
	}

	reEncoded, err := schema.Encode(decoded, opts)
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}

	if string(encoded) != string(reEncoded) {
		t.Errorf("Round-trip produced different bytes.\nOriginal:\n%s\nRe-encoded:\n%s", encoded, reEncoded)
	}
}

func TestDecode_UnsupportedVersion(t *testing.T) {
	data := []byte(`{"schema_version":"99.0.0","project":{},"nodes":[],"edges":[],"stats":{}}`)
	_, _, err := schema.Decode(data)
	if err == nil {
		t.Error("expected error for unsupported schema_version")
	}
}

func TestDecode_InvalidJSON(t *testing.T) {
	_, _, err := schema.Decode([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecode_UnknownEdgeNode(t *testing.T) {
	data := []byte(`{
		"schema_version":"1.0.0",
		"project":{},
		"nodes":[{"id":"a@v1","name":"a","version":"v1","kind":"module"}],
		"edges":[{"from":"a@v1","to":"UNKNOWN@v1","kind":"depends_on"}],
		"stats":{}
	}`)
	_, _, err := schema.Decode(data)
	if err == nil {
		t.Error("expected error for edge referencing unknown node")
	}
	if !errors.Is(err, graph.ErrUnknownNode) {
		t.Errorf("expected ErrUnknownNode in error chain, got: %v", err)
	}
}

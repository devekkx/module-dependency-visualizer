package golang_test

import (
	"os"
	"path/filepath"
	"testing"

	goprovider "module-dependency-visualizer/internal/provider/golang"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "testdata", "fixtures", "go-simple", name))
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return data
}

func TestParseModGraph_Simple(t *testing.T) {
	data := readFixture(t, "modgraph.txt")
	edges, err := goprovider.ParseModGraph(data)
	if err != nil {
		t.Fatalf("ParseModGraph: %v", err)
	}

	if len(edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(edges))
	}

	cases := []struct {
		from, to string
	}{
		{"example.com/app", "github.com/spf13/cobra@v1.8.0"},
		{"github.com/spf13/cobra@v1.8.0", "github.com/spf13/pflag@v1.0.9"},
	}

	for i, tc := range cases {
		if edges[i].From != tc.from {
			t.Errorf("edge[%d].From = %q; want %q", i, edges[i].From, tc.from)
		}
		if edges[i].To != tc.to {
			t.Errorf("edge[%d].To = %q; want %q", i, edges[i].To, tc.to)
		}
	}
}

func TestParseModGraph_InvalidLine(t *testing.T) {
	_, err := goprovider.ParseModGraph([]byte("single-field-only\n"))
	if err == nil {
		t.Error("expected error for invalid line format")
	}
}

func TestParseModGraph_Empty(t *testing.T) {
	edges, err := goprovider.ParseModGraph([]byte(""))
	if err != nil {
		t.Fatalf("ParseModGraph: %v", err)
	}
	if len(edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(edges))
	}
}

func TestParseModList_Simple(t *testing.T) {
	data := readFixture(t, "modlist.json")
	infos, err := goprovider.ParseModList(data)
	if err != nil {
		t.Fatalf("ParseModList: %v", err)
	}

	if len(infos) != 3 {
		t.Fatalf("expected 3 infos, got %d", len(infos))
	}

	mainCount := 0
	for _, info := range infos {
		if info.Main {
			mainCount++
		}
	}
	if mainCount != 1 {
		t.Errorf("expected 1 main module, got %d", mainCount)
	}
}

func TestParseModList_IndirectFlag(t *testing.T) {
	data := readFixture(t, "modlist.json")
	infos, err := goprovider.ParseModList(data)
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range infos {
		if info.Path == "github.com/spf13/pflag" && !info.Indirect {
			t.Error("pflag should be marked as indirect")
		}
		if info.Path == "github.com/spf13/cobra" && info.Indirect {
			t.Error("cobra should not be marked as indirect")
		}
	}
}

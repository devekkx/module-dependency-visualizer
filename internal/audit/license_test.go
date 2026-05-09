package audit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devekkx/module-dependency-visualizer/internal/audit"
	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

func TestSystemFor(t *testing.T) {
	cases := []struct {
		language string
		system   string
	}{
		{"go", "go"},
		{"Go", "go"},
		{"node", "npm"},
		{"python", "pypi"},
		{"Python", "pypi"},
		{"ruby", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := audit.SystemFor(tc.language)
		if got != tc.system {
			t.Errorf("SystemFor(%q) = %q, want %q", tc.language, got, tc.system)
		}
	}
}

func TestLicenseScan_FetchesLicenses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"version": map[string]any{
				"licenses": []string{"MIT"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/pkg@v1.0.0", Name: "github.com/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	a := audit.NewWithEndpoints(audit.Options{SkipVulnScan: true, SkipConflicts: true}, "", srv.URL+"/v3alpha")
	result, err := a.Run(context.Background(), g, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lic, ok := result.Licenses["github.com/pkg@v1.0.0"]
	if !ok {
		t.Fatal("expected license for github.com/pkg@v1.0.0")
	}
	if lic != "MIT" {
		t.Errorf("license = %q, want MIT", lic)
	}
}

func TestLicenseScan_UnknownOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/pkg@v1.0.0", Name: "github.com/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	a := audit.NewWithEndpoints(audit.Options{SkipVulnScan: true, SkipConflicts: true}, "", srv.URL+"/v3alpha")
	result, err := a.Run(context.Background(), g, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lic := result.Licenses["github.com/pkg@v1.0.0"]
	if lic != "Unknown" {
		t.Errorf("expected Unknown license on server error, got %q", lic)
	}
}

func TestLicenseScan_UnsupportedLanguage(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()

	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "some/pkg@v1.0.0", Name: "some/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	a := audit.NewWithEndpoints(audit.Options{SkipVulnScan: true, SkipConflicts: true}, "", srv.URL+"/v3alpha")
	result, err := a.Run(context.Background(), g, "ruby")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no HTTP call for unsupported language")
	}
	if len(result.Licenses) != 0 {
		t.Errorf("expected empty licenses map, got %v", result.Licenses)
	}
}

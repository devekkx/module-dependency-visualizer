package audit_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"module-dependency-visualizer/internal/audit"
	"module-dependency-visualizer/internal/graph"
)

func TestEcosystemFor(t *testing.T) {
	cases := []struct {
		language  string
		ecosystem string
	}{
		{"go", "Go"},
		{"Go", "Go"},
		{"node", "npm"},
		{"python", "PyPI"},
		{"Python", "PyPI"},
		{"ruby", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := audit.EcosystemFor(tc.language)
		if got != tc.ecosystem {
			t.Errorf("EcosystemFor(%q) = %q, want %q", tc.language, got, tc.ecosystem)
		}
	}
}

func TestOSVScan_EmptyEcosystem(t *testing.T) {
	a := audit.New(audit.Options{SkipLicense: true, SkipConflicts: true})
	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/foo@v1.0.0", Name: "github.com/foo", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})
	result, err := a.Run(context.Background(), g, "ruby") // unsupported
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Vulnerabilities) != 0 {
		t.Errorf("expected no vulnerabilities for unsupported language, got %d", len(result.Vulnerabilities))
	}
}

func TestOSVScan_WithVulnerabilities(t *testing.T) {
	// Mock OSV server returning one vulnerability for the first package.
	response := map[string]any{
		"results": []any{
			map[string]any{
				"vulns": []any{
					map[string]any{
						"id":      "GHSA-test-1234-5678",
						"summary": "Test vulnerability",
						"affected": []any{
							map[string]any{
								"ranges": []any{
									map[string]any{
										"events": []any{
											map[string]any{"introduced": "0"},
											map[string]any{"fixed": "v1.1.0"},
										},
									},
								},
							},
						},
						"database_specific": map[string]any{
							"severity": "HIGH",
						},
					},
				},
			},
			// Second package has no vulns.
			map[string]any{"vulns": nil},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer srv.Close()

	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/vulnerable@v1.0.0", Name: "github.com/vulnerable", Version: "v1.0.0", Kind: graph.NodeKindModule},
		{ID: "github.com/safe@v2.0.0", Name: "github.com/safe", Version: "v2.0.0", Kind: graph.NodeKindModule},
	})

	a := audit.NewWithEndpoints(audit.Options{SkipLicense: true, SkipConflicts: true}, srv.URL+"/v1/querybatch", "")
	result, err := a.Run(context.Background(), g, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Vulnerabilities) != 1 {
		t.Fatalf("expected 1 vulnerability, got %d", len(result.Vulnerabilities))
	}
	v := result.Vulnerabilities[0]
	if v.ID != "GHSA-test-1234-5678" {
		t.Errorf("vuln ID = %q, want %q", v.ID, "GHSA-test-1234-5678")
	}
	if v.Severity != "HIGH" {
		t.Errorf("severity = %q, want HIGH", v.Severity)
	}
	if v.FixedIn != "v1.1.0" {
		t.Errorf("fixed_in = %q, want v1.1.0", v.FixedIn)
	}
}

func TestOSVScan_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	g := buildConflictGraph(t, []graph.Node{
		{ID: "example.com/app@", Name: "app", Kind: graph.NodeKindMain},
		{ID: "github.com/pkg@v1.0.0", Name: "github.com/pkg", Version: "v1.0.0", Kind: graph.NodeKindModule},
	})

	a := audit.NewWithEndpoints(audit.Options{SkipLicense: true, SkipConflicts: true}, srv.URL+"/v1/querybatch", "")
	_, err := a.Run(context.Background(), g, "go")
	if err == nil {
		t.Fatal("expected error from server 503, got nil")
	}
}

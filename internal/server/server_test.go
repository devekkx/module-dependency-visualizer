package server_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"module-dependency-visualizer/internal/graph"
	"module-dependency-visualizer/internal/provider"
	"module-dependency-visualizer/internal/schema"
	"module-dependency-visualizer/internal/server"
)

// ── helpers ───────────────────────────────────────────────────

func sampleGraph(t *testing.T) []byte {
	t.Helper()
	b := graph.NewBuilder()
	if err := b.AddNode(graph.Node{ID: "root@1.0.0", Name: "root", Version: "1.0.0", Kind: graph.NodeKindMain}); err != nil {
		t.Fatal(err)
	}
	if err := b.AddNode(graph.Node{ID: "dep1@0.1.0", Name: "dep-one", Version: "0.1.0", Kind: graph.NodeKindModule}); err != nil {
		t.Fatal(err)
	}
	if err := b.AddEdge(graph.Edge{From: "root@1.0.0", To: "dep1@0.1.0", Kind: graph.EdgeKindDependsOn}); err != nil {
		t.Fatal(err)
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	data, err := schema.Encode(g, schema.EncodeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func sampleProject() provider.Project {
	return provider.Project{
		Name:       "test-project",
		Language:   "go",
		RootPath:   "/tmp/test",
		MainModule: "root",
	}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := server.NewFromJSON(sampleGraph(t), sampleProject())
	return httptest.NewServer(srv.Handler())
}

// ── static asset tests ────────────────────────────────────────

func TestHandler_ServesIndexHTML(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "<html") {
		t.Errorf("response body does not look like HTML")
	}
}

func TestHandler_ServesStaticCSS(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/static/style.css")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /static/style.css status = %d, want 200", resp.StatusCode)
	}
}

func TestHandler_ServesStaticJS(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/static/app.js")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /static/app.js status = %d, want 200", resp.StatusCode)
	}
}

func TestHandler_404ForUnknownPath(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET /does-not-exist status = %d, want 404", resp.StatusCode)
	}
}

// ── /api/graph tests ──────────────────────────────────────────

func TestHandler_GraphEndpoint_OK(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/graph")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/graph status = %d, want 200", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if _, ok := payload["nodes"]; !ok {
		t.Errorf("response JSON missing 'nodes' key; got %v", payload)
	}
}

func TestHandler_GraphEndpoint_CacheControlNoStore(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/graph")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	cc := resp.Header.Get("Cache-Control")
	if cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestHandler_GraphEndpoint_Unavailable(t *testing.T) {
	// A server with nil graphJSON simulates the unloaded state.
	srv := server.NewFromJSON(nil, provider.Project{})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/graph")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", resp.StatusCode)
	}
}

func TestHandler_GraphEndpoint_MethodNotAllowed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/graph", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Fatal("POST /api/graph should not return 200")
	}
}

// ── /api/info tests ───────────────────────────────────────────

func TestHandler_InfoEndpoint_OK(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/info status = %d, want 200", resp.StatusCode)
	}
	var info struct {
		Name     string `json:"name"`
		Language string `json:"language"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if info.Name != "test-project" {
		t.Errorf("name = %q, want test-project", info.Name)
	}
	if info.Language != "go" {
		t.Errorf("language = %q, want go", info.Language)
	}
}

// ── httptest.Recorder tests ───────────────────────────────────

func TestHandleGraph_Recorder(t *testing.T) {
	srv := server.NewFromJSON(sampleGraph(t), sampleProject())
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200", rr.Code)
	}
}

func TestHandleGraph_Recorder_Unavailable(t *testing.T) {
	srv := server.NewFromJSON(nil, provider.Project{})
	handler := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d, want 503", rr.Code)
	}
}

// ── Start / Shutdown ──────────────────────────────────────────

func TestServer_StartAndShutdown(t *testing.T) {
	srv := server.NewFromJSON(sampleGraph(t), sampleProject())
	ctx := context.Background()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if addr == "" {
		t.Fatal("addr is empty")
	}

	// Verify the server actually listens.
	resp, err := http.Get("http://" + addr + "/api/graph")
	if err != nil {
		t.Fatalf("GET after Start: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status after Start = %d, want 200", resp.StatusCode)
	}

	shutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
}

func TestServer_ShutdownBeforeStart(t *testing.T) {
	srv := server.NewFromJSON(sampleGraph(t), sampleProject())
	// Shutting down a never-started server must be a no-op.
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown before Start: %v", err)
	}
}

// ── loadGraph (New constructor) ───────────────────────────────

// stubProvider satisfies provider.Provider using an in-memory graph.
type stubProvider struct {
	g    *graph.Graph
	proj provider.Project
}

func (s *stubProvider) Name() string { return "stub" }
func (s *stubProvider) Detect(_ context.Context, _ string) (bool, error) {
	return true, nil
}
func (s *stubProvider) Parse(_ context.Context, _ string, _ provider.ParseOptions) (*graph.Graph, provider.Project, error) {
	return s.g, s.proj, nil
}

func newStubGraph(t *testing.T) *graph.Graph {
	t.Helper()
	b := graph.NewBuilder()
	if err := b.AddNode(graph.Node{ID: "root@0.0.1", Name: "root", Version: "0.0.1", Kind: graph.NodeKindMain}); err != nil {
		t.Fatal(err)
	}
	g, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestServer_New_LoadsGraphViaProvider(t *testing.T) {
	stub := &stubProvider{
		g:    newStubGraph(t),
		proj: provider.Project{Name: "stub-proj", Language: "stub"},
	}

	reg := provider.NewRegistry()
	reg.Register(stub)

	srv := server.New(reg, server.Options{Port: 0, Path: t.TempDir()})
	ctx := context.Background()

	addr, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("Start with provider: %v", err)
	}
	defer func() {
		shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	resp, err := http.Get("http://" + addr + "/api/graph")
	if err != nil {
		t.Fatalf("GET /api/graph: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := payload["nodes"]; !ok {
		t.Errorf("response missing 'nodes': %v", payload)
	}
}

func TestServer_New_DetectError(t *testing.T) {
	reg := provider.NewRegistry() // no providers registered → Detect returns error

	srv := server.New(reg, server.Options{Port: 0, Path: t.TempDir()})
	_, err := srv.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when no provider matches, got nil")
	}
}

// ── /api/audit tests ──────────────────────────────────────────

func TestHandler_AuditEndpoint_OK(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/audit")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /api/audit status = %d, want 200; body: %s", resp.StatusCode, body)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	for _, key := range []string{"vulnerabilities", "conflicts", "licenses"} {
		if _, ok := payload[key]; !ok {
			t.Errorf("audit response missing %q key; got %v", key, payload)
		}
	}
}

func TestHandler_AuditEndpoint_Cached(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// First call runs the audit.
	resp1, err := http.Get(ts.URL + "/api/audit")
	if err != nil {
		t.Fatal(err)
	}
	resp1.Body.Close()

	// Second call should return the same cached result.
	resp2, err := http.Get(ts.URL + "/api/audit")
	if err != nil {
		t.Fatal(err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("second GET /api/audit status = %d, want 200", resp2.StatusCode)
	}
}

func TestHandler_AuditEndpoint_Unavailable(t *testing.T) {
	srv := server.NewFromJSON(nil, provider.Project{})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/audit")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", resp.StatusCode)
	}
}


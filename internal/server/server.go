package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"module-dependency-visualizer/internal/provider"
	"module-dependency-visualizer/internal/schema"
)

//go:embed all:web
var webFiles embed.FS

// Options configures the Server.
type Options struct {
	// Port to listen on. 0 lets the OS assign a free port.
	Port int
	// Path is the project root directory to analyse.
	Path string
}

// Server serves the D3.js dependency visualisation UI over HTTP.
type Server struct {
	providers *provider.Registry
	opts      Options

	mu        sync.RWMutex
	graphJSON []byte
	project   provider.Project

	httpSrv *http.Server
}

// New creates a Server backed by the given provider registry.
// Call Start to begin listening.
func New(providers *provider.Registry, opts Options) *Server {
	return &Server{providers: providers, opts: opts}
}

// NewFromJSON creates a Server preloaded with graph JSON.
// Useful for testing without running provider parsing. Calling Start on such
// a server skips provider detection and uses the supplied data directly.
func NewFromJSON(graphJSON []byte, proj provider.Project) *Server {
	return &Server{graphJSON: graphJSON, project: proj}
}

// Handler returns the HTTP handler that serves the visualisation UI.
// Exposed for use with httptest.NewServer in tests.
func (s *Server) Handler() http.Handler {
	webContent, err := fs.Sub(webFiles, "web")
	if err != nil {
		panic("server: sub web FS: " + err.Error())
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/graph", s.handleGraph)
	mux.HandleFunc("GET /api/info", s.handleInfo)

	// Static assets under /static/
	mux.Handle("GET /static/", http.FileServer(http.FS(webContent)))

	// SPA index - serve index.html for the root only.
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, webContent, "index.html")
	})

	return mux
}

// Start parses the dependency graph (if not already loaded) and begins
// serving HTTP. It returns the network address being listened on
// (e.g. "127.0.0.1:7070"), or an error.
func (s *Server) Start(ctx context.Context) (string, error) {
	s.mu.RLock()
	alreadyLoaded := len(s.graphJSON) > 0
	s.mu.RUnlock()

	if !alreadyLoaded {
		if err := s.loadGraph(ctx); err != nil {
			return "", err
		}
	}

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.opts.Port))
	if err != nil {
		return "", fmt.Errorf("server: listen: %w", err)
	}

	httpSrv := &http.Server{
		Handler:      s.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	s.mu.Lock()
	s.httpSrv = httpSrv
	s.mu.Unlock()

	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			_ = err // expected on Shutdown
		}
	}()

	return ln.Addr().String(), nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.RLock()
	httpSrv := s.httpSrv
	s.mu.RUnlock()
	if httpSrv == nil {
		return nil
	}
	return httpSrv.Shutdown(ctx)
}

// loadGraph runs provider detection and parsing, caching the result.
func (s *Server) loadGraph(ctx context.Context) error {
	prov, err := s.providers.Detect(ctx, s.opts.Path)
	if err != nil {
		return fmt.Errorf("server: detect provider for %q: %w", s.opts.Path, err)
	}

	g, proj, err := prov.Parse(ctx, s.opts.Path, provider.ParseOptions{})
	if err != nil {
		return fmt.Errorf("server: parse graph: %w", err)
	}

	jsonData, err := schema.Encode(g, schema.EncodeOptions{})
	if err != nil {
		return fmt.Errorf("server: encode graph: %w", err)
	}

	s.mu.Lock()
	s.graphJSON = jsonData
	s.project = proj
	s.mu.Unlock()

	return nil
}

// handleGraph serves the cached graph JSON.
func (s *Server) handleGraph(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	data := s.graphJSON
	s.mu.RUnlock()

	if data == nil {
		http.Error(w, `{"error":"graph not yet loaded"}`, http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

// handleInfo serves lightweight project metadata as JSON.
func (s *Server) handleInfo(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	proj := s.project
	s.mu.RUnlock()

	info := struct {
		Name       string `json:"name"`
		Language   string `json:"language"`
		RootPath   string `json:"root_path"`
		MainModule string `json:"main_module"`
	}{
		Name:       proj.Name,
		Language:   proj.Language,
		RootPath:   proj.RootPath,
		MainModule: proj.MainModule,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(info)
}

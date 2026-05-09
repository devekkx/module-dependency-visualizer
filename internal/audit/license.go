package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

const depsDevBase = "https://api.deps.dev/v3alpha"

// SystemFor maps a language name to the deps.dev system identifier.
// Returns an empty string for unknown languages.
func SystemFor(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "go"
	case "node", "javascript", "typescript":
		return "npm"
	case "python":
		return "pypi"
	default:
		return ""
	}
}

type depsDevVersionResp struct {
	Version struct {
		Licenses []string `json:"licenses"`
	} `json:"version"`
}

type licenseScanner struct {
	client  HTTPClient
	baseURL string
}

func newLicenseScanner(client HTTPClient) *licenseScanner {
	return &licenseScanner{client: client, baseURL: depsDevBase}
}

// FetchOne returns the SPDX license expression for a single package version.
// Returns "Unknown" on any error or when no license data is available.
func (s *licenseScanner) FetchOne(ctx context.Context, system, pkg, version string) string {
	if system == "" || pkg == "" || version == "" {
		return "Unknown"
	}

	rawURL := fmt.Sprintf("%s/systems/%s/packages/%s/versions/%s",
		s.baseURL,
		url.PathEscape(system),
		url.PathEscape(pkg),
		url.PathEscape(version),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "Unknown"
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "Unknown"
	}

	var data depsDevVersionResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "Unknown"
	}

	if len(data.Version.Licenses) == 0 {
		return "Unknown"
	}

	return strings.Join(data.Version.Licenses, " AND ")
}

// FetchAll returns a map of nodeID → SPDX license string for all versioned
// non-main nodes in g. Requests are made sequentially.
func (s *licenseScanner) FetchAll(ctx context.Context, g *graph.Graph, system string) map[string]string {
	result := make(map[string]string)
	for _, n := range g.Nodes() {
		if n.Kind == graph.NodeKindMain || n.Version == "" {
			continue
		}
		result[string(n.ID)] = s.FetchOne(ctx, system, n.Name, n.Version)
	}
	return result
}

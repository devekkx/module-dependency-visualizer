package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

const osvBatchEndpoint = "https://api.osv.dev/v1/querybatch"

// HTTPClient abstracts http.Client so tests can inject a stub.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// EcosystemFor maps a language name to the OSV ecosystem identifier.
// Returns an empty string for unknown languages.
func EcosystemFor(language string) string {
	switch strings.ToLower(language) {
	case "go":
		return "Go"
	case "node", "javascript", "typescript":
		return "npm"
	case "python":
		return "PyPI"
	default:
		return ""
	}
}

type osvQuery struct {
	Version string `json:"version"`
	Package osvPkg `json:"package"`
}

type osvPkg struct {
	Name      string `json:"name"`
	Ecosystem string `json:"ecosystem"`
}

type osvBatchReq struct {
	Queries []osvQuery `json:"queries"`
}

type osvBatchResp struct {
	Results []osvQueryResult `json:"results"`
}

type osvQueryResult struct {
	Vulns []osvVuln `json:"vulns"`
}

type osvVuln struct {
	ID               string                 `json:"id"`
	Summary          string                 `json:"summary"`
	Affected         []osvAffected          `json:"affected"`
	DatabaseSpecific map[string]any `json:"database_specific"`
}

type osvAffected struct {
	Ranges []osvRange `json:"ranges"`
}

type osvRange struct {
	Events []osvEvent `json:"events"`
}

type osvEvent struct {
	Fixed string `json:"fixed"`
}

type osvScanner struct {
	client   HTTPClient
	endpoint string
}

func newOSVScanner(client HTTPClient) *osvScanner {
	return &osvScanner{client: client, endpoint: osvBatchEndpoint}
}

// Scan queries the OSV batch endpoint for known vulnerabilities in the graph.
// Only modules with a non-empty version are queried. Returns nil, nil when
// ecosystem is empty (language not supported).
func (s *osvScanner) Scan(ctx context.Context, g *graph.Graph, ecosystem string) ([]Vulnerability, error) {
	if ecosystem == "" {
		return nil, nil
	}

	type entry struct {
		nodeID  string
		name    string
		version string
	}

	var entries []entry
	for _, n := range g.Nodes() {
		if n.Kind == graph.NodeKindMain || n.Version == "" || n.Name == "" {
			continue
		}
		if !isExactVersion(n.Version) {
			continue
		}
		entries = append(entries, entry{
			nodeID:  string(n.ID),
			name:    n.Name,
			version: n.Version,
		})
	}

	if len(entries) == 0 {
		return nil, nil
	}

	queries := make([]osvQuery, len(entries))
	for i, e := range entries {
		queries[i] = osvQuery{
			Version: e.version,
			Package: osvPkg{Name: e.name, Ecosystem: ecosystem},
		}
	}

	body, err := json.Marshal(osvBatchReq{Queries: queries})
	if err != nil {
		return nil, fmt.Errorf("audit/osv: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("audit/osv: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("audit/osv: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("audit/osv: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var batchResp osvBatchResp
	if err := json.NewDecoder(resp.Body).Decode(&batchResp); err != nil {
		return nil, fmt.Errorf("audit/osv: decode response: %w", err)
	}

	var vulns []Vulnerability
	for i, result := range batchResp.Results {
		if i >= len(entries) {
			break
		}
		for _, v := range result.Vulns {
			vuln := Vulnerability{
				NodeID:   entries[i].nodeID,
				ID:       v.ID,
				Summary:  v.Summary,
				Severity: osvSeverity(v),
				FixedIn:  osvFixedVersion(v),
				Link:     "https://osv.dev/vulnerability/" + v.ID,
			}
			vulns = append(vulns, vuln)
		}
	}

	return vulns, nil
}

// osvSeverity extracts a human-readable severity from database_specific.severity.
func osvSeverity(v osvVuln) string {
	if v.DatabaseSpecific != nil {
		if s, ok := v.DatabaseSpecific["severity"].(string); ok && s != "" {
			return strings.ToUpper(s)
		}
	}
	return "UNKNOWN"
}

// isExactVersion reports whether v looks like an exact, queryable version.
// It rejects semver range specifiers, workspace/file/git protocols, and
// anything else OSV's batch endpoint would reject with a 400.
func isExactVersion(v string) bool {
	if v == "" {
		return false
	}
	// Range operators and wildcard
	if strings.ContainsAny(v[:1], "^~><*=") {
		return false
	}
	// Protocol-style versions: workspace:*, file:../, git+https:, etc.
	if strings.Contains(v, ":") {
		return false
	}
	// npm "latest", "next", unresolved tags
	if strings.ContainsAny(v, " |&") {
		return false
	}
	return true
}

// osvFixedVersion returns the earliest "fixed" version from affected ranges.
func osvFixedVersion(v osvVuln) string {
	for _, aff := range v.Affected {
		for _, r := range aff.Ranges {
			for _, ev := range r.Events {
				if ev.Fixed != "" {
					return ev.Fixed
				}
			}
		}
	}
	return ""
}

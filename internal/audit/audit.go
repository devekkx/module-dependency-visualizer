package audit

import (
	"context"
	"net/http"
	"time"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// Options configures which checks are run and how.
type Options struct {
	// SkipVulnScan disables OSV vulnerability scanning.
	SkipVulnScan bool
	// SkipLicense disables license fetching via deps.dev.
	SkipLicense bool
	// SkipConflicts disables version-conflict detection.
	SkipConflicts bool
	// HTTPClient is used for all outbound requests.
	// If nil, a default client with a 30-second timeout is used.
	HTTPClient HTTPClient
}

// Auditor runs the intelligence checks on a dependency graph.
type Auditor struct {
	opts    Options
	osv     *osvScanner
	license *licenseScanner
}

// New creates an Auditor with the given options and the real OSV / deps.dev
// endpoints.
func New(opts Options) *Auditor {
	return newAuditor(opts, osvBatchEndpoint, depsDevBase)
}

// NewWithEndpoints creates an Auditor that points to the given endpoint URLs.
// Pass an empty string for either endpoint to use the respective default.
// Intended for tests that inject httptest servers.
func NewWithEndpoints(opts Options, osvEndpoint, depsDevEndpoint string) *Auditor {
	if osvEndpoint == "" {
		osvEndpoint = osvBatchEndpoint
	}
	if depsDevEndpoint == "" {
		depsDevEndpoint = depsDevBase
	}
	return newAuditor(opts, osvEndpoint, depsDevEndpoint)
}

func newAuditor(opts Options, osvEndpoint, depsDevEndpoint string) *Auditor {
	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}
	s := newOSVScanner(hc)
	s.endpoint = osvEndpoint

	l := newLicenseScanner(hc)
	l.baseURL = depsDevEndpoint

	return &Auditor{opts: opts, osv: s, license: l}
}

// Run executes all enabled checks and returns the combined Result.
func (a *Auditor) Run(ctx context.Context, g *graph.Graph, language string) (*Result, error) {
	r := &Result{
		ScannedAt: time.Now().UTC(),
		Licenses:  make(map[string]string),
	}

	if !a.opts.SkipConflicts {
		r.Conflicts = DetectConflicts(g)
	}

	if !a.opts.SkipVulnScan {
		vulns, err := a.osv.Scan(ctx, g, EcosystemFor(language))
		if err != nil {
			return nil, err
		}
		r.Vulnerabilities = vulns
	}

	if !a.opts.SkipLicense {
		system := SystemFor(language)
		if system != "" {
			r.Licenses = a.license.FetchAll(ctx, g, system)
		}
	}

	return r, nil
}

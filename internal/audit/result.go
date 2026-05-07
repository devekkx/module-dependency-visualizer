package audit

import "time"

// Result is the complete output of an audit run.
type Result struct {
	ScannedAt       time.Time
	Vulnerabilities []Vulnerability
	Conflicts       []Conflict
	Licenses        map[string]string // nodeID → SPDX expression or "Unknown"
}

// Vulnerability describes a known security issue affecting a dependency.
type Vulnerability struct {
	NodeID   string
	ID       string // OSV / GHSA / CVE identifier
	Summary  string
	Severity string // CRITICAL, HIGH, MEDIUM, LOW, or UNKNOWN
	FixedIn  string // first version that resolves the issue, if known
	Link     string // canonical advisory URL
}

// Conflict describes a module that appears with more than one resolved version.
type Conflict struct {
	Module   string   // import path without version
	Versions []string // all observed versions, sorted
}

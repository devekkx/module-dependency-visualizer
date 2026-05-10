package schema

import "time"

const Version = "1.1.0"

// Document is the top-level JSON envelope for a serialized dependency graph.
type Document struct {
	SchemaVersion string    `json:"schema_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	Project       Project   `json:"project"`
	Nodes         []NodeDTO `json:"nodes"`
	Edges         []EdgeDTO `json:"edges"`
	Stats         StatsDTO  `json:"stats"`
	Audit         *AuditDTO `json:"audit,omitempty"`
}

// Project describes the analyzed project.
type Project struct {
	Name       string `json:"name"`
	Language   string `json:"language"`
	RootPath   string `json:"root_path"`
	MainModule string `json:"main_module"`
}

// NodeDTO is the JSON representation of a graph node.
type NodeDTO struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Version    string         `json:"version"`
	Kind       string         `json:"kind"`
	Indirect   bool           `json:"indirect"`
	Dev        bool           `json:"dev,omitempty"`
	ReplacedBy *ReplacementDTO `json:"replaced_by,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ReplacementDTO describes a module replace directive.
type ReplacementDTO struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

// EdgeDTO is the JSON representation of a graph edge.
type EdgeDTO struct {
	From     string         `json:"from"`
	To       string         `json:"to"`
	Kind     string         `json:"kind"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StatsDTO holds aggregate metrics.
type StatsDTO struct {
	NodeCount int  `json:"node_count"`
	EdgeCount int  `json:"edge_count"`
	MaxDepth  int  `json:"max_depth"`
	HasCycles bool `json:"has_cycles"`
}

// AuditDTO embeds the results of a Phase 4 intelligence scan.
type AuditDTO struct {
	ScannedAt       time.Time         `json:"scanned_at"`
	Vulnerabilities []VulnDTO         `json:"vulnerabilities"`
	Conflicts       []ConflictDTO     `json:"conflicts"`
	Licenses        map[string]string `json:"licenses"`
}

// VulnDTO describes a single known vulnerability affecting a dependency.
type VulnDTO struct {
	NodeID   string `json:"node_id"`
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Severity string `json:"severity"`
	FixedIn  string `json:"fixed_in,omitempty"`
	Link     string `json:"link,omitempty"`
}

// ConflictDTO describes a module that appears with multiple resolved versions.
type ConflictDTO struct {
	Module   string   `json:"module"`
	Versions []string `json:"versions"`
}

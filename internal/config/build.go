package config

import "fmt"

// BuildInfo holds version and build metadata injected at link time via -ldflags.
type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

// String returns a human-readable single-line representation.
func (b BuildInfo) String() string {
	return fmt.Sprintf("version=%s commit=%s built=%s", b.Version, b.Commit, b.BuildDate)
}

// Default is the BuildInfo used when no ldflags are provided (development builds).
var Default = BuildInfo{
	Version:   "dev",
	Commit:    "none",
	BuildDate: "unknown",
}

package config

import "time"

const (
	SchemaVersion        = "1.0.0"
	DefaultTimeout       = 30 * time.Second
	DefaultAuditTimeout  = 2 * time.Minute
	DefaultDepth         = -1
	DefaultPort    = 0

	NodeKindMain    = "main"
	NodeKindModule  = "module"
	NodeKindReplace = "replace"

	EdgeKindDependsOn = "depends_on"
	EdgeKindReplaces  = "replaces"

	ExitSuccess  = 0
	ExitUser     = 1
	ExitProvider = 2
	ExitInternal = 3
)

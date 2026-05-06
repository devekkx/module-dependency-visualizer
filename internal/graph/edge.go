package graph

// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
	EdgeKindDependsOn EdgeKind = "depends_on"
	EdgeKindReplaces  EdgeKind = "replaces"
)

// Edge is an immutable directed relationship between two nodes.
type Edge struct {
	From     NodeID
	To       NodeID
	Kind     EdgeKind
	Metadata map[string]any
}

// Copy returns a copy that callers cannot mutate.
func (e Edge) Copy() Edge {
	e.Metadata = metadataCopy(e.Metadata)
	return e
}

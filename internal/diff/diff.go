package diff

import (
	"sort"
	"strings"

	"github.com/devekkx/module-dependency-visualizer/internal/graph"
)

// ChangeKind classifies how a dependency changed between two graphs.
type ChangeKind string

const (
	ChangeAdded   ChangeKind = "added"
	ChangeRemoved ChangeKind = "removed"
	ChangeUpdated ChangeKind = "updated"
)

// Change describes a single dependency change.
type Change struct {
	Kind        ChangeKind
	Name        string
	FromVersion string // empty for added
	ToVersion   string // empty for removed
	Indirect    bool   // indirect status in the target (or source for removals)
}

// Result is the complete comparison between two dependency graphs.
type Result struct {
	Added     []Change
	Removed   []Change
	Updated   []Change
	Unchanged int
}

// Diff computes the dependency differences between two graphs.
// The main module is excluded from comparison.
func Diff(from, to *graph.Graph) *Result {
	fromMap := indexByName(from)
	toMap := indexByName(to)

	r := &Result{}

	for name, fn := range fromMap {
		if tn, ok := toMap[name]; ok {
			if fn.Version != tn.Version {
				r.Updated = append(r.Updated, Change{
					Kind:        ChangeUpdated,
					Name:        name,
					FromVersion: fn.Version,
					ToVersion:   tn.Version,
					Indirect:    tn.Indirect,
				})
			} else {
				r.Unchanged++
			}
		} else {
			r.Removed = append(r.Removed, Change{
				Kind:        ChangeRemoved,
				Name:        name,
				FromVersion: fn.Version,
				Indirect:    fn.Indirect,
			})
		}
	}

	for name, tn := range toMap {
		if _, ok := fromMap[name]; !ok {
			r.Added = append(r.Added, Change{
				Kind:      ChangeAdded,
				Name:      name,
				ToVersion: tn.Version,
				Indirect:  tn.Indirect,
			})
		}
	}

	sortChanges(r.Added)
	sortChanges(r.Removed)
	sortChanges(r.Updated)

	return r
}

// indexByName maps module name → Node for all non-main nodes.
// When the same name appears at multiple versions (a conflict), the last one
// wins - the diff is still meaningful for detecting presence/absence.
func indexByName(g *graph.Graph) map[string]graph.Node {
	m := make(map[string]graph.Node)
	for _, n := range g.Nodes() {
		if n.Kind == graph.NodeKindMain {
			continue
		}
		name := n.Name
		if name == "" {
			// Fall back to extracting the path from the ID.
			id := string(n.ID)
			if at := strings.LastIndex(id, "@"); at >= 0 {
				name = id[:at]
			}
		}
		m[name] = n
	}
	return m
}

func sortChanges(cs []Change) {
	sort.Slice(cs, func(i, j int) bool { return cs[i].Name < cs[j].Name })
}

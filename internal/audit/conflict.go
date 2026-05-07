package audit

import (
	"sort"
	"strings"

	"module-dependency-visualizer/internal/graph"
)

// DetectConflicts returns all modules that appear with more than one resolved
// version in the graph. The main module is excluded from the analysis.
func DetectConflicts(g *graph.Graph) []Conflict {
	versionSets := make(map[string]map[string]struct{})

	for _, n := range g.Nodes() {
		if n.Kind == graph.NodeKindMain {
			continue
		}
		id := string(n.ID)
		at := strings.LastIndex(id, "@")
		if at < 0 {
			continue
		}
		path, ver := id[:at], id[at+1:]
		if ver == "" {
			continue
		}
		if versionSets[path] == nil {
			versionSets[path] = make(map[string]struct{})
		}
		versionSets[path][ver] = struct{}{}
	}

	var conflicts []Conflict
	for path, vset := range versionSets {
		if len(vset) < 2 {
			continue
		}
		vs := make([]string, 0, len(vset))
		for v := range vset {
			vs = append(vs, v)
		}
		sort.Strings(vs)
		conflicts = append(conflicts, Conflict{Module: path, Versions: vs})
	}
	sort.Slice(conflicts, func(i, j int) bool {
		return conflicts[i].Module < conflicts[j].Module
	})
	return conflicts
}

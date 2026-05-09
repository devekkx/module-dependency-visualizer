package node_test

import (
	"testing"

	nodeprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/node"
)

func TestParsePackageJSON_Simple(t *testing.T) {
	data := fixtureFile(t, "npm-simple", "package.json")
	pkg, err := nodeprovider.ParsePackageJSON(data)
	if err != nil {
		t.Fatalf("ParsePackageJSON: %v", err)
	}
	if pkg.Name != "my-app" {
		t.Errorf("Name = %q; want %q", pkg.Name, "my-app")
	}
	if pkg.Version != "1.0.0" {
		t.Errorf("Version = %q; want 1.0.0", pkg.Version)
	}
	if len(pkg.Dependencies) != 1 {
		t.Errorf("Dependencies len = %d; want 1", len(pkg.Dependencies))
	}
	if len(pkg.DevDependencies) != 1 {
		t.Errorf("DevDependencies len = %d; want 1", len(pkg.DevDependencies))
	}
}

func TestParsePackageJSON_InvalidJSON(t *testing.T) {
	_, err := nodeprovider.ParsePackageJSON([]byte("{bad json}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseNpmLS_BuildsTree(t *testing.T) {
	data := []byte(`{
		"name": "my-app", "version": "1.0.0",
		"dependencies": {
			"chalk": {
				"version": "5.3.0",
				"dependencies": {
					"supports-color": {"version": "7.2.0"}
				}
			}
		}
	}`)
	root, err := nodeprovider.ParseNpmLS(data)
	if err != nil {
		t.Fatalf("ParseNpmLS: %v", err)
	}
	if root.Name != "my-app" {
		t.Errorf("root.Name = %q; want my-app", root.Name)
	}
	if len(root.Children) != 1 {
		t.Errorf("root.Children len = %d; want 1", len(root.Children))
	}
	if len(root.Children[0].Children) != 1 {
		t.Errorf("chalk.Children len = %d; want 1", len(root.Children[0].Children))
	}
}

func TestParseNpmLS_InvalidJSON(t *testing.T) {
	_, err := nodeprovider.ParseNpmLS([]byte("{not json}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParsePnpmLS_BuildsTree(t *testing.T) {
	data := fixtureFile(t, "pnpm-simple", "pnpm-ls.json")
	root, err := nodeprovider.ParsePnpmLS(data)
	if err != nil {
		t.Fatalf("ParsePnpmLS: %v", err)
	}
	if root.Name != "pnpm-app" {
		t.Errorf("root.Name = %q; want pnpm-app", root.Name)
	}
	if len(root.Children) == 0 {
		t.Error("expected at least one child dep")
	}
}

func TestParsePnpmLS_EmptyOutput(t *testing.T) {
	_, err := nodeprovider.ParsePnpmLS([]byte("[]"))
	if err == nil {
		t.Error("expected error for empty pnpm ls output")
	}
}

func TestParsePnpmLS_InvalidJSON(t *testing.T) {
	_, err := nodeprovider.ParsePnpmLS([]byte("{bad}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseYarnList_BuildsTree(t *testing.T) {
	pkg := nodeprovider.PackageJSON{
		Name:    "yarn-app",
		Version: "1.0.0",
		DevDependencies: map[string]string{"typescript": "^5.0.0"},
	}
	data := fixtureFile(t, "yarn-simple", "yarn-list.json")
	root, err := nodeprovider.ParseYarnList(pkg, data)
	if err != nil {
		t.Fatalf("ParseYarnList: %v", err)
	}
	if root.Name != "yarn-app" {
		t.Errorf("root.Name = %q; want yarn-app", root.Name)
	}
	if len(root.Children) != 2 { // chalk + typescript
		t.Errorf("root.Children len = %d; want 2", len(root.Children))
	}
	for _, c := range root.Children {
		if c.Name == "typescript" && !c.Dev {
			t.Error("typescript should be marked as Dev=true")
		}
	}
}

func TestParseYarnList_InvalidJSON(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	_, err := nodeprovider.ParseYarnList(pkg, []byte("{bad}"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseYarnInfo_BuildsGraph(t *testing.T) {
	pkg := nodeprovider.PackageJSON{
		Name:         "berry-app",
		Version:      "1.0.0",
		Dependencies: map[string]string{"chalk": "^5.3.0"},
	}
	ndjson := `{"value":"chalk@npm:5.3.0","children":{"Version":"5.3.0","Dependencies":[]}}` + "\n"
	g, main, err := nodeprovider.ParseYarnInfo(pkg, []byte(ndjson))
	if err != nil {
		t.Fatalf("ParseYarnInfo: %v", err)
	}
	if main != "berry-app" {
		t.Errorf("main = %q; want berry-app", main)
	}
	if g.NodeCount() < 2 {
		t.Errorf("NodeCount = %d; want ≥ 2 (root + chalk)", g.NodeCount())
	}
}

func TestParseYarnInfo_SkipsMalformedLines(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	// Mix of valid and invalid lines.
	ndjson := "not json\n" + `{"value":"chalk@npm:5.3.0","children":{"Version":"5.3.0","Dependencies":[]}}` + "\n"
	_, _, err := nodeprovider.ParseYarnInfo(pkg, []byte(ndjson))
	if err != nil {
		t.Fatalf("ParseYarnInfo should skip bad lines, got: %v", err)
	}
}

func TestParseBunLS_BuildsGraph(t *testing.T) {
	pkg := nodeprovider.PackageJSON{
		Name:            "bun-app",
		Version:         "1.0.0",
		Dependencies:    map[string]string{"chalk": "^5.3.0"},
		DevDependencies: map[string]string{"typescript": "^5.0.0"},
	}
	data := fixtureFile(t, "bun-simple", "bun-pm-ls.txt")
	g, main, err := nodeprovider.ParseBunLS(pkg, data)
	if err != nil {
		t.Fatalf("ParseBunLS: %v", err)
	}
	if main != "bun-app" {
		t.Errorf("main = %q; want bun-app", main)
	}
	if g.NodeCount() < 3 { // root + chalk + typescript (+ supports-color)
		t.Errorf("NodeCount = %d; want ≥ 3", g.NodeCount())
	}
}

func TestParseBunLS_EmptyOutput(t *testing.T) {
	pkg := nodeprovider.PackageJSON{Name: "app", Version: "1.0.0"}
	g, main, err := nodeprovider.ParseBunLS(pkg, []byte("bun pm v1.0.0\n"))
	if err != nil {
		t.Fatalf("ParseBunLS empty: %v", err)
	}
	if main != "app" {
		t.Errorf("main = %q; want app", main)
	}
	if g.NodeCount() != 1 {
		t.Errorf("NodeCount = %d; want 1 (root only)", g.NodeCount())
	}
}

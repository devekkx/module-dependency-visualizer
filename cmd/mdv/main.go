package main

import (
	"os"

	"github.com/devekkx/module-dependency-visualizer/internal/cli"
	"github.com/devekkx/module-dependency-visualizer/internal/config"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/dot"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/jsonexp"
	"github.com/devekkx/module-dependency-visualizer/internal/exporter/mermaid"
	"github.com/devekkx/module-dependency-visualizer/internal/logging"
	"github.com/devekkx/module-dependency-visualizer/internal/provider"
	goprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/golang"
	nodeprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/node"
	pyprovider "github.com/devekkx/module-dependency-visualizer/internal/provider/python"
	"github.com/devekkx/module-dependency-visualizer/internal/schema"
)

// These variables are set by goreleaser / ldflags at build time.
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	provReg := provider.NewRegistry()
	if err := provReg.Register(goprovider.New()); err != nil {
		panic(err)
	}
	if err := provReg.Register(nodeprovider.New()); err != nil {
		panic(err)
	}
	if err := provReg.Register(pyprovider.New()); err != nil {
		panic(err)
	}

	expReg := exporter.NewRegistry()
	if err := expReg.Register(jsonexp.New(schema.EncodeOptions{})); err != nil {
		panic(err)
	}
	if err := expReg.Register(dot.New()); err != nil {
		panic(err)
	}
	if err := expReg.Register(mermaid.New()); err != nil {
		panic(err)
	}

	deps := &cli.Deps{
		Log: logging.Discard(),
		BuildInfo: config.BuildInfo{
			Version:   version,
			Commit:    commit,
			BuildDate: buildDate,
		},
		Providers: provReg,
		Exporters: expReg,
	}

	os.Exit(cli.ExecuteWithDeps(deps, nil))
}

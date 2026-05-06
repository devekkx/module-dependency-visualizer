package main

import (
	"os"

	"module-dependency-visualizer/internal/cli"
	"module-dependency-visualizer/internal/config"
	"module-dependency-visualizer/internal/exporter"
	"module-dependency-visualizer/internal/exporter/dot"
	"module-dependency-visualizer/internal/exporter/jsonexp"
	"module-dependency-visualizer/internal/exporter/mermaid"
	"module-dependency-visualizer/internal/logging"
	"module-dependency-visualizer/internal/provider"
	goprovider "module-dependency-visualizer/internal/provider/golang"
	"module-dependency-visualizer/internal/schema"
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

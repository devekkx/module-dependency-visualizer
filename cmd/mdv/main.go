package main

import (
	"os"

	"module-dependency-visualizer/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}

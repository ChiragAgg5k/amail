// Command amail reads, searches and writes mail through the Apple Mail app.
package main

import (
	"os"
	"runtime/debug"

	"github.com/ChiragAgg5k/amail/internal/cli"
)

const development = "dev"

// version is set at build time for released binaries.
var version = development

func main() {
	os.Exit(cli.Execute(release()))
}

// release falls back to the module version, which `go install` records.
func release() string {
	if version != development {
		return version
	}
	if build, found := debug.ReadBuildInfo(); found && build.Main.Version != "" && build.Main.Version != "(devel)" {
		return build.Main.Version
	}
	return version
}

package main

import "github.com/redwoodjs/rw-cli/cli/cmd"

// NOTE: These variables are set at compile time via ldflags by goreleaser
var (
	version = "unknown"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	// Forward the version information to the cmd package
	cmd.BuildVersion = version
	cmd.BuildCommit = commit
	cmd.BuildDate = date

	cmd.Execute()
}

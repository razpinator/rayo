package main

import "rayo/internal/cli"

// Version information; overridable at build time with -ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.Execute("rayo", version, commit, date)
}

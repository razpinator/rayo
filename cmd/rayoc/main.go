package main

import "rayo/internal/cli"

// rayoc is an alias binary for the Rayo CLI (historically the "compiler"
// entry point). It shares the exact command set with `rayo`.
//
// Version information; overridable at build time with -ldflags.
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cli.Execute("rayoc", version, commit, date)
}

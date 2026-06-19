// Command bifrost is the Go implementation of the Bifrost cross-implementation
// Shen meta test harness AND the Roswell-style front door for Shen across all
// ports. The differential test matrix is the default invocation; front-door
// verbs (run/eval/repl/impls/use/install/build) are added as subcommands.
//
// It reuses the same adapters.json / builders.json data contracts as the
// original Python tool; the embedded copies make the binary self-contained for
// `go install` and release downloads, overridable via $BIFROST_ADAPTERS or a
// local ./adapters.json.
package main

import (
	"flag"
	"fmt"
	"os"
)

var subcommands = map[string]bool{
	"run": true, "eval": true, "repl": true, "impls": true,
	"use": true, "install": true, "build": true,
}

// newFlagSet returns a FlagSet that reports parse errors instead of exiting, so
// each verb handler controls its own exit code.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) > 0 && subcommands[args[0]] {
		a, err := loadAdapters()
		if err != nil {
			fmt.Fprintln(os.Stderr, "bifrost:", err)
			return 2
		}
		verb, rest := args[0], args[1:]
		switch verb {
		case "run", "eval", "repl":
			return cmdLaunch(verb, rest, a)
		case "impls":
			return cmdImpls(rest, a)
		case "use":
			return cmdUse(rest, a)
		case "install":
			return cmdInstall(rest, a)
		case "build":
			return cmdBuild(rest, a)
		}
	}
	// Default: the differential test matrix.
	return runMatrix(args)
}

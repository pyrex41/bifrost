// Command bifrost is the Go implementation of the Bifrost cross-implementation
// Shen meta test harness AND the Roswell-style front door for Shen across all
// ports. The differential test matrix is the default invocation; front-door
// verbs (run/eval/repl/impls/use/install/build) are added as subcommands.
//
// It embeds the adapters.json registry and case corpus, making the binary self-contained for
// `go install` and release downloads, overridable via $BIFROST_ADAPTERS or a
// local ./adapters.json.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var subcommands = map[string]bool{
	"run": true, "eval": true, "repl": true, "impls": true,
	"use": true, "install": true, "build": true, "bench": true,
}

// newFlagSet returns a FlagSet that reports parse errors instead of exiting, so
// each verb handler controls its own exit code.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// reorderArgs moves flags (and the values of the named value-flags) ahead of
// positional args, so flags may appear after positionals — Go's flag package
// otherwise stops parsing at the first non-flag token. valueFlags lists the
// flag names (without leading dashes) that consume the following token.
func reorderArgs(args []string, valueFlags ...string) []string {
	vf := map[string]bool{}
	for _, f := range valueFlags {
		vf[f] = true
	}
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" { // pass-through separator: rest are positional
			pos = append(pos, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			flags = append(flags, a)
			name := strings.TrimLeft(a, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				name = name[:eq]
			} else if vf[name] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			pos = append(pos, a)
		}
	}
	return append(flags, pos...)
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
		case "bench":
			return cmdBench(rest)
		}
	}
	// Default: the differential test matrix.
	return runMatrix(args)
}

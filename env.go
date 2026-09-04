package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// cmdEnv runs a command inside the composition of one or more port-owned Nix
// toolchains. Bifrost selects ports; each port remains responsible for pinning
// and exporting its own #toolchain package.
func cmdEnv(args []string, a *Adapters) int {
	root := os.Getenv("BIFROST_PORTS_ROOT")
	var ports, command []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--":
			command = append(command, args[i+1:]...)
			i = len(args)
		case args[i] == "--root" && i+1 < len(args):
			i++
			root = args[i]
		case strings.HasPrefix(args[i], "--root="):
			root = strings.TrimPrefix(args[i], "--root=")
		case strings.HasPrefix(args[i], "-"):
			fmt.Fprintf(os.Stderr, "bifrost env: unknown option %q\n", args[i])
			return 2
		default:
			ports = append(ports, args[i])
		}
	}
	if len(ports) == 0 {
		fmt.Fprintln(os.Stderr, "usage: bifrost env [--root DIR] PORT [PORT ...|all] -- COMMAND [ARG ...]")
		return 2
	}
	if root == "" {
		root = filepath.Dir(a.baseDir)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost env:", err)
		return 2
	}
	selected, err := expandPortSelection(ports, a.names())
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost env:", err)
		return 2
	}
	installables := make([]string, 0, len(selected))
	for _, port := range selected {
		portDir := filepath.Join(absRoot, port)
		if _, err := os.Stat(filepath.Join(portDir, "flake.nix")); err != nil {
			fmt.Fprintf(os.Stderr, "bifrost env: no port flake at %s\n", portDir)
			return 2
		}
		installables = append(installables, "path:"+portDir+"#toolchain")
	}
	if len(command) == 0 {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "sh"
		}
		command = []string{shell}
	}
	// The flake's env app invokes this binary by store path, which is not
	// necessarily present on PATH inside the composed shell. Make the common
	// recursive spelling (`... -- bifrost ...`) resolve to this exact build.
	if command[0] == "bifrost" {
		if self, err := os.Executable(); err == nil {
			command[0] = self
		}
	}
	if _, err := exec.LookPath("nix"); err != nil {
		fmt.Fprintln(os.Stderr, "bifrost env: Nix is required to compose port toolchains")
		return 2
	}
	argv := append([]string{"nix", "shell"}, installables...)
	argv = append(argv, "--command")
	argv = append(argv, command...)
	return runForeground(argv, "")
}

func expandPortSelection(requested, known []string) ([]string, error) {
	knownSet := make(map[string]bool, len(known))
	for _, name := range known {
		knownSet[name] = true
	}
	seen := map[string]bool{}
	var selected []string
	add := func(name string) error {
		if !knownSet[name] {
			return fmt.Errorf("unknown port %q (known: %s)", name, strings.Join(known, ", "))
		}
		if !seen[name] {
			seen[name] = true
			selected = append(selected, name)
		}
		return nil
	}
	for _, name := range requested {
		if name == "all" {
			for _, candidate := range known {
				if err := add(candidate); err != nil {
					return nil, err
				}
			}
			continue
		}
		if err := add(name); err != nil {
			return nil, err
		}
	}
	return selected, nil
}

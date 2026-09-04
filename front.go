package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const projectPin = ".bifrost-impl"

// userConfigDir uses %APPDATA%\bifrost on Windows, else
// $XDG_CONFIG_HOME/bifrost or ~/.config/bifrost.
func userConfigDir() string {
	home, _ := os.UserHomeDir()
	return userConfigDirFor(runtime.GOOS, os.Getenv, home)
}

// userConfigDirFor is the parameterised core (tested on any OS).
func userConfigDirFor(goos string, getenv func(string) string, home string) string {
	if goos == "windows" {
		base := getenv("APPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "bifrost")
	}
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "bifrost")
}

func globalPinPath() string { return filepath.Join(userConfigDir(), "impl") }

func readPin(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// resolveActiveImpl: --impl override > ./.bifrost-impl > global pin > default
// (shen-cl if present, else first available in implOrder). Returns (name, source).
func resolveActiveImpl(a *Adapters, override string) (string, string) {
	if override != "" {
		return override, "--impl"
	}
	cwd, _ := os.Getwd()
	if p := readPin(filepath.Join(cwd, projectPin)); p != "" {
		return p, projectPin
	}
	if p := readPin(globalPinPath()); p != "" {
		return p, globalPinPath()
	}
	if a.has("shen-cl") {
		return "shen-cl", "default"
	}
	for _, n := range a.names() {
		return n, "default"
	}
	return "", "default"
}

// Port banners put the Shen kernel version first (for S42 this is the integer
// `42`), followed by implementation/port versions. Capture either an integer
// or dotted version and use the first token so `--versions` reports the runtime
// kernel rather than the port release (e.g. shen-cl's 3.0.3).
var versionRe = regexp.MustCompile(`\b(\d+(?:\.\d+)?)\b`)

// probeKernelVersion runs a port's version probe live (README badges drift).
func probeKernelVersion(impl Impl) string {
	var c aCase
	if impl.Cfg.Version != nil {
		c = aCase{Mode: "version"}
	} else {
		c = aCase{Mode: "eval", Expr: "(version)"}
	}
	r := runInvocation(buildArgv(impl, c, ""), 30e9, false, "")
	if r.Timeout {
		return "timeout"
	}
	if m := versionRe.FindStringSubmatch(r.Out); m != nil {
		return m[1]
	}
	return "?"
}

// cmdLaunch implements run / eval / repl.
func cmdLaunch(verb string, rest []string, a *Adapters) int {
	fs := newFlagSet("bifrost " + verb)
	impl := fs.String("impl", "", "port to use (overrides pin/default)")
	raw := fs.Bool("raw", false, "print launcher output verbatim (skip normalisation)")
	var expr string
	var file string
	if verb == "eval" {
		fs.StringVar(&expr, "e", "", "expression to evaluate")
		fs.StringVar(&expr, "expr", "", "expression to evaluate")
	}
	if err := fs.Parse(reorderArgs(rest, "impl", "e", "expr")); err != nil {
		return 2
	}
	if verb == "run" {
		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "bifrost run: need a .shen file")
			return 2
		}
		file = fs.Arg(0)
	}
	if verb == "eval" && expr == "" {
		fmt.Fprintln(os.Stderr, "bifrost eval: need -e EXPR")
		return 2
	}

	name, source := resolveActiveImpl(a, *impl)
	if name == "" {
		fmt.Fprintln(os.Stderr, "bifrost: no impl available; enter a port's Nix environment with `bifrost env <impl> -- bifrost <command>`")
		return 2
	}
	im, err := a.resolveOne(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost: "+err.Error())
		return 2
	}

	if verb == "repl" {
		tmpl := im.Cfg.Repl
		if tmpl == nil {
			tmpl = im.Cfg.Eval
		}
		argv := make([]string, len(tmpl))
		for i, t := range tmpl {
			argv[i] = subTokens(t, map[string]string{"{bin}": im.Bin})
		}
		argv = launcherArgv(im.Cfg, argv)
		argv = wrapExecutable(argv)
		fmt.Fprintf(os.Stderr, "# bifrost repl on %s (%s)\n", name, source)
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				return ee.ExitCode()
			}
			fmt.Fprintln(os.Stderr, "bifrost: launcher error:", err)
			return 2
		}
		return 0
	}

	var c aCase
	if verb == "run" {
		abs, _ := filepath.Abs(file)
		c = aCase{Mode: "script", Program: abs}
	} else {
		c = aCase{Mode: "eval", Expr: expr}
	}
	r := runInvocation(buildArgv(im, c, ""), timeoutDefault, false, "")
	out := r.Out
	if !*raw {
		out = normalize(r.Out, nil)
	}
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	fmt.Print(out)
	if r.Timeout {
		fmt.Fprintf(os.Stderr, "bifrost: %s timed out\n", name)
		return 1
	}
	if r.Rc < 0 {
		return 1
	}
	return r.Rc
}

// cmdImpls lists every known port: availability, kernel version, status, active.
func cmdImpls(rest []string, a *Adapters) int {
	fs := newFlagSet("bifrost impls")
	versions := fs.Bool("versions", false, "probe each available port's kernel version live")
	if err := fs.Parse(reorderArgs(rest)); err != nil {
		return 2
	}
	available, skipped := a.discover(nil)
	active, source := resolveActiveImpl(a, "")
	fmt.Printf("%-13s %-11s %-8s %-12s %s\n", "impl", "state", "kernel", "status", "launcher/why")
	fmt.Println(strings.Repeat("-", 78))
	for _, name := range a.names() {
		cfg, _ := a.effective(name)
		status := cfg.Status
		if status == "" {
			status = "?"
		}
		star := ""
		if name == active {
			star = " *"
		}
		if im, ok := available[name]; ok {
			kern := cfg.Kernel
			if kern == "" {
				kern = "-"
			}
			if *versions {
				kern = probeKernelVersion(im)
			}
			fmt.Printf("%-13s %-11s %-8s %-12s %s%s\n", name, "installed", kern, status, im.Bin, star)
		} else {
			fmt.Printf("%-13s %-11s %-8s %-12s %s\n", name, "missing", "-", status, skipped[name])
		}
	}
	fmt.Println(strings.Repeat("-", 78))
	suffix := ""
	if *versions {
		suffix = "  [--versions probed live]"
	}
	fmt.Printf("active impl: %s (%s)%s\n", active, source, suffix)
	return 0
}

// cmdUse sets the active port (global, or --project pin).
func cmdUse(rest []string, a *Adapters) int {
	fs := newFlagSet("bifrost use")
	project := fs.Bool("project", false, "pin in ./"+projectPin+" instead of the global default")
	if err := fs.Parse(reorderArgs(rest)); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "bifrost use: need an impl name")
		return 2
	}
	name := fs.Arg(0)
	if !a.has(name) {
		fmt.Fprintf(os.Stderr, "bifrost: unknown impl %q (known: %s)\n", name, strings.Join(a.names(), ", "))
		return 2
	}
	var path string
	if *project {
		cwd, _ := os.Getwd()
		path = filepath.Join(cwd, projectPin)
	} else {
		path = globalPinPath()
		os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err := os.WriteFile(path, []byte(name+"\n"), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 1
	}
	fmt.Printf("active impl set to %s (%s)\n", name, path)
	cfg, _ := a.effective(name)
	if a.resolveBin(name, cfg) == "" {
		fmt.Fprintf(os.Stderr, "note: %s is not installed yet — `bifrost install %s`\n", name, name)
	}
	return 0
}

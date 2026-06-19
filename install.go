package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// runStep runs one install/build step with inherited stdio. A bare command
// (no path separator) is resolved via LookPath first, so Windows `npm`/
// `luarocks` resolve to npm.cmd/.bat which wrapExecutable then runs via cmd /c.
func runStep(argv []string, cwd string, env map[string]string) bool {
	argv = append([]string{}, argv...)
	if len(argv) > 0 && !strings.ContainsAny(argv[0], `/\`) {
		if resolved, err := exec.LookPath(argv[0]); err == nil {
			argv[0] = resolved
		}
	}
	suffix := ""
	if cwd != "" {
		suffix = "  (cwd=" + cwd + ")"
	}
	fmt.Fprintf(os.Stderr, "  $ %s%s\n", strings.Join(argv, " "), suffix)
	argv = wrapExecutable(argv)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr // build output to stderr
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	return cmd.Run() == nil
}

// cmdInstall installs a port via its declared backend.
func cmdInstall(rest []string, a *Adapters) int {
	fs := newFlagSet("bifrost install")
	method := fs.String("method", "", "override install method (brew/npm/luarocks/git-build)")
	git := fs.String("git", "", "git remote to build from (git-build; for forks)")
	ref := fs.String("ref", "", "git branch/tag to build (git-build)")
	force := fs.Bool("force", false, "reinstall even if present; allow experimental")
	if err := fs.Parse(reorderArgs(rest, "method", "git", "ref")); err != nil {
		return 2
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "bifrost install: need an impl name")
		return 2
	}
	name := fs.Arg(0)
	if !a.has(name) {
		fmt.Fprintf(os.Stderr, "bifrost: unknown impl %q (known: %s)\n", name, strings.Join(a.names(), ", "))
		return 2
	}
	cfg, err := a.effective(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	if a.resolveBin(name, cfg) != "" && !*force {
		fmt.Printf("%s is already installed (%s). Use --force to reinstall.\n", name, a.resolveBin(name, cfg))
		return 0
	}
	if cfg.Status == "experimental" && !*force {
		fmt.Fprintf(os.Stderr, "bifrost: %s is experimental and may not boot. Re-run with --force to try anyway.\n", name)
		return 2
	}
	spec := cfg.Install
	if spec == nil {
		spec = &InstallSpec{}
	}
	meth := *method
	if meth == "" {
		meth = spec.Method
	}
	if meth == "" {
		env := cfg.Env
		if env == "" {
			env = "BIFROST_?"
		}
		fmt.Fprintf(os.Stderr, "bifrost: no install method known for %s. Build it manually and point $%s at the launcher.\n", name, env)
		return 2
	}
	for _, tool := range spec.Needs {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "bifrost: %s needs %q on PATH to install via %s. Install %q first.\n", name, tool, meth, tool)
			return 2
		}
	}

	fmt.Printf("installing %s via %s ...\n", name, meth)
	ok := false
	pkg := spec.Package
	if pkg == "" {
		pkg = name
	}
	switch meth {
	case "brew":
		ok = runStep([]string{"brew", "install", pkg}, "", nil)
	case "npm":
		ok = runStep([]string{"npm", "install", "-g", pkg}, "", nil)
	case "luarocks":
		ok = runStep([]string{"luarocks", "install", pkg}, "", nil)
	case "git-build":
		ok = installGitBuild(name, cfg, spec, *git, *ref)
	default:
		fmt.Fprintf(os.Stderr, "bifrost: unknown install method %q\n", meth)
		return 2
	}
	if !ok {
		fmt.Fprintf(os.Stderr, "bifrost: install of %s failed.\n", name)
		return 1
	}
	if a.resolveBin(name, cfg) == "" {
		env := cfg.Env
		if env == "" {
			env = "BIFROST_?"
		}
		fmt.Fprintf(os.Stderr, "bifrost: %s installed but launcher still not found; set $%s to its path.\n", name, env)
		return 1
	}
	fmt.Printf("installed %s -> %s\n", name, a.resolveBin(name, cfg))
	return 0
}

// installGitBuild clones (if absent) the port repo and runs its build recipe.
// Remote/ref overridable for forks via --git/--ref or $BIFROST_<IMPL>_GIT/_REF.
func installGitBuild(name string, cfg Adapter, spec *InstallSpec, git, ref string) bool {
	envKey := strings.ToUpper(cfg.Env)
	if git == "" {
		git = os.Getenv(envKey + "_GIT")
	}
	if git == "" {
		git = spec.Git
	}
	if ref == "" {
		ref = os.Getenv(envKey + "_REF")
	}
	if ref == "" {
		ref = spec.Ref
	}
	if cfg.Build == nil || cfg.Build.Cwd == "" {
		fmt.Fprintf(os.Stderr, "bifrost: %s has no git-build recipe in adapters.json\n", name)
		return false
	}
	repoDir := cfg.Build.Cwd
	if _, err := os.Stat(filepath.Join(repoDir, ".git")); err != nil {
		if git == "" {
			fmt.Fprintf(os.Stderr, "bifrost: %s not cloned at %s and no git remote known (pass --git URL or set %s_GIT)\n", name, repoDir, envKey)
			return false
		}
		if !runStep([]string{"git", "clone", git, repoDir}, "", nil) {
			return false
		}
	}
	if ref != "" {
		if !runStep([]string{"git", "-C", repoDir, "checkout", ref}, "", nil) {
			return false
		}
	}
	argv := make([]string, len(cfg.Build.Argv))
	for i, x := range cfg.Build.Argv {
		argv[i] = subTokens(x, map[string]string{"{out}": cfg.Build.Out})
	}
	return runStep(argv, repoDir, nil)
}

// cmdBuild delegates to the ratatoskr CLI: `ratatoskr` on PATH, else the Python
// wrapper at $BIFROST_RATATOSKR_DIR (transition), to shake+build an artifact.
func cmdBuild(rest []string, a *Adapters) int {
	fs := newFlagSet("bifrost build")
	target := fs.String("target", "", "ratatoskr target (lisp/lua/go/rust/js/julia)")
	run := fs.Bool("run", false, "run the artifact after building")
	if err := fs.Parse(reorderArgs(rest, "target")); err != nil {
		return 2
	}
	if fs.NArg() < 2 || *target == "" {
		fmt.Fprintln(os.Stderr, "usage: bifrost build FILE OUTDIR --target T [--run]")
		return 2
	}
	file, outdir := fs.Arg(0), fs.Arg(1)
	sub := "build"
	if *run {
		sub = "run"
	}
	args := []string{sub, file, outdir, "--target", *target}

	if bin, err := exec.LookPath("ratatoskr"); err == nil {
		return runForeground(append([]string{bin}, args...), "")
	}
	if dir := os.Getenv("BIFROST_RATATOSKR_DIR"); dir != "" {
		py := filepath.Join(dir, "ratatoskr_cli.py")
		if _, err := os.Stat(py); err == nil {
			return runForeground(append([]string{"python3", py}, args...), "")
		}
	}
	fmt.Fprintln(os.Stderr, "bifrost: ratatoskr CLI not found (install it on PATH or set $BIFROST_RATATOSKR_DIR)")
	return 2
}

// runForeground runs argv with inherited stdio and returns its exit code.
func runForeground(argv []string, cwd string) int {
	argv = wrapExecutable(argv)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	return 0
}

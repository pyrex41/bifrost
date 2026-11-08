package main

import (
	"bufio"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// The agent skills ride in the binary, so `bifrost skills install` works from a
// `go install` or a release download with no checkout.
//
//go:embed skills
var embeddedSkills embed.FS

// skillsRoot is the embedded skills/ directory: one folder per skill.
func skillsRoot() fs.FS {
	sub, err := fs.Sub(embeddedSkills, "skills")
	if err != nil {
		panic(err)
	}
	return sub
}

// skillTargets maps each harness to where it discovers skills, globally (under
// $HOME) and per project (under --project DIR). raw has no convention: it
// writes the Markdown files alone into --dir.
var skillTargets = map[string]struct{ global, project string }{
	"claude":   {".claude/skills", ".claude/skills"},
	"codex":    {".agents/skills", ".agents/skills"},
	"pi":       {".pi/agent/skills", ".pi/skills"},
	"opencode": {".config/opencode/skills", ".opencode/skills"},
}

var skillTargetOrder = []string{"claude", "codex", "pi", "opencode"}

// skillNames lists the skills in fsys: each top-level directory that holds a
// SKILL.md.
func skillNames(fsys fs.FS) []string {
	entries, _ := fs.ReadDir(fsys, ".")
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := fs.Stat(fsys, path.Join(e.Name(), "SKILL.md")); err == nil {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// skillDescription reads the description line of a SKILL.md's front matter.
func skillDescription(fsys fs.FS, name string) string {
	b, err := fs.ReadFile(fsys, path.Join(name, "SKILL.md"))
	if err != nil {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	inFront := false
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if line == "---" {
			if inFront {
				break
			}
			inFront = true
			continue
		}
		if inFront && strings.HasPrefix(line, "description:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "description:"))
		}
	}
	return ""
}

func cmdSkills(rest []string) int {
	if len(rest) == 0 || rest[0] == "-h" || rest[0] == "--help" || rest[0] == "help" {
		fmt.Fprint(os.Stderr, skillsUsage)
		if len(rest) == 0 {
			return 2
		}
		return 0
	}
	switch rest[0] {
	case "list":
		for _, n := range skillNames(skillsRoot()) {
			fmt.Printf("%-24s %s\n", n, truncate(skillDescription(skillsRoot(), n), 100))
		}
		return 0
	case "install":
		return cmdSkillsInstall(rest[1:])
	case "check":
		return cmdSkillsCheck(rest[1:])
	}
	fmt.Fprintf(os.Stderr, "bifrost skills: unknown action %q\n%s", rest[0], skillsUsage)
	return 2
}

const skillsUsage = `usage:
  bifrost skills list
  bifrost skills install --target TARGET [--project DIR] [--force] [--dry-run] [SKILL...]
  bifrost skills install --target raw --dir DIR [--force] [SKILL...]
  bifrost skills check [--impl PORT] [--dir DIR] [-v] [SKILL...]

targets: claude, codex, pi, opencode, all (the four harnesses), raw (Markdown only)
  claude    ~/.claude/skills            DIR/.claude/skills
  codex     ~/.agents/skills            DIR/.agents/skills
  pi        ~/.pi/agent/skills          DIR/.pi/skills
  opencode  ~/.config/opencode/skills   DIR/.opencode/skills
--project DIR installs into the project instead of your home directory.
check typechecks every shen block in the skills on a port (see skills/README.md).
`

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func cmdSkillsInstall(rest []string) int {
	fsx := newFlagSet("bifrost skills install")
	target := fsx.String("target", "", "claude, codex, pi, opencode, all or raw")
	project := fsx.String("project", "", "install into this project instead of $HOME")
	dir := fsx.String("dir", "", "destination for --target raw")
	force := fsx.Bool("force", false, "replace skills that are already installed")
	dry := fsx.Bool("dry-run", false, "print what would be written")
	if err := fsx.Parse(reorderArgs(rest, "target", "project", "dir")); err != nil {
		return 2
	}
	names, ok := selectSkills(skillsRoot(), fsx.Args())
	if !ok {
		return 2
	}
	var dests []skillDest
	switch *target {
	case "":
		fmt.Fprint(os.Stderr, "bifrost skills install: need --target\n"+skillsUsage)
		return 2
	case "raw":
		if *dir == "" {
			fmt.Fprintln(os.Stderr, "bifrost skills install: --target raw needs --dir DIR")
			return 2
		}
		dests = []skillDest{{"raw", *dir, true}}
	default:
		if *dir != "" {
			fmt.Fprintln(os.Stderr, "bifrost skills install: --dir is only for --target raw; use --project")
			return 2
		}
		targets := []string{*target}
		if *target == "all" {
			targets = skillTargetOrder
		}
		base := *project
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Fprintln(os.Stderr, "bifrost skills install:", err)
				return 2
			}
			base = home
		}
		for _, t := range targets {
			spec, ok := skillTargets[t]
			if !ok {
				fmt.Fprintf(os.Stderr, "bifrost skills install: unknown target %q\n%s", t, skillsUsage)
				return 2
			}
			sub := spec.global
			if *project != "" {
				sub = spec.project
			}
			dests = append(dests, skillDest{t, filepath.Join(base, filepath.FromSlash(sub)), false})
		}
	}
	rc := 0
	for _, d := range dests {
		for _, n := range names {
			to := filepath.Join(d.root, n)
			if *dry {
				fmt.Printf("would install %s -> %s\n", n, to)
			}
			if err := installSkill(skillsRoot(), n, to, d.mdOnly, *force, *dry); err != nil {
				fmt.Fprintf(os.Stderr, "bifrost skills install: %s: %v\n", d.target, err)
				rc = 1
				continue
			}
			if !*dry {
				fmt.Printf("installed %s -> %s\n", n, to)
			}
		}
	}
	return rc
}

type skillDest struct {
	target string
	root   string
	mdOnly bool
}

func selectSkills(fsys fs.FS, args []string) ([]string, bool) {
	all := skillNames(fsys)
	if len(args) == 0 {
		return all, true
	}
	known := map[string]bool{}
	for _, n := range all {
		known[n] = true
	}
	for _, a := range args {
		if !known[a] {
			fmt.Fprintf(os.Stderr, "bifrost skills: no skill %q (have: %s)\n", a, strings.Join(all, ", "))
			return nil, false
		}
	}
	return args, true
}

// installSkill copies NAME from fsys into dest. An existing dest is left
// alone unless force, in which case it is replaced whole, so files a newer
// version dropped do not linger. mdOnly copies only Markdown files.
func installSkill(fsys fs.FS, name, dest string, mdOnly, force, dry bool) error {
	if _, err := os.Lstat(dest); err == nil {
		if !force {
			return fmt.Errorf("%s exists (use --force to replace it)", dest)
		}
		if !dry {
			if err := os.RemoveAll(dest); err != nil {
				return err
			}
		}
	}
	src := name
	return fs.WalkDir(fsys, src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, src), "/")
		out := filepath.Join(dest, filepath.FromSlash(rel))
		if d.IsDir() {
			if dry {
				return nil
			}
			return os.MkdirAll(out, 0o755)
		}
		if mdOnly && !strings.HasSuffix(p, ".md") {
			return nil
		}
		if dry {
			fmt.Printf("  %s\n", out)
			return nil
		}
		b, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o644)
	})
}

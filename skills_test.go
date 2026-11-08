package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEmbeddedSkillsHaveFrontMatter(t *testing.T) {
	names := skillNames(skillsRoot())
	if len(names) == 0 {
		t.Fatal("no embedded skills")
	}
	for _, n := range names {
		b, err := fs.ReadFile(skillsRoot(), n+"/SKILL.md")
		if err != nil {
			t.Fatal(err)
		}
		s := strings.ReplaceAll(string(b), "\r\n", "\n")
		if !strings.HasPrefix(s, "---\nname: "+n+"\n") {
			t.Errorf("%s: SKILL.md must start with front matter naming it %q", n, n)
		}
		if skillDescription(skillsRoot(), n) == "" {
			t.Errorf("%s: no description", n)
		}
		if _, err := extractShenBlocks(n, s); err != nil {
			t.Error(err)
		}
	}
}

func TestSkillsInstallTargets(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	name := skillNames(skillsRoot())[0]
	if rc := cmdSkills([]string{"install", "--target", "all", name}); rc != 0 {
		t.Fatalf("install all: rc %d", rc)
	}
	for _, sub := range []string{".claude/skills", ".agents/skills", ".pi/agent/skills", ".config/opencode/skills"} {
		if _, err := os.Stat(filepath.Join(home, sub, name, "SKILL.md")); err != nil {
			t.Errorf("%s: %v", sub, err)
		}
	}
	// A second install refuses to clobber, and --force replaces.
	if rc := cmdSkills([]string{"install", "--target", "claude", name}); rc == 0 {
		t.Error("reinstall without --force succeeded")
	}
	stale := filepath.Join(home, ".claude/skills", name, "stale.md")
	os.WriteFile(stale, []byte("x"), 0o644)
	if rc := cmdSkills([]string{"install", "--target", "claude", "--force", name}); rc != 0 {
		t.Fatalf("--force: rc %d", rc)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("--force kept a file the skill does not have")
	}

	proj := t.TempDir()
	if rc := cmdSkills([]string{"install", "--target", "pi", "--project", proj, name}); rc != 0 {
		t.Fatalf("project install: rc %d", rc)
	}
	if _, err := os.Stat(filepath.Join(proj, ".pi/skills", name, "SKILL.md")); err != nil {
		t.Error(err)
	}
}

func TestSkillsInstallRawIsMarkdownOnly(t *testing.T) {
	fsys := fstest.MapFS{
		"s/SKILL.md":            {Data: []byte("---\nname: s\n---\n")},
		"s/references/types.md": {Data: []byte("# t\n")},
		"s/examples/a.shen":     {Data: []byte("(+ 1 2)\n")},
	}
	dest := filepath.Join(t.TempDir(), "s")
	if err := installSkill(fsys, "s", dest, true, false, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"SKILL.md", "references/types.md"} {
		if _, err := os.Stat(filepath.Join(dest, f)); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "examples/a.shen")); err == nil {
		t.Error("raw install copied a non-Markdown file")
	}
	if rc := cmdSkills([]string{"install", "--target", "raw"}); rc != 2 {
		t.Errorf("raw without --dir: rc %d, want 2", rc)
	}
}

func TestExtractShenBlocks(t *testing.T) {
	md := "# x\n\n```shen\n(define a {number --> number} X -> X)\n```\n\n" +
		"<!-- reject -->\n```shen\n(define b {number --> string} X -> X)\n```\n\n" +
		"<!-- skip -->\n```shen\n...\n```\n\n```go\nfunc() {}\n```\n\n" +
		"<!-- fresh -->\n````shen\n(+ 1 2)\n````\n"
	blocks, err := extractShenBlocks("x.md", md)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		mode  string
		fresh bool
		line  int
	}{{"accept", false, 3}, {"reject", false, 8}, {"skip", false, 13}, {"accept", true, 22}}
	if len(blocks) != len(want) {
		t.Fatalf("got %d blocks, want %d", len(blocks), len(want))
	}
	for i, w := range want {
		if blocks[i].Mode != w.mode || blocks[i].Fresh != w.fresh || blocks[i].Line != w.line {
			t.Errorf("block %d: got %s fresh=%v line %d, want %+v", i, blocks[i].Mode, blocks[i].Fresh, blocks[i].Line, w)
		}
	}
	if _, err := extractShenBlocks("y.md", "```shen\n(+ 1 2)\n"); err == nil {
		t.Error("unclosed fence not reported")
	}
}

func TestSkillProgramsAccumulate(t *testing.T) {
	fsys := fstest.MapFS{
		"s/SKILL.md": {Data: []byte("---\nname: s\n---\n```shen\nA\n```\n<!-- reject -->\n```shen\nB\n```\n```shen\nC\n```\n<!-- fresh -->\n```shen\nD\n```\n")},
	}
	progs, err := skillPrograms(fsys, []string{"s"})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, p := range progs {
		code := ""
		for _, b := range append(p.Before, p.Block) {
			code += b.Code
		}
		got = append(got, strings.ReplaceAll(code, "\n", ""))
	}
	if strings.Join(got, ",") != "A,AB,AC,D" {
		t.Errorf("programs %v, want A,AB,AC,D", got)
	}
}

func TestJudgeCheck(t *testing.T) {
	cases := []struct {
		mode, out string
		pass      bool
	}{
		{"accept", "x\n" + markOK + "\n", true},
		{"accept", "type error in rule 1 of f\n", false},
		{"reject", markRejected + " type error in rule 1 of f\n", true},
		{"reject", markRejected + " f is undefined\n", false},
		{"reject", markAccepted + "\n", false},
		{"reject", "type error\n", false},
	}
	for _, c := range cases {
		if pass, _ := judgeCheck(c.mode, c.out); pass != c.pass {
			t.Errorf("judgeCheck(%s, %q) = %v", c.mode, c.out, pass)
		}
	}
}

func TestSkillDescriptionToleratesCRLF(t *testing.T) {
	fsys := fstest.MapFS{"s/SKILL.md": {Data: []byte("---\r\nname: s\r\ndescription: d\r\n---\r\n")}}
	if got := skillDescription(fsys, "s"); got != "d" {
		t.Errorf("description %q, want d", got)
	}
}

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

func TestExtractShenRejectReason(t *testing.T) {
	cases := []struct {
		name, annotation, reason, wantErr string
	}{
		{"legacy", "<!-- reject -->", "", ""},
		{"function", "<!-- reject: wrong-unit -->", "wrong-unit", ""},
		{"diagnostic phrase", "<!-- reject: rule 2 of wrong -->", "rule 2 of wrong", ""},
		{"literal punctuation", "<!-- reject: valid? -->", "valid?", ""},
		{"whitespace", "  <!-- reject:   wrong  -->\r", "wrong", ""},
		{"empty", "<!-- reject: -->", "", "non-empty diagnostic substring"},
		{"unclosed", "<!-- reject: wrong", "", "must end with -->"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			md := c.annotation + "\n\n```shen\n(define wrong {number --> string} X -> X)\n```\n"
			blocks, err := extractShenBlocks("x.md", md)
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), "x.md:3:") || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want x.md:3 and %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(blocks) != 1 || blocks[0].Mode != "reject" || blocks[0].RejectReason != c.reason {
				t.Fatalf("blocks = %+v, want one reject with reason %q", blocks, c.reason)
			}
		})
	}
}

func TestSkillProgramsAccumulate(t *testing.T) {
	fsys := fstest.MapFS{
		"s/SKILL.md": {Data: []byte("---\nname: s\n---\n```shen\nA\n```\n<!-- reject: broken -->\n```shen\nB\n```\n```shen\nC\n```\n<!-- fresh -->\n```shen\nD\n```\n")},
	}
	progs, err := skillPrograms(fsys, []string{"s"})
	if err != nil {
		t.Fatal(err)
	}
	if len(progs) != 4 {
		t.Fatalf("got %d programs, want 4", len(progs))
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
	if progs[1].Block.RejectReason != "broken" {
		t.Errorf("reject reason = %q, want broken", progs[1].Block.RejectReason)
	}
}

func TestJudgeCheck(t *testing.T) {
	cases := []struct {
		name, mode, reason, out string
		pass                    bool
		wantWhy                 string
	}{
		{"accept", "accept", "", "x\n" + markOK + "\n", true, ""},
		{"missing success marker", "accept", "", "type error in rule 1 of f\n", false, "did not load"},
		{"generic rejection", "reject", "", markRejected + " type error in rule 1 of f\n", true, ""},
		{"named rejection", "reject", "wrong", markRejected + " type error in rule 1 of wrong\n", true, ""},
		{"diagnostic substring", "reject", "rule 2 of wrong", markRejected + " type error in rule 2 of wrong\n", true, ""},
		{"different failure", "reject", "wrong", markRejected + " type error in rule 1 of unrelated\n", false, "did not contain \"wrong\""},
		{"not a type error", "reject", "wrong", markRejected + " wrong is undefined\n", false, "not with a type error"},
		{"reason before diagnostic", "reject", "wrong", "(fn wrong)\n" + markRejected + " type error in rule 1 of unrelated\n", false, "did not contain \"wrong\""},
		{"reason after diagnostic", "reject", "wrong", markRejected + " type error in rule 1 of unrelated\nwrong\n", false, "did not contain \"wrong\""},
		{"literal match", "reject", "valid?", markRejected + " type error in rule 1 of valid?\n", true, ""},
		{"not a regular expression", "reject", "valid?", markRejected + " type error in rule 1 of valid\n", false, "did not contain \"valid?\""},
		{"case sensitive", "reject", "Wrong", markRejected + " type error in rule 1 of wrong\n", false, "did not contain \"Wrong\""},
		{"unexpected acceptance", "reject", "wrong", markAccepted + "\n", false, "accepted"},
		{"earlier failure", "reject", "", "type error\n", false, "an earlier block"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			block := shenBlock{Mode: c.mode, RejectReason: c.reason}
			pass, why := judgeCheck(block, runResult{Out: c.out})
			if pass != c.pass || (pass && why != "") || !strings.Contains(why, c.wantWhy) {
				t.Errorf("judgeCheck(%+v, %q) = (%v, %q), want (%v, %q)", block, c.out, pass, why, c.pass, c.wantWhy)
			}
		})
	}
}

func TestJudgeCheckRequiresSuccessfulInvocation(t *testing.T) {
	for _, mode := range []string{"accept", "reject"} {
		// Both forms of success marker must be ignored after a failed process.
		out := markOK + "\n" + markRejected + " type error in rule 1 of wrong\n"
		for _, c := range []struct {
			name    string
			rc      int
			timeout bool
			wantWhy string
		}{
			{"nonzero exit", 7, false, "status 7"},
			{"unknown exit", -1, false, "status -1"},
			{"timeout", -1, true, "timed out"},
		} {
			t.Run(mode+"/"+c.name, func(t *testing.T) {
				block := shenBlock{Mode: mode, RejectReason: "wrong"}
				result := runResult{Rc: c.rc, Out: out, Timeout: c.timeout}
				if pass, why := judgeCheck(block, result); pass || !strings.Contains(why, c.wantWhy) {
					t.Errorf("judgeCheck(%+v, %+v) = (%v, %q), want failure containing %q", block, result, pass, why, c.wantWhy)
				}
			})
		}
	}
}

func TestSkillDescriptionToleratesCRLF(t *testing.T) {
	fsys := fstest.MapFS{"s/SKILL.md": {Data: []byte("---\r\nname: s\r\ndescription: d\r\n---\r\n")}}
	if got := skillDescription(fsys, "s"); got != "d" {
		t.Errorf("description %q, want d", got)
	}
}

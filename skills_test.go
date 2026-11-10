package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
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
		"s/SKILL.md":                {Data: []byte("---\nname: s\n---\n")},
		"s/references/types.md":     {Data: []byte("# t\n")},
		"s/references/deep/case.md": {Data: []byte("<!-- reject: wrong -->\n```shen\n(define wrong {number --> string} X -> X)\n```\n")},
		"s/examples/a.shen":         {Data: []byte("(+ 1 2)\n")},
	}
	dest := filepath.Join(t.TempDir(), "s")
	if err := installSkill(fsys, "s", dest, true, false, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"SKILL.md", "references/types.md", "references/deep/case.md"} {
		got, err := os.ReadFile(filepath.Join(dest, f))
		if err != nil {
			t.Fatal(err)
		}
		want, err := fs.ReadFile(fsys, "s/"+f)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: installed bytes differ: %v", f, err)
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
				if err == nil || !strings.Contains(err.Error(), "x.md:1:") || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v, want x.md:1 and %q", err, c.wantErr)
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
		{"CRLF marker", "accept", "", markOK + "\r\n", true, ""},
		{"embedded success marker", "accept", "", "error: " + markOK + "\n", false, "did not load"},
		{"partial success marker", "accept", "", markOK + "-NOT\n", false, "did not load"},
		{"missing success marker", "accept", "", "type error in rule 1 of f\n", false, "did not load"},
		{"generic rejection", "reject", "", markRejected + " type error in rule 1 of f\n", true, ""},
		{"named rejection", "reject", "wrong", markRejected + " type error in rule 1 of wrong\n", true, ""},
		{"diagnostic substring", "reject", "rule 2 of wrong", markRejected + " type error in rule 2 of wrong\n", true, ""},
		{"different failure", "reject", "wrong", markRejected + " type error in rule 1 of unrelated\n", false, "did not contain \"wrong\""},
		{"function suffix", "reject", "wrong", markRejected + " type error in rule 1 of not-wrong\n", false, "whole tokens"},
		{"function prefix", "reject", "wrong", markRejected + " type error in rule 1 of wrong-helper\n", false, "whole tokens"},
		{"not a type error", "reject", "wrong", markRejected + " wrong is undefined\n", false, "not with a type error"},
		{"incidental type error text", "reject", "wrong", markRejected + " not a type error in wrong\n", false, "not with a type error"},
		{"invalid rule number", "reject", "wrong", markRejected + " type error in rule zero of wrong\n", false, "not with a type error"},
		{"negative rule number", "reject", "wrong", markRejected + " type error in rule -1 of wrong\n", false, "not with a type error"},
		{"embedded rejection marker", "reject", "wrong", "error: " + markRejected + " type error in rule 1 of wrong\n", false, "an earlier block"},
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

func TestMalformedSkillAnnotations(t *testing.T) {
	for _, md := range []string{
		"<!-- rejeect: f -->\n```shen\ntrue\n```",
		"<!-- untype -->\n```shen\ntrue\n```",
		"<!-- fresh: yes -->\n```shen\ntrue\n```",
		"<!-- reject: -->\n```shen\ntrue\n```",
		"<!-- reject: f\n```shen\ntrue\n```",
		"<!-- skip -->\n<!-- fresh -->\n```shen\ntrue\n```",
		"<!-- fresh -->\nprose\n```shen\ntrue\n```",
		"<!-- untyped -->\n```go\ntrue\n```",
		"<!-- reject -->\n",
	} {
		t.Run(md, func(t *testing.T) {
			if _, err := extractShenBlocks("bad.md", md); err == nil {
				t.Fatal("malformed annotation silently accepted")
			}
		})
	}
}

func TestSkillProgramsNestedAndIndependent(t *testing.T) {
	fsys := fstest.MapFS{
		"s/SKILL.md":                         {Data: []byte("```shen\nentry\n```\n")},
		"s/references/deep/a.md":             {Data: []byte("```shen\nA\n```\n<!-- skip -->\n```shen\nignored\n```\n<!-- reject: bad -->\n```shen\nbad\n```\n```shen\nB\n```\n<!-- fresh -->\n```shen\nC\n```\n```shen\nD\n```\n")},
		"s/references/deep/b.md":             {Data: []byte("```shen\nseparate\n```\n")},
		"s/references/deep/not-markdown.txt": {Data: []byte("```shen\nunseen\n```\n")},
	}
	progs, err := skillPrograms(fsys, []string{"s"})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range progs {
		var code string
		for _, b := range append(append([]shenBlock(nil), p.Before...), p.Block) {
			code += strings.TrimSpace(b.Code)
		}
		got = append(got, code)
	}
	if strings.Join(got, ",") != "entry,A,Abad,AB,C,CD,separate" {
		t.Fatalf("nested/isolation programs = %v", got)
	}
}

func TestCheckExprLoadBoundaries(t *testing.T) {
	blocks := []shenBlock{{Mode: "accept"}, {Mode: "accept", Untyped: true}, {Mode: "reject"}}
	expr := checkExpr(blocks, []string{"setup.shen", "untyped.shen", "negative.shen"})
	for _, want := range []string{
		`(do (tc +) (load "setup.shen"))`,
		`(do (tc -) (load "untyped.shen") (tc +))`,
		`(trap-error (do (do (tc +) (load "negative.shen"))`,
	} {
		if !strings.Contains(expr, want) {
			t.Errorf("expression missing %s: %s", want, expr)
		}
	}
	if strings.Index(expr, `load "untyped.shen"`) > strings.Index(expr, "trap-error") {
		t.Fatal("setup is inside rejection trap")
	}
}

// Exercise runInvocation, not just synthetic runResult values. These subprocesses
// deliberately print convincing markers before exiting or hanging.
func TestSkillsCheckProcessHelper(t *testing.T) {
	mode := os.Getenv("BIFROST_SKILLS_PROCESS_HELPER")
	if mode == "" {
		return
	}
	fmt.Println(markOK)
	fmt.Println(markRejected + " type error in rule 1 of intended")
	if mode == "timeout" {
		time.Sleep(10 * time.Second)
	}
	os.Exit(7)
}

func TestSkillsCheckRealProcessFailures(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"exit", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("BIFROST_SKILLS_PROCESS_HELPER", mode)
			r := runInvocation([]string{exe, "-test.run=^TestSkillsCheckProcessHelper$"}, time.Second, true, "")
			if !strings.Contains(r.Out, markOK) || !strings.Contains(r.Out, markRejected) {
				t.Fatalf("helper did not emit partial output: %+v", r)
			}
			if mode == "timeout" && !r.Timeout || mode == "exit" && r.Rc != 7 {
				t.Fatalf("unexpected process outcome: %+v", r)
			}
			for _, blockMode := range []string{"accept", "reject"} {
				if pass, why := judgeCheck(shenBlock{Mode: blockMode, RejectReason: "intended"}, r); pass {
					t.Fatalf("%s credited failed process: %+v (%s)", blockMode, r, why)
				}
			}
		})
	}
}

func TestSkillsSourceEmbeddedInstalledBytes(t *testing.T) {
	root := skillsRoot()
	for _, name := range skillNames(root) {
		for _, mdOnly := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/raw=%v", name, mdOnly), func(t *testing.T) {
				dest := filepath.Join(t.TempDir(), name)
				if err := installSkill(root, name, dest, mdOnly, false, false); err != nil {
					t.Fatal(err)
				}
				var expected []string
				err := fs.WalkDir(root, name, func(p string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return err
					}
					if mdOnly && !strings.HasSuffix(p, ".md") {
						return nil
					}
					embedded, err := fs.ReadFile(root, p)
					if err != nil {
						return err
					}
					source, err := os.ReadFile(filepath.Join("skills", filepath.FromSlash(p)))
					if err != nil {
						return err
					}
					if !bytes.Equal(source, embedded) {
						t.Errorf("%s: source differs from embedded bytes", p)
					}
					rel := strings.TrimPrefix(p, name+"/")
					expected = append(expected, rel)
					installed, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
					if err != nil {
						return err
					}
					if !bytes.Equal(source, installed) {
						t.Errorf("%s: installation differs from checked source", p)
					}
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				var sourceFiles []string
				if err := fs.WalkDir(os.DirFS("skills"), name, func(p string, d fs.DirEntry, err error) error {
					if err == nil && !d.IsDir() && (!mdOnly || strings.HasSuffix(p, ".md")) {
						sourceFiles = append(sourceFiles, strings.TrimPrefix(p, name+"/"))
					}
					return err
				}); err != nil {
					t.Fatal(err)
				}
				if strings.Join(sourceFiles, "\n") != strings.Join(expected, "\n") {
					t.Fatalf("embedded file set %v, want source set %v", expected, sourceFiles)
				}
				var installed []string
				if err := fs.WalkDir(os.DirFS(dest), ".", func(p string, d fs.DirEntry, err error) error {
					if err == nil && !d.IsDir() {
						installed = append(installed, p)
					}
					return err
				}); err != nil {
					t.Fatal(err)
				}
				if strings.Join(installed, "\n") != strings.Join(expected, "\n") {
					t.Fatalf("installed file set %v, want %v", installed, expected)
				}
			})
		}
	}
}

// Opt in to actual port execution; ordinary go test remains runtime-independent.
// BIFROST_SKILLS_TEST_IMPLS=shen-jit,shen-scheme go test -run TestSkillsCheckerOnPorts -v
func TestSkillsCheckerOnPorts(t *testing.T) {
	names := os.Getenv("BIFROST_SKILLS_TEST_IMPLS")
	if names == "" {
		t.Skip("set BIFROST_SKILLS_TEST_IMPLS to run adversarial checks on real ports")
	}
	adapters, err := loadAdapters()
	if err != nil {
		t.Fatal(err)
	}
	accept := func(code string) shenBlock { return shenBlock{Mode: "accept", Code: code} }
	reject := func(code string) shenBlock {
		return shenBlock{Mode: "reject", RejectReason: "intended", Code: code}
	}
	good := `(define intended {number --> number} X -> X)`
	bad := `(define intended {number --> string} X -> X)`
	cases := []struct {
		name   string
		before []shenBlock
		block  shenBlock
		pass   bool
	}{
		{"typed control", nil, accept(good), true},
		{"checking enabled before load", nil, accept(bad), false},
		{"intended rejection", nil, reject(bad), true},
		{"unexpected acceptance", nil, reject(good), false},
		{"wrong definition", nil, reject(`(define unrelated {number --> string} X -> X)` + bad), false},
		{"substring definition", nil, reject(`(define unintended {number --> string} X -> X)`), false},
		{"undefined helper", nil, reject(`(define intended {number --> number} X -> (missing-helper X))`), false},
		{"undefined zero arity helper", nil, reject(`(define intended {--> number} -> (missing-zero))`), false},
		{"same block forward helper", nil, reject(`(define intended {number --> string} X -> (late-helper X)) (define late-helper {number --> number} X -> X)`), true},
		{"same block accepted setup", nil, reject(`(define helper {number --> number} X -> X)` + bad), true},
		{"valid helper", []shenBlock{accept(`(define helper {number --> number} X -> X)`)}, accept(`(define client {number --> number} X -> (helper X))`), true},
		{"setup type error", []shenBlock{accept(bad)}, reject(bad), false},
		{"setup runtime error", []shenBlock{accept(`(simple-error "setup failed")`)}, reject(bad), false},
		{"runtime assertion control", nil, accept(`(if (= 1 1) true (simple-error "mismatch"))`), true},
		{"runtime assertion mismatch", nil, accept(`(if (= 1 2) true (simple-error "mismatch"))`), false},
		{"runtime error is not rejection", nil, reject(`(simple-error "intended")`), false},
		{"typed load resets tc", []shenBlock{accept(`(tc -)`)}, reject(bad), true},
		{"untyped setup restores tc", []shenBlock{{Mode: "accept", Untyped: true, Code: `(define helper X -> X)`}}, reject(bad), true},
	}
	// Exercise discovery and accumulation too: rejected definitions and fresh
	// contexts must not reach the subsequent check's process.
	fixture := fstest.MapFS{"s/SKILL.md": {Data: []byte("```shen\n(define earlier {number --> number} X -> X)\n```\n<!-- reject: poison -->\n```shen\n(define poison {number --> string} X -> X)\n```\n```shen\n(define reuse {number --> number} X -> (poison X))\n```\n<!-- fresh -->\n```shen\n(define reset {number --> number} X -> (earlier X))\n```\n")}}
	programs, err := skillPrograms(fixture, []string{"s"})
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"accepted context", "rejected context", "rejected definition not reused", "fresh context"} {
		p := programs[i]
		cases = append(cases, struct {
			name   string
			before []shenBlock
			block  shenBlock
			pass   bool
		}{name, p.Before, p.Block, i < 2})
	}
	for _, name := range strings.Split(names, ",") {
		t.Run(name, func(t *testing.T) {
			impl, err := adapters.resolveOne(name)
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					blocks := append(append([]shenBlock(nil), c.before...), c.block)
					var files []string
					for i, b := range blocks {
						file := filepath.Join(t.TempDir(), fmt.Sprintf("block-%d.shen", i))
						if err := os.WriteFile(file, []byte(b.Code+"\n"), 0o644); err != nil {
							t.Fatal(err)
						}
						files = append(files, file)
					}
					r := runInvocation(buildArgv(impl, aCase{Mode: "eval", Expr: checkExpr(blocks, files)}, ""), 10*time.Second, true, "")
					if pass, why := judgeCheck(c.block, r); pass != c.pass {
						t.Fatalf("pass=%v want %v: %s\n%+v", pass, c.pass, why, r)
					}
				})
			}
		})
	}
}

func TestSkillDescriptionToleratesCRLF(t *testing.T) {
	fsys := fstest.MapFS{"s/SKILL.md": {Data: []byte("---\r\nname: s\r\ndescription: d\r\n---\r\n")}}
	if got := skillDescription(fsys, "s"); got != "d" {
		t.Errorf("description %q, want d", got)
	}
}

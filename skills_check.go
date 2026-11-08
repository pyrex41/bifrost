package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// A skill's Shen examples are checked, not trusted. Every fenced block tagged
// `shen` in a skill's Markdown is typechecked under (tc +) on a real port. The
// blocks of one file accumulate, like a literate program, so a later block may
// use an earlier one's datatypes and functions. Each block is loaded as a file
// of its own, after the earlier ones: a datatype is compiled when its file is
// read, so a side condition may only call functions loaded before it. An HTML
// comment on the line
// before the fence changes what is expected of the block:
//
//	<!-- reject -->  the checker must refuse it with a type error; it is not
//	                 kept for later blocks
//	<!-- skip -->    not checked (a fragment, or another language's syntax)
//	<!-- fresh -->   checked, but starts the accumulation again
//	<!-- untyped --> loaded under (tc -), as defprolog and code without
//	                 signatures must be; later blocks are typechecked again
type shenBlock struct {
	File    string // path of the Markdown file within the skills tree
	Line    int    // line of the opening fence
	Mode    string // "accept", "reject" or "skip"
	Fresh   bool
	Untyped bool
	Code    string
}

// extractShenBlocks returns the shen blocks of one Markdown file, in order.
func extractShenBlocks(file, text string) ([]shenBlock, error) {
	var blocks []shenBlock
	lines := strings.Split(text, "\n")
	prev := ""
	for i := 0; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r")
		trimmed := strings.TrimSpace(line)
		if fence, info, ok := openingFence(trimmed); ok {
			start := i + 1
			var body []string
			closed := false
			for i++; i < len(lines); i++ {
				l := strings.TrimRight(lines[i], "\r")
				if strings.HasPrefix(strings.TrimSpace(l), fence) && strings.Trim(strings.TrimSpace(l), string(fence[0])) == "" {
					closed = true
					break
				}
				body = append(body, l)
			}
			if !closed {
				return nil, fmt.Errorf("%s:%d: unclosed code fence", file, start)
			}
			if info == "shen" {
				b := shenBlock{File: file, Line: start, Mode: "accept", Code: strings.Join(body, "\n") + "\n"}
				switch prev {
				case "<!-- reject -->":
					b.Mode = "reject"
				case "<!-- skip -->":
					b.Mode = "skip"
				case "<!-- fresh -->":
					b.Fresh = true
				case "<!-- untyped -->":
					b.Untyped = true
				}
				blocks = append(blocks, b)
			}
			prev = ""
			continue
		}
		if trimmed != "" {
			prev = trimmed
		}
	}
	return blocks, nil
}

// openingFence recognises ``` or ~~~ fences and returns the fence and the first
// word of its info string.
func openingFence(line string) (string, string, bool) {
	for _, c := range []string{"`", "~"} {
		if strings.HasPrefix(line, c+c+c) {
			n := 0
			for n < len(line) && string(line[n]) == c {
				n++
			}
			info := strings.Fields(line[n:])
			word := ""
			if len(info) > 0 {
				word = strings.ToLower(info[0])
			}
			return line[:n], word, true
		}
	}
	return "", "", false
}

// shenProgram is one check: the block, and the earlier accepted blocks of its
// file, which are loaded first.
type shenProgram struct {
	Block  shenBlock
	Before []shenBlock
}

// skillPrograms turns the blocks of every Markdown file of the named skills
// into programs to check.
func skillPrograms(fsys fs.FS, names []string) ([]shenProgram, error) {
	var progs []shenProgram
	for _, n := range names {
		var files []string
		err := fs.WalkDir(fsys, n, func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".md") {
				files = append(files, p)
			}
			return err
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(files)
		for _, f := range files {
			b, err := fs.ReadFile(fsys, f)
			if err != nil {
				return nil, err
			}
			blocks, err := extractShenBlocks(f, string(b))
			if err != nil {
				return nil, err
			}
			var acc []shenBlock
			for _, bl := range blocks {
				if bl.Mode == "skip" {
					continue
				}
				if bl.Fresh {
					acc = nil
				}
				progs = append(progs, shenProgram{Block: bl, Before: append([]shenBlock(nil), acc...)})
				if bl.Mode == "accept" {
					acc = append(acc, bl)
				}
			}
		}
	}
	return progs, nil
}

const (
	markOK       = "BIFROST-SKILL-OK"
	markAccepted = "BIFROST-SKILL-ACCEPTED"
	markRejected = "BIFROST-SKILL-REJECTED"
)

// checkExpr is the expression a port evaluates to check one program: load the
// earlier files under (tc +), then the block's own file.
// files[i] holds blocks[i]; the last is the block under check.
func checkExpr(blocks []shenBlock, files []string) string {
	q := func(f string) string { return `"` + strings.ReplaceAll(f, `\`, `/`) + `"` }
	load := func(bl shenBlock, f string) string {
		if bl.Untyped {
			return "(do (tc -) (load " + q(f) + ") (tc +))"
		}
		return "(load " + q(f) + ")"
	}
	var b strings.Builder
	b.WriteString("(do (tc +) ")
	n := len(blocks) - 1
	for i := 0; i < n; i++ {
		b.WriteString(load(blocks[i], files[i]) + " ")
	}
	if blocks[n].Mode == "reject" {
		fmt.Fprintf(&b, `(trap-error (do %s (output "~%%%s~%%")) (/. E (output "~%%%s ~A~%%" (error-to-string E)))))`, load(blocks[n], files[n]), markAccepted, markRejected)
	} else {
		fmt.Fprintf(&b, `%s (output "~%%%s~%%"))`, load(blocks[n], files[n]), markOK)
	}
	return b.String()
}

// judgeCheck reads a port's output for one program.
func judgeCheck(mode, out string) (bool, string) {
	if mode == "reject" {
		i := strings.Index(out, markRejected)
		if i < 0 {
			if strings.Contains(out, markAccepted) {
				return false, "accepted, but the block is marked <!-- reject -->"
			}
			return false, "an earlier block of the file did not load"
		}
		why := strings.TrimSpace(firstLine(out[i+len(markRejected):]))
		if !strings.Contains(why, "type error") {
			return false, "failed, but not with a type error: " + why
		}
		return true, ""
	}
	if !strings.Contains(out, markOK) {
		return false, "did not load under (tc +)"
	}
	return true, ""
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func cmdSkillsCheck(rest []string) int {
	fsx := newFlagSet("bifrost skills check")
	impl := fsx.String("impl", "", "port to use (overrides pin/default)")
	dir := fsx.String("dir", "", "check the skills in this directory instead of the embedded ones")
	verbose := fsx.Bool("v", false, "print each port's output")
	if err := fsx.Parse(reorderArgs(rest, "impl", "dir")); err != nil {
		return 2
	}
	fsys := skillsRoot()
	if *dir != "" {
		fsys = os.DirFS(*dir)
	}
	names, ok := selectSkills(fsys, fsx.Args())
	if !ok {
		return 2
	}
	progs, err := skillPrograms(fsys, names)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost skills check:", err)
		return 2
	}
	a, err := loadAdapters()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	name, source := resolveActiveImpl(a, *impl)
	if name == "" {
		fmt.Fprintln(os.Stderr, "bifrost: no impl available")
		return 2
	}
	im, err := a.resolveOne(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost: "+err.Error())
		return 2
	}
	tmp, err := os.MkdirTemp("", "bifrost-skills-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost skills check:", err)
		return 2
	}
	defer os.RemoveAll(tmp)

	fmt.Fprintf(os.Stderr, "# checking %d blocks on %s (%s)\n", len(progs), name, source)
	failed := 0
	for i, p := range progs {
		var files []string
		blocks := append(append([]shenBlock(nil), p.Before...), p.Block)
		for j, bl := range blocks {
			f := filepath.Join(tmp, fmt.Sprintf("check-%03d-%02d.shen", i, j))
			if err := os.WriteFile(f, []byte(bl.Code), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "bifrost skills check:", err)
				return 2
			}
			files = append(files, f)
		}
		expr := checkExpr(blocks, files)
		r := runInvocation(buildArgv(im, aCase{Mode: "eval", Expr: expr}, ""), timeoutDefault, true, "")
		pass, why := judgeCheck(p.Block.Mode, r.Out)
		if r.Timeout {
			pass, why = false, "timed out"
		}
		where := path.Clean(p.Block.File) + fmt.Sprintf(":%d", p.Block.Line)
		mode := p.Block.Mode
		if p.Block.Untyped {
			mode = "untyped"
		}
		if pass {
			fmt.Printf("PASS %s (%s)\n", where, mode)
		} else {
			failed++
			fmt.Printf("FAIL %s (%s): %s\n", where, mode, why)
		}
		if *verbose || !pass {
			for _, l := range strings.Split(strings.TrimRight(r.Out, "\n"), "\n") {
				fmt.Printf("    | %s\n", l)
			}
		}
	}
	fmt.Printf("%d blocks, %d failed (%s)\n", len(progs), failed, name)
	if failed > 0 {
		return 1
	}
	return 0
}

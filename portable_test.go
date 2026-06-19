package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWrapExecutableWindows(t *testing.T) {
	cases := []struct {
		argv []string
		win  bool
		want []string
	}{
		{[]string{`C:\p\shen.cmd`, "eval"}, true, []string{"cmd", "/c", `C:\p\shen.cmd`, "eval"}},
		{[]string{`C:\p\x.bat`}, true, []string{"cmd", "/c", `C:\p\x.bat`}},
		{[]string{"builders/lisp/build.sh", "a"}, true, []string{"sh", "builders/lisp/build.sh", "a"}},
		{[]string{`C:\p\shen.exe`, "eval"}, true, []string{`C:\p\shen.exe`, "eval"}},
		{[]string{"/bin/shen", "eval"}, false, []string{"/bin/shen", "eval"}},
		{[]string{"x/build.sh"}, false, []string{"x/build.sh"}},
	}
	for _, c := range cases {
		got := wrapExecutableFor(c.argv, c.win)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("wrapExecutableFor(%v, win=%v) = %v, want %v", c.argv, c.win, got, c.want)
		}
	}
}

func TestFindExecutableWindowsExtension(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "shen.exe"), []byte("MZ"), 0o644)
	base := filepath.Join(dir, "shen")
	if got := findExecutableFor(base, true, []string{".exe", ".cmd"}); got != base+".exe" {
		t.Errorf("windows ext probe = %q, want %q", got, base+".exe")
	}
	if got := findExecutableFor(base, false, []string{".exe"}); got != "" {
		t.Errorf("posix must not invent .exe, got %q", got)
	}
}

func TestFindExecutableExactAndMissing(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "host")
	os.WriteFile(f, []byte("#!/bin/sh\n"), 0o755)
	if got := findExecutableFor(f, false, nil); got != f {
		t.Errorf("exact = %q, want %q", got, f)
	}
	if got := findExecutableFor(filepath.Join(dir, "nope"), true, []string{".exe"}); got != "" {
		t.Errorf("missing = %q, want empty", got)
	}
}

func TestUserConfigDir(t *testing.T) {
	win := userConfigDirFor("windows", func(k string) string {
		if k == "APPDATA" {
			return `C:\Users\me\AppData\Roaming`
		}
		return ""
	}, `C:\Users\me`)
	if !strings.HasPrefix(win, `C:\Users\me\AppData\Roaming`) || !strings.HasSuffix(win, "bifrost") {
		t.Errorf("windows config dir = %q", win)
	}
	winFallback := userConfigDirFor("windows", func(string) string { return "" }, "/home/me")
	if !strings.Contains(winFallback, "AppData") {
		t.Errorf("windows fallback = %q", winFallback)
	}
	lin := userConfigDirFor("linux", func(k string) string {
		if k == "XDG_CONFIG_HOME" {
			return "/x"
		}
		return ""
	}, "/home/me")
	if lin != filepath.Join("/x", "bifrost") {
		t.Errorf("linux XDG = %q", lin)
	}
	def := userConfigDirFor("darwin", func(string) string { return "" }, "/home/me")
	if def != filepath.Join("/home/me", ".config", "bifrost") {
		t.Errorf("posix default = %q", def)
	}
}

func TestOSOverridesMerge(t *testing.T) {
	raw := json.RawMessage(`{
		"eval": ["a"],
		"default_paths": ["p"],
		"os_overrides": {"win32": {"eval": ["b"], "launcher": ["luajit"]}}
	}`)
	a := &Adapters{raw: map[string]json.RawMessage{"x": raw}}
	win, err := a.effectiveForPlatform("x", "win32")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(win.Eval, []string{"b"}) {
		t.Errorf("eval override = %v, want [b]", win.Eval)
	}
	if !reflect.DeepEqual(win.Launcher, []string{"luajit"}) {
		t.Errorf("launcher added = %v, want [luajit]", win.Launcher)
	}
	if !reflect.DeepEqual(win.DefaultPaths, []string{"p"}) {
		t.Errorf("default_paths untouched = %v, want [p]", win.DefaultPaths)
	}
	lin, _ := a.effectiveForPlatform("x", "linux")
	if !reflect.DeepEqual(lin.Eval, []string{"a"}) {
		t.Errorf("other platform no override = %v, want [a]", lin.Eval)
	}
}

func TestNormalizeStripsChatter(t *testing.T) {
	in := "(fn foo)\nhello\nrun time: 0.1 secs\n0\n"
	if got := normalize(in, nil); got != "hello" {
		t.Errorf("normalize = %q, want %q", got, "hello")
	}
	// suite-supplied prefix
	if got := normalize("banner line\nreal\n", []string{"banner"}); got != "real" {
		t.Errorf("prefix strip = %q", got)
	}
	// trailing boolean echo (shen-lua)
	if got := normalize("4/4 passed\nALL PASS\ntrue\n", nil); got != "4/4 passed\nALL PASS" {
		t.Errorf("trailing echo = %q", got)
	}
}

func TestReorderArgs(t *testing.T) {
	// Flags placed after positionals must be pulled ahead (Go's flag package
	// stops at the first non-flag token), with value-flag values kept attached.
	got := reorderArgs([]string{"prog.shen", "--impl", "shen-go", "--raw"}, "impl", "e", "expr")
	want := []string{"--impl", "shen-go", "--raw", "prog.shen"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reorderArgs = %v, want %v", got, want)
	}
	// `--` ends flag processing; the rest are positional.
	got2 := reorderArgs([]string{"--impl", "x", "--", "--not-a-flag"}, "impl")
	want2 := []string{"--impl", "x", "--not-a-flag"}
	if !reflect.DeepEqual(got2, want2) {
		t.Errorf("reorderArgs(--) = %v, want %v", got2, want2)
	}
}

func TestRouterRecognisesSubcommands(t *testing.T) {
	for _, v := range []string{"run", "eval", "repl", "impls", "use", "install", "build"} {
		if !subcommands[v] {
			t.Errorf("subcommand %q not recognised", v)
		}
	}
	if subcommands["--list"] || subcommands["--suite"] {
		t.Error("test-matrix flags must not be treated as subcommands")
	}
}

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

func TestExitThreeIsCapabilitySkip(t *testing.T) {
	got := resultFromInvocation(
		runResult{Rc: 3, Out: "SKIP: cannot-reach=eval\n"}, nil)
	if got.Status != "SKIP" || got.Rc != 3 {
		t.Fatalf("exit 3 result = %#v, want capability SKIP", got)
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

func TestLauncherPrefixIsAppliedOnce(t *testing.T) {
	cfg := Adapter{Launcher: []string{"node"}}
	if got := launcherArgv(cfg, []string{"/tmp/shen.js", "eval"}); !reflect.DeepEqual(got, []string{"node", "/tmp/shen.js", "eval"}) {
		t.Fatalf("launcher prefix = %v", got)
	}
	if got := launcherArgv(cfg, []string{"node", "/tmp/shen.js"}); !reflect.DeepEqual(got, []string{"node", "/tmp/shen.js"}) {
		t.Fatalf("duplicate launcher prefix = %v", got)
	}
}

func TestShenTruffleAdapter(t *testing.T) {
	a, err := loadAdapters()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := a.effectiveForPlatform("shen-truffle", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "BIFROST_SHEN_TRUFFLE" || cfg.Kernel != "42" || cfg.Status != "production" {
		t.Fatalf("unexpected shen-truffle adapter: %#v", cfg)
	}
	got := buildArgv(Impl{Name: "shen-truffle", Cfg: cfg, Bin: "/tmp/shen-truffle"}, aCase{Mode: "eval", Expr: "(+ 1 2)"}, "")
	want := []string{"/tmp/shen-truffle", "eval", "-e", "(+ 1 2)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shen-truffle eval argv = %v, want %v", got, want)
	}
}

func TestDefaultSuiteCwdIsCheckout(t *testing.T) {
	s := defaultSuite()
	if s.DefaultCwd == "" {
		t.Fatal("default suite cwd must not be empty")
	}
	probe := filepath.Join(s.DefaultCwd, "programs", "load-echo-probe.shen")
	if _, err := os.Stat(probe); err != nil {
		t.Fatalf("load-echo probe missing at %s: %v", probe, err)
	}
	cases, err := s.loadCases(false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range cases {
		if c.Name == "load-toplevel-echo" {
			found = true
			if c.Cwd != "." {
				t.Fatalf("load-toplevel-echo cwd = %q, want .", c.Cwd)
			}
			if got := s.cwdFor(c); got != s.DefaultCwd && got != s.Root {
				absRoot, _ := filepath.Abs(s.Root)
				if got != absRoot {
					t.Fatalf("cwdFor(load-toplevel-echo) = %q, want checkout %q", got, s.DefaultCwd)
				}
			}
		}
	}
	if !found {
		t.Fatal("load-toplevel-echo missing from default suite")
	}
}

func TestShenCAdapter(t *testing.T) {
	a, err := loadAdapters()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := a.effectiveForPlatform("shen-c", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "BIFROST_SHEN_C" || cfg.Status != "experimental" || cfg.Kernel != "42" {
		t.Fatalf("unexpected shen-c adapter: %#v", cfg)
	}
	if len(cfg.DefaultPaths) != 1 || cfg.DefaultPaths[0] != "../shen-c/bin/shen-c" {
		t.Fatalf("shen-c default_paths = %v, want [../shen-c/bin/shen-c] only", cfg.DefaultPaths)
	}
	impl := Impl{Name: "shen-c", Cfg: cfg, Bin: "/tmp/shen-c"}
	evalGot := buildArgv(impl, aCase{Mode: "eval", Expr: "(+ 1 2)"}, "")
	evalWant := []string{"/tmp/shen-c", "eval", "-e", "(+ 1 2)"}
	if !reflect.DeepEqual(evalGot, evalWant) {
		t.Fatalf("shen-c eval argv = %v, want %v", evalGot, evalWant)
	}
	scriptGot := buildArgv(impl, aCase{Mode: "script", Program: "/tmp/p.shen"}, "")
	scriptWant := []string{"/tmp/shen-c", "script", "/tmp/p.shen"}
	if !reflect.DeepEqual(scriptGot, scriptWant) {
		t.Fatalf("shen-c script argv = %v, want %v", scriptGot, scriptWant)
	}
	versionGot := buildArgv(impl, aCase{Mode: "version"}, "")
	versionWant := []string{"/tmp/shen-c", "--version"}
	if !reflect.DeepEqual(versionGot, versionWant) {
		t.Fatalf("shen-c version argv = %v, want %v", versionGot, versionWant)
	}
	replGot := buildArgv(impl, aCase{Mode: "repl-eof"}, "")
	replWant := []string{"/tmp/shen-c"}
	if !reflect.DeepEqual(replGot, replWant) {
		t.Fatalf("shen-c repl argv = %v, want %v", replGot, replWant)
	}
	found := false
	for _, name := range implOrder {
		if name == "shen-c" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("implOrder omits shen-c")
	}
}

func TestAvailableForCaseKernelSplit(t *testing.T) {
	available := map[string]Impl{
		"shen-go": {Name: "shen-go", Cfg: Adapter{Kernel: "41.2"}},
		"shen-c":  {Name: "shen-c", Cfg: Adapter{Kernel: "42"}},
	}
	got := availableForCase(aCase{Kernels: []string{"41.2"}}, available)
	if len(got) != 1 || got["shen-go"].Name != "shen-go" {
		t.Fatalf("41.2 filter = %#v", got)
	}
	got42 := availableForCase(aCase{Kernels: []string{"42"}}, available)
	if len(got42) != 1 || got42["shen-c"].Name == "" {
		t.Fatalf("42 filter = %#v", got42)
	}
	if adapterKernel(Impl{Cfg: Adapter{}}) != "41.2" {
		t.Fatal("missing kernel should default to 41.2")
	}
}

func TestShenErlAdapter(t *testing.T) {
	a, err := loadAdapters()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := a.effectiveForPlatform("shen-erl", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Env != "BIFROST_SHEN_ERL" || cfg.Status != "production" {
		t.Fatalf("unexpected shen-erl adapter: %#v", cfg)
	}
	got := buildArgv(Impl{Name: "shen-erl", Cfg: cfg, Bin: "/tmp/shen-erl"}, aCase{Mode: "eval", Expr: "(+ 1 2)"}, "")
	want := []string{"/tmp/shen-erl", "eval", "-e", "(+ 1 2)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("shen-erl eval argv = %v, want %v", got, want)
	}
	if targetToImpl["erlang"] != "shen-erl" {
		t.Fatalf("erlang shake target maps to %q", targetToImpl["erlang"])
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
	for _, v := range []string{"run", "eval", "repl", "impls", "use", "env", "install", "build"} {
		if !subcommands[v] {
			t.Errorf("subcommand %q not recognised", v)
		}
	}
	if subcommands["--list"] || subcommands["--suite"] {
		t.Error("test-matrix flags must not be treated as subcommands")
	}
}

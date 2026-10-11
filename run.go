package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	timeoutDefault = 60 * time.Second
	timeoutHeavy   = 300 * time.Second
)

// pathext returns the lowercased Windows executable extensions.
func pathext() []string {
	raw := os.Getenv("PATHEXT")
	if raw == "" {
		raw = ".COM;.EXE;.BAT;.CMD"
	}
	var out []string
	for _, e := range strings.Split(raw, ";") {
		if strings.TrimSpace(e) != "" {
			out = append(out, strings.ToLower(e))
		}
	}
	return out
}

func isWindows() bool { return runtime.GOOS == "windows" }

// findExecutablePath returns an existing executable for path, or "".
// The path itself if it exists; otherwise, on Windows, path + each PATHEXT ext
// (default_paths are written POSIX-style; the real launcher is shen.exe/.cmd).
func findExecutablePath(path string) string {
	return findExecutableFor(path, isWindows(), pathext())
}

// findExecutableFor is the OS-parameterised core (tested on any OS).
func findExecutableFor(path string, windows bool, exts []string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	if windows {
		for _, ext := range exts {
			cand := path + ext
			if _, err := os.Stat(cand); err == nil {
				return cand
			}
		}
	}
	return ""
}

// wrapExecutable makes argv[0] runnable on this platform: on Windows a .bat/.cmd
// is wrapped in `cmd /c` and a .sh in `sh` (CreateProcess can't launch them
// directly). No-op elsewhere and for native/.exe launchers.
func wrapExecutable(argv []string) []string { return wrapExecutableFor(argv, isWindows()) }

// wrapExecutableFor is the OS-parameterised core (tested on any OS).
func wrapExecutableFor(argv []string, windows bool) []string {
	if windows && len(argv) > 0 {
		low := strings.ToLower(argv[0])
		switch {
		case strings.HasSuffix(low, ".bat"), strings.HasSuffix(low, ".cmd"):
			return append([]string{"cmd", "/c"}, argv...)
		case strings.HasSuffix(low, ".sh"):
			return append([]string{"sh"}, argv...)
		}
	}
	return argv
}

func subTokens(tok string, subs map[string]string) string {
	for k, v := range subs {
		tok = strings.ReplaceAll(tok, k, v)
	}
	return tok
}

// launcherArgv applies an adapter's optional command prefix (for example
// ["node"] for a JavaScript launcher). Templates may include the prefix for
// backwards compatibility; in that case it is not duplicated.
func launcherArgv(cfg Adapter, argv []string) []string {
	if len(cfg.Launcher) == 0 {
		return argv
	}
	if len(argv) >= len(cfg.Launcher) {
		match := true
		for i := range cfg.Launcher {
			if argv[i] != cfg.Launcher[i] {
				match = false
				break
			}
		}
		if match {
			return argv
		}
	}
	out := make([]string, 0, len(cfg.Launcher)+len(argv))
	out = append(out, cfg.Launcher...)
	return append(out, argv...)
}

// normalize strips launcher chatter so behaviour is compared, not chrome. A
// shared output normalizer (run-time banner, (fn ...) load
// echoes, shen-lua's trailing value echo, suite-supplied prefixes, blank runs).
func normalize(text string, extraPrefixes []string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	var out []string
	for _, ln := range lines {
		s := strings.TrimRight(ln, " \t")
		stripped := strings.TrimSpace(s)
		if strings.HasPrefix(stripped, "run time:") {
			// The kernel prints the banner with a leading newline, so drop
			// the blank line it produces too: a port that omits the banner
			// (e.g. shen-lua on a warm fasl-cache hit) must not differ by
			// that blank line alone.
			if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
				out = out[:len(out)-1]
			}
			continue
		}
		if isJVMOptionsNotice(stripped) {
			continue
		}
		if strings.HasPrefix(stripped, "(fn ") && strings.HasSuffix(stripped, ")") {
			continue
		}
		skip := false
		for _, p := range extraPrefixes {
			if strings.HasPrefix(stripped, p) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		out = append(out, s)
	}
	// drop leading/trailing blanks
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	// shen-lua's load echoes the final form's value after real output.
	if len(out) >= 2 {
		last := strings.TrimSpace(out[len(out)-1])
		if last == "0" || last == "true" || last == "false" || last == "nil" {
			out = out[:len(out)-1]
		}
	}
	// collapse interior blank runs
	var collapsed []string
	blank := false
	for _, s := range out {
		if strings.TrimSpace(s) == "" {
			if blank {
				continue
			}
			blank = true
		} else {
			blank = false
		}
		collapsed = append(collapsed, s)
	}
	return strings.Join(collapsed, "\n")
}

// isJVMOptionsNotice matches the line a JVM writes to stderr whenever
// JAVA_TOOL_OPTIONS, _JAVA_OPTIONS or JDK_JAVA_OPTIONS is set (agent proxies
// and CI images set them). It is host chrome, not Shen output, so a JVM-hosted
// port such as shen-truffle is compared without it.
func isJVMOptionsNotice(line string) bool {
	for _, v := range []string{"JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "JDK_JAVA_OPTIONS"} {
		if strings.HasPrefix(line, "Picked up "+v+":") || strings.HasPrefix(line, "NOTE: Picked up "+v+":") {
			return true
		}
	}
	return false
}

// capabilitySkipMarker is printed by a corpus program when the port does not
// boot an optional kernel facility (e.g. the namespaces extension). Shen has
// no portable exit primitive, so this line is the in-language equivalent of a
// launcher exiting 3: the (impl, case) cell becomes a capability SKIP.
const capabilitySkipMarker = "BIFROST-SKIP:"

func hasCapabilitySkip(norm string) bool {
	for _, ln := range strings.Split(norm, "\n") {
		if strings.HasPrefix(strings.TrimSpace(ln), capabilitySkipMarker) {
			return true
		}
	}
	return false
}

// maxCapture bounds what one invocation may buffer. A port whose REPL loops
// at stdin EOF (canonical Shen, upstream shen-scheme 0.50) prints megabytes
// per second until the timeout; keeping the head is enough to report it.
const maxCapture = 8 << 20

type cappedBuffer struct {
	buf       bytes.Buffer
	truncated bool
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	if room := maxCapture - c.buf.Len(); room > 0 {
		if len(p) > room {
			c.buf.Write(p[:room])
			c.truncated = true
		} else {
			c.buf.Write(p)
		}
	} else if len(p) > 0 {
		c.truncated = true
	}
	return len(p), nil
}

func (c *cappedBuffer) String() string {
	if c.truncated {
		return c.buf.String() + "\n[bifrost: output truncated]\n"
	}
	return c.buf.String()
}

// runResult captures the portable process outcome used by the matrix.
type runResult struct {
	Rc      int // -1 if unknown/timeout
	Out     string
	Timeout bool
}

// runInvocation runs argv, capturing combined stdout+stderr. cwd optional.
// stdinEOF feeds an immediately-closed stdin (the EOF probe); otherwise stdin
// is the null device.
func runInvocation(argv []string, timeout time.Duration, stdinEOF bool, cwd string) runResult {
	argv = wrapExecutable(argv)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if cwd != "" {
		cmd.Dir = cwd
	}
	if stdinEOF {
		cmd.Stdin = strings.NewReader("")
	} // else nil -> null device
	var buf cappedBuffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return runResult{Rc: -1, Out: buf.String(), Timeout: true}
	}
	rc := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			// launcher missing / spawn error
			return runResult{Rc: -1, Out: "launcher error: " + err.Error(), Timeout: false}
		}
	}
	return runResult{Rc: rc, Out: buf.String(), Timeout: false}
}

// aCase is a single corpus/suite case (the JSON shape in cases/*.json and suite
// manifests). Fields are a superset across modes.
type aCase struct {
	Name            string            `json:"name"`
	Mode            string            `json:"mode"`   // eval | script | version | repl-eof | special-hush
	Expect          string            `json:"expect"` // output | agreement | marker | version | clean-exit | divergence-table | yggdrasil-parity
	Expr            string            `json:"expr"`
	Program         string            `json:"program"`
	Golden          string            `json:"golden"`
	Marker          string            `json:"marker"`
	Contains        string            `json:"contains"`
	Doc             string            `json:"doc"`
	KnownDivergence string            `json:"known_divergence"`
	Heavy           bool              `json:"heavy"`
	Cwd             string            `json:"cwd"`
	Table           map[string]string `json:"table"`
	ShakeFiles      string            `json:"shake_files"`
	Group           string            `json:"group"`
	Kernels         []string          `json:"kernels"`
}

// buildArgv builds the argv for (impl, case) per its mode.
func buildArgv(impl Impl, c aCase, programsDir string) []string {
	subs := map[string]string{"{bin}": impl.Bin}
	var tmpl []string
	switch c.Mode {
	case "eval":
		subs["{expr}"] = c.Expr
		tmpl = impl.Cfg.Eval
	case "script":
		prog := c.Program
		if !filepath.IsAbs(prog) {
			prog = filepath.Join(programsDir, prog)
		}
		abs, _ := filepath.Abs(prog)
		subs["{file}"] = abs
		tmpl = impl.Cfg.Script
	case "version":
		tmpl = impl.Cfg.Version
		if tmpl == nil {
			subs["{expr}"] = "(version)"
			tmpl = impl.Cfg.Eval
		}
	case "repl-eof":
		tmpl = impl.Cfg.Repl
		if tmpl == nil {
			tmpl = impl.Cfg.Eval
		}
	default:
		tmpl = impl.Cfg.Eval
	}
	out := make([]string, len(tmpl))
	for i, t := range tmpl {
		out[i] = subTokens(t, subs)
	}
	return launcherArgv(impl.Cfg, out)
}

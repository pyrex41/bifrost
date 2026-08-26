package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Each Yggdrasil build target maps onto the impl column it runs on, so shake
// results line up with the normal matrix. All Bifrost ports have a
// Yggdrasil builder.
//
// The `julia` target is special: shen-julia's stand-alone artifact AOT-compiles
// the shaken kernel+user KL into baked Julia methods and (per builders.json)
// bakes a per-program sysimage via PackageCompiler — the Julia analogue of the
// Lisp saved image / Go-Rust native binary. That bake costs minutes, so the
// shake timeout below is generous. The `scheme` target compiles the shaken slice
// with shen-scheme's own kl->scheme into a self-contained Chez artifact; the
// `swift` target drives the shen-swift tree-walking interpreter on the slice
// (boot a ~200-line shaken kernel instead of the full ~2500-line kernel).
var targetToImpl = map[string]string{
	"lisp": "shen-cl", "lua": "shen-lua", "go": "shen-go",
	"erlang": "shen-erl", "rust": "shen-rust", "js": "ShenScript", "julia": "shen-julia",
	"scheme": "shen-scheme", "swift": "shen-swift",
	"truffle": "shen-truffle",
}

const shakeTargetTimeout = 20 * time.Minute // accommodates the julia sysimage bake

// resolveYggdrasil returns the argv prefix to invoke the Go yggdrasil CLI, or
// nil. Order: `yggdrasil` on PATH (the installed Go binary) -> $YGGDRASIL_BIN
// -> a sibling dev build (./.bin or $BIFROST_YGGDRASIL_DIR). The Python wrapper
// is no longer used — bifrost and yggdrasil are both Go.
func resolveYggdrasil() []string {
	if p, err := exec.LookPath("yggdrasil"); err == nil {
		return []string{p}
	}
	var cands []string
	if v := os.Getenv("YGGDRASIL_BIN"); v != "" {
		cands = append(cands, v)
	}
	if d := os.Getenv("BIFROST_YGGDRASIL_DIR"); d != "" {
		cands = append(cands, filepath.Join(d, ".bin", "yggdrasil"), filepath.Join(d, "yggdrasil"))
	}
	cwd, _ := os.Getwd()
	cands = append(cands,
		filepath.Join(cwd, "..", "yggdrasil", ".bin", "yggdrasil"),
		filepath.Join(cwd, "..", "yggdrasil", "yggdrasil"))
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return []string{abs}
		}
	}
	return nil
}

// runShakeCase builds a standalone artifact for each target via the Go yggdrasil
// CLI, runs it, and diffs the artifact outputs (the real stand-alone deploy
// path). Mirrors evaluateCase's return shape so it drops into the matrix.
func runShakeCase(c aCase, available map[string]Impl, s suite, programsDir string) caseResult {
	results := map[string]*implResult{}
	for n := range available {
		results[n] = &implResult{Status: "SKIP"}
	}
	ygg := resolveYggdrasil()
	if ygg == nil {
		return caseResult{PerImpl: results, Verdict: "PASS",
			Detail: "SKIPPED: yggdrasil CLI not found (go install github.com/pyrex41/yggdrasil@latest, or set $YGGDRASIL_BIN / $BIFROST_YGGDRASIL_DIR)"}
	}
	prog := c.Program
	if !filepath.IsAbs(prog) {
		prog = filepath.Join(programsDir, prog)
	}
	prog, _ = filepath.Abs(prog)
	tmp, _ := os.MkdirTemp("", "bifrost_goshake_")

	var targets []string
	for t := range targetToImpl {
		targets = append(targets, t)
	}
	sort.Strings(targets)

	for _, target := range targets {
		impl := targetToImpl[target]
		if _, ok := available[impl]; !ok {
			continue // leaves the column SKIP
		}
		r := results[impl]
		out := filepath.Join(tmp, target)
		argv := append(append([]string{}, ygg...), "run", prog, out, "--target", target)
		ctx, cancel := context.WithTimeout(context.Background(), shakeTargetTimeout)
		var buf bytes.Buffer
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Stdout = &buf      // pure artifact stdout (build chatter is on stderr)
		cmd.Stderr = os.Stderr // surface build progress live
		err := cmd.Run()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			r.Status, r.Norm, r.Timeout = "FAIL", "timeout", true
			continue
		}
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 3 {
				r.Status, r.Norm = "SKIP", "no-tool" // target toolchain missing
				continue
			}
			r.Status, r.Norm = "FAIL", "build/run-failed"
			continue
		}
		r.Norm, r.Status = normalize(buf.String(), s.ExtraPrefixes), "PASS"
	}

	// Decide the verdict over targets that actually ran.
	ran := map[string]*implResult{}
	for n, r := range results {
		if r.Status == "PASS" || r.Status == "FAIL" {
			ran[n] = r
		}
	}
	if len(ran) == 0 {
		return caseResult{PerImpl: results, Verdict: "PASS", Detail: "SKIPPED: no buildable targets available"}
	}
	for _, n := range sortedKeys(ran) {
		if ran[n].Status == "FAIL" {
			var bad []string
			for _, m := range sortedKeys(ran) {
				if ran[m].Status == "FAIL" {
					bad = append(bad, fmt.Sprintf("%s(%s)", m, ran[m].Norm))
				}
			}
			return caseResult{PerImpl: results, Verdict: "FAIL", Detail: "artifact build/run failed: " + strings.Join(bad, ", ")}
		}
	}
	norms := map[string]string{}
	for n, r := range ran {
		norms[n] = r.Norm
	}
	if len(distinctValues(norms)) > 1 {
		setAll(ran, "FAIL")
		return caseResult{PerImpl: results, Verdict: "FAIL", Detail: "shaken artifacts DISAGREE: " + bucketDetail(norms)}
	}
	var one string
	for _, v := range norms {
		one = v
		break
	}
	if c.Marker != "" && !strings.Contains(one, c.Marker) {
		setAll(ran, "FAIL")
		return caseResult{PerImpl: results, Verdict: "FAIL", Detail: fmt.Sprintf("shaken artifacts agree but marker %q absent", c.Marker)}
	}
	return caseResult{PerImpl: results, Verdict: "PASS",
		Detail: fmt.Sprintf("shaken artifacts BYTE-IDENTICAL across %d targets (%s)", len(ran), strings.Join(sortedKeys(ran), ", "))}
}

// yggdrasilRepoDir resolves the Yggdrasil repo (for tests/fib.shen etc.):
// $BIFROST_YGGDRASIL_DIR, else a sibling ../yggdrasil.
func yggdrasilRepoDir() string {
	if d := os.Getenv("BIFROST_YGGDRASIL_DIR"); d != "" {
		return d
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "..", "yggdrasil")
}

// runYggdrasilParity shakes the program on EACH available host (via the Go
// yggdrasil CLI's --host) and asserts the produced kernel.kl + manifest are
// byte-identical across hosts (the user KL differs only by gensym counter, so
// it is not compared). The Go analogue of the Python parity runner.
func runYggdrasilParity(c aCase, available map[string]Impl) caseResult {
	results := map[string]*implResult{}
	for n := range available {
		results[n] = &implResult{Status: "SKIP"}
	}
	ygg := resolveYggdrasil()
	repo := yggdrasilRepoDir()
	// shake_files is a JSON array literal like ["tests/fib.shen"]; take the first.
	rel := "tests/fib.shen"
	if c.ShakeFiles != "" {
		s := strings.Trim(c.ShakeFiles, "[]")
		if i := strings.IndexByte(s, '"'); i >= 0 {
			s = s[i+1:]
			if j := strings.IndexByte(s, '"'); j >= 0 {
				rel = s[:j]
			}
		}
	}
	prog := filepath.Join(repo, rel)
	if ygg == nil || !fileExists(filepath.Join(repo, "yggdrasil.shen")) || !fileExists(prog) {
		return caseResult{PerImpl: results, Verdict: "PASS",
			Detail: "SKIPPED: yggdrasil CLI or repo not found (set $BIFROST_YGGDRASIL_DIR / $YGGDRASIL_BIN)"}
	}
	tmp, _ := os.MkdirTemp("", "bifrost_goparity_")
	type dig struct{ kernel, manifest string }
	digs := map[string]dig{}
	for _, name := range sortedImpls(available) {
		// shen-julia hosts the shake correctly but, with no sysimage, loading
		// yggdrasil + walking the callgraph in the interpreter takes minutes —
		// impractical for this gate. Skip it as a parity HOST (it remains a
		// first-class shake TARGET). Set BIFROST_PARITY_JULIA=1 to include it.
		if name == "shen-julia" && os.Getenv("BIFROST_PARITY_JULIA") == "" {
			results[name].Status = "SKIP"
			results[name].Norm = "host too slow (no sysimage); target only"
			continue
		}
		im := available[name]
		out := filepath.Join(tmp, name)
		launcher := append(append([]string{}, im.Cfg.Launcher...), im.Bin)
		argv := append(append([]string{}, ygg...),
			"shake", prog, out, "--host", strings.Join(launcher, " "))
		if name == "shen-lua" {
			argv = append(argv, "--eval-style", "positional")
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeoutHeavy)
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		cancel()
		r := results[name]
		kpath, mpath := filepath.Join(out, "kernel.kl"), filepath.Join(out, "yggdrasil.manifest")
		if err != nil || !fileNonEmpty(kpath) || !fileNonEmpty(mpath) {
			r.Status, r.Norm = "FAIL", "no/empty artefacts"
			continue
		}
		kd, md := md5File(kpath), md5File(mpath)
		digs[name] = dig{kd, md}
		r.Status, r.Norm = "PASS", kd[:8]+"/"+md[:8]
	}
	if len(digs) == 0 {
		return caseResult{PerImpl: results, Verdict: "FAIL", Detail: "no host produced stage-1 artefacts"}
	}
	// Any FAIL among the impls that ran?
	for _, n := range sortedKeys(results) {
		if results[n].Status == "FAIL" {
			return caseResult{PerImpl: results, Verdict: "FAIL", Detail: "some hosts produced no/empty stage-1 artefacts"}
		}
	}
	distinct := map[dig]bool{}
	for _, d := range digs {
		distinct[d] = true
	}
	if len(distinct) > 1 {
		setAll(results, "FAIL")
		var parts []string
		for _, n := range sortedKeys2map(digs) {
			parts = append(parts, fmt.Sprintf("%s=%s/%s", n, digs[n].kernel[:8], digs[n].manifest[:8]))
		}
		return caseResult{PerImpl: results, Verdict: "FAIL", Detail: "kernel.kl/manifest digests disagree: " + strings.Join(parts, "; ")}
	}
	var sample dig
	for _, d := range digs {
		sample = d
		break
	}
	return caseResult{PerImpl: results, Verdict: "PASS",
		Detail: fmt.Sprintf("kernel.kl + yggdrasil.manifest BYTE-IDENTICAL across %d hosts (kernel md5 %s, manifest md5 %s)", len(digs), sample.kernel[:8], sample.manifest[:8])}
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }
func fileNonEmpty(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.Size() > 0
}
func md5File(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	sum := md5.Sum(b)
	return hex.EncodeToString(sum[:])
}
func sortedImpls(m map[string]Impl) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
func sortedKeys2map[V any](m map[string]V) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

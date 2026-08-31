package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

// suite bundles the cases to run with where their programs live and the cwd to
// run from. The built-in suite is Bifrost's own corpus (embedded); a project
// supplies its own via a bifrost.suite.json manifest.
type suite struct {
	Name          string
	ProgramsDir   string // "" => embedded corpus
	CasesDir      string // "" => embedded corpus
	InlineCases   []aCase
	Root          string
	DefaultCwd    string
	ExtraPrefixes []string
	AdaptersPath  string
	useEmbedded   bool
}

func defaultSuite() suite {
	root := bifrostCheckoutRoot()
	return suite{Name: "bifrost", useEmbedded: true, Root: root, DefaultCwd: root}
}

// bifrostCheckoutRoot is the directory that contains programs/load-echo-probe.shen
// so eval (load "programs/...") is cwd-stable when the binary is launched from
// a parent directory. Falls back to process cwd.
func bifrostCheckoutRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = ""
	}
	dir := cwd
	for i := 0; i < 6 && dir != ""; i++ {
		if _, err := os.Stat(filepath.Join(dir, "programs", "load-echo-probe.shen")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		src := filepath.Dir(thisFile)
		if _, err := os.Stat(filepath.Join(src, "programs", "load-echo-probe.shen")); err == nil {
			return src
		}
	}
	return cwd
}

type manifest struct {
	Name              string   `json:"name"`
	Root              string   `json:"root"`
	ProgramsDir       string   `json:"programs_dir"`
	CasesDir          string   `json:"cases_dir"`
	Cases             []aCase  `json:"cases"`
	DefaultCwd        string   `json:"default_cwd"`
	StripLinePrefixes []string `json:"strip_line_prefixes"`
	Adapters          string   `json:"adapters"`
}

func buildSuiteFromManifest(path string) (suite, error) {
	abs, _ := filepath.Abs(path)
	base := filepath.Dir(abs)
	b, err := os.ReadFile(abs)
	if err != nil {
		return suite{}, err
	}
	var m manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return suite{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	resolve := func(rel, def string) string {
		if rel == "" {
			return def
		}
		if filepath.IsAbs(rel) {
			return rel
		}
		return filepath.Join(base, rel)
	}
	root := resolve(m.Root, base)
	s := suite{
		Name:          orDefault(m.Name, filepath.Base(base)),
		ProgramsDir:   resolve(m.ProgramsDir, root),
		Root:          root,
		DefaultCwd:    resolve(m.DefaultCwd, root),
		ExtraPrefixes: m.StripLinePrefixes,
		InlineCases:   m.Cases,
	}
	if m.CasesDir != "" {
		s.CasesDir = resolve(m.CasesDir, "")
	}
	if m.Adapters != "" {
		s.AdaptersPath = resolve(m.Adapters, "")
	}
	return s, nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// loadCases collects cases for a run: inline (preferred), else a cases dir, else
// the embedded corpus. Heavy cases are dropped unless includeHeavy.
func (s suite) loadCases(includeHeavy bool) ([]aCase, error) {
	var raw []aCase
	switch {
	case s.InlineCases != nil:
		for _, c := range s.InlineCases {
			if c.Group == "" {
				c.Group = "suite"
			}
			raw = append(raw, c)
		}
	case s.CasesDir != "":
		cs, err := casesFromDir(os.DirFS(s.CasesDir))
		if err != nil {
			return nil, err
		}
		raw = cs
	default:
		cs, err := casesFromDir(mustSub(embeddedCases, "cases"))
		if err != nil {
			return nil, err
		}
		raw = cs
	}
	var out []aCase
	for _, c := range raw {
		if c.Heavy && !includeHeavy {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// programsDirFor returns the dir script-mode {file} programs resolve against,
// materialising the embedded corpus to a temp dir when needed.
func (s suite) programsDirFor() (string, error) {
	if s.ProgramsDir != "" {
		return s.ProgramsDir, nil
	}
	return materializeEmbedded(embeddedPrograms, "programs")
}

// cwdFor resolves the launcher cwd for a case (per-case > suite default), with
// relative paths against the suite root.
func (s suite) cwdFor(c aCase) string {
	cwd := c.Cwd
	if cwd == "" {
		cwd = s.DefaultCwd
	}
	if cwd == "" {
		return ""
	}
	if !filepath.IsAbs(cwd) {
		cwd = filepath.Join(orDefault(s.Root, "."), cwd)
	}
	abs, _ := filepath.Abs(cwd)
	return abs
}

type implResult struct {
	Raw     string
	Norm    string
	Status  string // PASS | FAIL | DIVERGE | SKIP
	Timeout bool
	Rc      int
	Ok      bool // for version/clean-exit/marker per-impl checks
}

type caseResult struct {
	PerImpl map[string]*implResult
	Verdict string
	Detail  string
}

// runCaseOnImpl runs one (impl, case) and returns its per-impl result.
func runCaseOnImpl(im Impl, c aCase, s suite, programsDir string, heavyTimeout time.Duration) *implResult {
	to := timeoutDefault
	if c.Heavy {
		to = heavyTimeout
	}
	cwd := s.cwdFor(c)
	switch c.Mode {
	case "version":
		r := runInvocation(buildArgv(im, c, programsDir), to, false, cwd)
		ok := !r.Timeout && strings.Contains(r.Out, c.Contains)
		return &implResult{Raw: r.Out, Norm: normalize(r.Out, s.ExtraPrefixes), Ok: ok, Timeout: r.Timeout, Rc: r.Rc}
	case "repl-eof":
		r := runInvocation(buildArgv(im, c, programsDir), to, true, cwd)
		return &implResult{Raw: r.Out, Norm: normalize(r.Out, s.ExtraPrefixes), Ok: !r.Timeout, Timeout: r.Timeout, Rc: r.Rc}
	default: // eval / script
		r := runInvocation(buildArgv(im, c, programsDir), to, false, cwd)
		return resultFromInvocation(r, s.ExtraPrefixes)
	}
}

func resultFromInvocation(r runResult, extraPrefixes []string) *implResult {
	result := &implResult{
		Raw: r.Out, Norm: normalize(r.Out, extraPrefixes),
		Timeout: r.Timeout, Rc: r.Rc,
	}
	if r.Rc == 3 {
		result.Status = "SKIP"
	}
	return result
}

func adapterKernel(im Impl) string {
	if im.Cfg.Kernel != "" {
		return im.Cfg.Kernel
	}
	return "41.2"
}

func availableForCase(c aCase, available map[string]Impl) map[string]Impl {
	if len(c.Kernels) == 0 {
		return available
	}
	want := map[string]bool{}
	for _, k := range c.Kernels {
		want[k] = true
	}
	out := map[string]Impl{}
	for n, im := range available {
		if want[adapterKernel(im)] {
			out[n] = im
		}
	}
	return out
}

// evaluateCase runs a case across impls and decides the verdict.
func evaluateCase(c aCase, available map[string]Impl, s suite, programsDir string, heavyTimeout time.Duration) caseResult {
	available = availableForCase(c, available)
	results := map[string]*implResult{}
	if len(available) == 0 {
		return caseResult{PerImpl: results, Verdict: "PASS", Detail: "skipped: no matching kernels"}
	}
	for name, im := range available {
		results[name] = runCaseOnImpl(im, c, s, programsDir, heavyTimeout)
	}
	res := caseResult{PerImpl: results, Verdict: "PASS"}
	active := map[string]*implResult{}
	for name, result := range results {
		if result.Status != "SKIP" {
			active[name] = result
		}
	}
	if len(active) == 0 {
		res.Detail = "SKIPPED: every implementation reported an unavailable capability (exit 3)"
		return res
	}

	markerCheck := func() (bool, []string) {
		var missing []string
		for n, r := range active {
			if r.Timeout || (c.Marker != "" && !strings.Contains(r.Norm, c.Marker)) {
				missing = append(missing, n)
			}
		}
		sort.Strings(missing)
		return len(missing) == 0, missing
	}

	switch c.Expect {
	case "marker":
		ok, missing := markerCheck()
		for n, r := range active {
			if contains(missing, n) {
				r.Status = "FAIL"
			} else {
				r.Status = "PASS"
			}
		}
		if !ok {
			res.Verdict = "FAIL"
			res.Detail = fmt.Sprintf("marker %q missing on: %s", c.Marker, strings.Join(missing, ", "))
		}
	case "output":
		for _, r := range active {
			if r.Timeout {
				r.Status = "FAIL"
				res.Verdict = "FAIL"
			} else if r.Norm == c.Golden {
				r.Status = "PASS"
			} else {
				r.Status = "FAIL"
				res.Verdict = "FAIL"
			}
		}
		if res.Verdict == "FAIL" {
			var parts []string
			for _, n := range sortedKeys(active) {
				if active[n].Status == "FAIL" {
					parts = append(parts, fmt.Sprintf("%s=%q", n, active[n].Norm))
				}
			}
			res.Detail = fmt.Sprintf("golden=%q; mismatches: %s", c.Golden, strings.Join(parts, ", "))
		}
	case "agreement":
		var timed []string
		norms := map[string]string{}
		for _, n := range sortedKeys(active) {
			norms[n] = active[n].Norm
			if active[n].Timeout {
				timed = append(timed, n)
			}
		}
		distinct := distinctValues(norms)
		isDiv := c.KnownDivergence != ""
		switch {
		case len(timed) > 0:
			res.Verdict = "FAIL"
			for _, r := range active {
				if r.Timeout {
					r.Status = "FAIL"
				} else {
					r.Status = "PASS"
				}
			}
			res.Detail = "timeout on: " + strings.Join(timed, ", ")
		case len(distinct) <= 1:
			if c.Marker != "" {
				ok, _ := markerCheck()
				if ok {
					setAll(active, "PASS")
				} else {
					setAll(active, "FAIL")
					res.Verdict = "FAIL"
					res.Detail = fmt.Sprintf("ports AGREE but marker %q absent (suite did not report success on any port)", c.Marker)
				}
			} else {
				setAll(active, "PASS")
			}
		default:
			if isDiv {
				res.Verdict = "DIVERGE"
				setAll(active, "DIVERGE")
			} else {
				res.Verdict = "FAIL"
				setAll(active, "FAIL")
			}
			res.Detail = bucketDetail(norms)
			if isDiv {
				res.Detail = fmt.Sprintf("KNOWN DIVERGENCE [%s]: %s", c.KnownDivergence, res.Detail)
			}
		}
	case "version":
		for _, r := range active {
			if r.Ok {
				r.Status = "PASS"
			} else {
				r.Status = "FAIL"
				res.Verdict = "FAIL"
			}
		}
		if res.Verdict == "FAIL" {
			res.Detail = fmt.Sprintf("version must contain %q", c.Contains)
		}
	case "clean-exit":
		for _, r := range active {
			if r.Ok {
				r.Status = "PASS"
			} else {
				r.Status = "FAIL"
				res.Verdict = "FAIL"
			}
		}
		if res.Verdict == "FAIL" {
			res.Detail = "REPL did not exit cleanly on EOF (hang/timeout)"
		}
	case "divergence-table":
		res.Verdict = "DIVERGE"
		res.Detail = c.Doc
		setAll(active, "DIVERGE")
	default:
		res.Verdict = "FAIL"
		res.Detail = "unknown expect " + c.Expect
		setAll(active, "FAIL")
	}
	return res
}

func setAll(m map[string]*implResult, status string) {
	for _, r := range m {
		r.Status = status
	}
}

func bucketDetail(norms map[string]string) string {
	buckets := map[string][]string{}
	for n, v := range norms {
		buckets[v] = append(buckets[v], n)
	}
	var parts []string
	for _, v := range sortedKeys2(buckets) {
		ns := buckets[v]
		sort.Strings(ns)
		parts = append(parts, fmt.Sprintf("%q <- %s", v, strings.Join(ns, ",")))
	}
	return strings.Join(parts, "; ")
}

func distinctValues(m map[string]string) map[string]bool {
	d := map[string]bool{}
	for _, v := range m {
		d[v] = true
	}
	return d
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]*implResult) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func sortedKeys2(m map[string][]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// runHushDivergence drives the -q/*hush* file-write behaviour per impl.
func runHushDivergence(c aCase, available map[string]Impl) caseResult {
	isDiv := c.KnownDivergence != ""
	table := c.Table
	results := map[string]*implResult{}
	tmpdir, _ := os.MkdirTemp("", "bifrost_hush_")
	for name, im := range available {
		outfile := filepath.Join(tmpdir, name+".out")
		os.Remove(outfile)
		prog := filepath.Join(tmpdir, name+"_write.shen")
		os.WriteFile(prog, []byte(fmt.Sprintf(`(let S (open "%s" out) (do (pr "hello" S) (close S)))`+"\n", outfile)), 0o644)
		argv := buildHushArgv(name, im, prog)
		runInvocation(argv, timeoutDefault, false, "")
		size := int64(0)
		if fi, err := os.Stat(outfile); err == nil {
			size = fi.Size()
		}
		observed := "empty"
		if size > 0 {
			observed = "written"
		}
		results[name] = &implResult{Raw: fmt.Sprintf("%d bytes (%s)", size, observed), Norm: observed}
	}
	buckets := map[string][]string{}
	for n, r := range results {
		buckets[r.Norm] = append(buckets[r.Norm], n)
	}
	if isDiv {
		verdict := "DIVERGE"
		for name, r := range results {
			exp, has := table[name]
			match := !has || r.Norm == exp
			if match {
				r.Status = "DIVERGE"
			} else {
				r.Status = "FAIL"
				verdict = "FAIL"
			}
		}
		return caseResult{PerImpl: results, Verdict: verdict, Detail: "KNOWN DIVERGENCE [" + c.KnownDivergence + "]: " + bucketJoin(buckets)}
	}
	agree := len(buckets) <= 1
	status := "FAIL"
	verdict := "FAIL"
	if agree {
		status, verdict = "PASS", "PASS"
	}
	setAll(results, status)
	detail := bucketJoin(buckets)
	if !agree {
		detail = "disagreement under -q/*hush* file write: " + detail
	}
	return caseResult{PerImpl: results, Verdict: verdict, Detail: detail}
}

func buildHushArgv(name string, im Impl, prog string) []string {
	if name == "shen-lua" {
		return []string{im.Bin, "-q", prog}
	}
	return launcherArgv(im.Cfg, []string{im.Bin, "eval", "-q", "-l", prog})
}

func bucketJoin(buckets map[string][]string) string {
	var parts []string
	for _, b := range sortedKeys2(buckets) {
		ns := buckets[b]
		sort.Strings(ns)
		parts = append(parts, fmt.Sprintf("%s <- %s", b, strings.Join(ns, ",")))
	}
	return strings.Join(parts, "; ")
}

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type caseFile struct {
	Group string  `json:"group"`
	Cases []aCase `json:"cases"`
}

func mustSub(efs fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(efs, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// casesFromDir reads *.json corpus files ({group, cases:[...]}) from a FS.
func casesFromDir(fsys fs.FS) ([]aCase, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	var out []aCase
	for _, fn := range names {
		b, err := fs.ReadFile(fsys, fn)
		if err != nil {
			return nil, err
		}
		var blob caseFile
		if err := json.Unmarshal(b, &blob); err != nil {
			return nil, fmt.Errorf("%s: %w", fn, err)
		}
		group := blob.Group
		if group == "" {
			group = strings.TrimSuffix(fn, ".json")
		}
		for _, c := range blob.Cases {
			if c.Group == "" {
				c.Group = group
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// materializeEmbedded copies an embedded dir to a temp dir, returning its path.
func materializeEmbedded(efs fs.FS, dir string) (string, error) {
	root, err := os.MkdirTemp("", "bifrost_"+dir+"_")
	if err != nil {
		return "", err
	}
	sub := mustSub(efs, dir)
	err = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(root, p), 0o755)
		}
		b, err := fs.ReadFile(sub, p)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, p), b, 0o644)
	})
	if err != nil {
		return "", err
	}
	return root, nil
}

// executeCases runs every case across the available impls.
func executeCases(cases []aCase, available map[string]Impl, s suite, programsDir string, shake bool) map[string]caseResult {
	out := map[string]caseResult{}
	for _, c := range cases {
		fmt.Fprintf(os.Stderr, "running %-28s ... ", c.Name)
		t0 := time.Now()
		var res caseResult
		switch {
		case c.Mode == "special-hush":
			res = runHushDivergence(c, available)
		case c.Expect == "ratatoskr-parity":
			res = runRatatoskrParity(c, available)
		case shake && c.Mode == "script":
			res = runShakeCase(c, available, s, programsDir)
		default:
			res = evaluateCase(c, available, s, programsDir, timeoutHeavy)
		}
		out[c.Name] = res
		fmt.Fprintf(os.Stderr, "%s (%.1fs)\n", res.Verdict, time.Since(t0).Seconds())
	}
	return out
}

func skipNote(available map[string]Impl, detail string) caseResult {
	results := map[string]*implResult{}
	for n := range available {
		results[n] = &implResult{Status: "SKIP"}
	}
	return caseResult{PerImpl: results, Verdict: "PASS", Detail: "SKIPPED: " + detail}
}

var sym = map[string]string{"PASS": "PASS", "FAIL": "FAIL", "DIVERGE": "DVRG", "SKIP": "----"}

func printMatrix(cases []aCase, results map[string]caseResult, available map[string]Impl, skipped map[string]string) int {
	var impls []string
	for _, n := range implOrder {
		if _, ok := available[n]; ok {
			impls = append(impls, n)
		}
	}
	namew := len("case")
	for _, c := range cases {
		if len(c.Name) > namew {
			namew = len(c.Name)
		}
	}
	namew += 2
	colw := 10
	for _, i := range impls {
		if len(i)+1 > colw {
			colw = len(i) + 1
		}
	}
	pad := func(s string, w int) string {
		if len(s) >= w {
			return s
		}
		return s + strings.Repeat(" ", w-len(s))
	}
	fmt.Println("\n" + strings.Repeat("=", 78))
	fmt.Println("BIFROST MATRIX  (impl x case)")
	fmt.Println(strings.Repeat("=", 78))
	header := pad("case", namew)
	for _, i := range impls {
		header += pad(i, colw)
	}
	header += "verdict"
	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))
	for _, c := range cases {
		res := results[c.Name]
		row := pad(c.Name, namew)
		for _, i := range impls {
			st := "SKIP"
			if r, ok := res.PerImpl[i]; ok && r.Status != "" {
				st = r.Status
			}
			row += pad(sym[st], colw)
		}
		row += res.Verdict
		fmt.Println(row)
	}
	fmt.Println(strings.Repeat("-", len(header)))

	if len(skipped) > 0 {
		fmt.Println("\nSKIPPED IMPLEMENTATIONS (launcher not found; reported, not failed):")
		for _, n := range sortedSkip(skipped) {
			fmt.Printf("  - %s: %s\n", n, skipped[n])
		}
	}
	var divs, fails []string
	for _, c := range cases {
		switch results[c.Name].Verdict {
		case "DIVERGE":
			divs = append(divs, c.Name)
		case "FAIL":
			fails = append(fails, c.Name)
		}
	}
	if len(divs) > 0 {
		fmt.Println("\nKNOWN DIVERGENCES (documented differences; do NOT fail the run):")
		for _, n := range divs {
			fmt.Printf("  * %s: %s\n", n, results[n].Detail)
		}
	}
	if len(fails) > 0 {
		fmt.Println("\nFAILURES:")
		for _, n := range fails {
			fmt.Printf("  ! %s: %s\n", n, results[n].Detail)
		}
	}
	npass := 0
	for _, c := range cases {
		if results[c.Name].Verdict == "PASS" {
			npass++
		}
	}
	fmt.Printf("\nSUMMARY: %d pass, %d diverge (known), %d FAIL  across %d impls\n", npass, len(divs), len(fails), len(impls))
	fmt.Println(strings.Repeat("=", 78))
	return len(fails)
}

func sortedSkip(m map[string]string) []string {
	var ks []string
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

// runMatrix is the default (no-subcommand) path: the differential test matrix.
func runMatrix(args []string) int {
	fs := newFlagSet("bifrost")
	heavy := fs.Bool("heavy", false, "include heavy (ratatoskr) cases")
	only := fs.String("only", "", "run only these cases (comma-separated)")
	implsFlag := fs.String("impls", "", "comma-separated subset of impls to drive")
	list := fs.Bool("list", false, "list impls + cases and exit")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON")
	suitePath := fs.String("suite", "", "run a project's bifrost.suite.json instead of the corpus")
	shake := fs.Bool("shake", false, "diff standalone artifacts per target (deploy-path parity)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	s := defaultSuite()
	if *suitePath != "" {
		var err error
		s, err = buildSuiteFromManifest(*suitePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "bifrost:", err)
			return 2
		}
	}
	if *suitePath != "" {
		if s.AdaptersPath != "" {
			os.Setenv("BIFROST_ADAPTERS", s.AdaptersPath)
		}
	}
	a, err := loadAdapters()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	var restrict map[string]bool
	if *implsFlag != "" {
		restrict = map[string]bool{}
		for _, n := range strings.Split(*implsFlag, ",") {
			restrict[strings.TrimSpace(n)] = true
		}
	}
	available, skipped := a.discover(restrict)

	cases, err := s.loadCases(*heavy)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	if *only != "" {
		want := map[string]bool{}
		for _, n := range strings.Split(*only, ",") {
			want[strings.TrimSpace(n)] = true
		}
		var filtered []aCase
		for _, c := range cases {
			if want[c.Name] {
				filtered = append(filtered, c)
			}
		}
		cases = filtered
	}

	if *list {
		if *suitePath != "" {
			fmt.Printf("Suite: %s  (root %s)\n\n", s.Name, s.Root)
		}
		fmt.Println("Implementations:")
		for _, n := range a.names() {
			if im, ok := available[n]; ok {
				fmt.Printf("  [available] %-12s %s\n", n, im.Bin)
			}
		}
		for _, n := range sortedSkip(skipped) {
			fmt.Printf("  [skipped]   %-12s %s\n", n, skipped[n])
		}
		fmt.Printf("\nCases (%d):\n", len(cases))
		for _, c := range cases {
			tag := ""
			if c.Heavy {
				tag = " [heavy]"
			}
			fmt.Printf("  %-26s %-8s %s%s\n", c.Name, c.Mode, c.Expect, tag)
		}
		return 0
	}

	if len(available) == 0 {
		fmt.Println("ERROR: no implementations available. Set env overrides or build them.")
		for _, n := range sortedSkip(skipped) {
			fmt.Printf("  - %s: %s\n", n, skipped[n])
		}
		return 2
	}

	programsDir, err := s.programsDirFor()
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost:", err)
		return 2
	}
	results := executeCases(cases, available, s, programsDir, *shake)

	if *jsonOut {
		printJSON(available, skipped, results)
	}
	nfail := printMatrix(cases, results, available, skipped)
	if nfail > 0 {
		return 1
	}
	return 0
}

func printJSON(available map[string]Impl, skipped map[string]string, results map[string]caseResult) {
	type implJSON struct {
		Status string `json:"status"`
		Norm   string `json:"norm"`
	}
	type caseJSON struct {
		Verdict string              `json:"verdict"`
		Detail  string              `json:"detail"`
		Impls   map[string]implJSON `json:"impls"`
	}
	blob := map[string]any{}
	var avail []string
	for _, n := range implOrder {
		if _, ok := available[n]; ok {
			avail = append(avail, n)
		}
	}
	blob["available"] = avail
	blob["skipped"] = skipped
	cj := map[string]caseJSON{}
	for name, r := range results {
		impls := map[string]implJSON{}
		for i, pr := range r.PerImpl {
			impls[i] = implJSON{Status: pr.Status, Norm: pr.Norm}
		}
		cj[name] = caseJSON{Verdict: r.Verdict, Detail: r.Detail, Impls: impls}
	}
	blob["cases"] = cj
	b, _ := json.MarshalIndent(blob, "", "  ")
	fmt.Println(string(b))
}

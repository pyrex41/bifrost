package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed benchmarks/sum-mid.sjk
var sumMidSJK []byte

type joyBenchLine struct {
	NSPerOp    float64 `json:"ns_per_op"`
	StepsPerOp float64 `json:"steps_per_op"`
}

type benchReport struct {
	Schema       int       `json:"schema"`
	GeneratedUTC string    `json:"generated_utc"`
	Platform     string    `json:"platform"`
	Machine      string    `json:"machine"`
	GoVersion    string    `json:"go_version"`
	ShenJoyRev   string    `json:"shen_joy_revision"`
	ShenGoRev    string    `json:"shen_go_revision"`
	ImageSHA     string    `json:"image_checksum"`
	Workload     string    `json:"workload"`
	Samples      int       `json:"samples"`
	ShenJoyNS    []float64 `json:"shen_joy_ns_per_op"`
	ShenGoNS     []float64 `json:"shen_go_ns_per_op"`
	ShenJoySteps float64   `json:"shen_joy_steps_per_op"`
}

var goBenchLine = regexp.MustCompile(`(?m)^BenchmarkVMSumMid(?:-\d+)?\s+\d+\s+([0-9.]+)\s+ns/op`)

func cmdBench(args []string) int {
	fs := newFlagSet("bifrost bench")
	samples := fs.Int("samples", 10, "independent samples per runtime")
	iterations := fs.Int("iterations", 500, "shen-joy VM runs per sample")
	benchtime := fs.String("benchtime", "500ms", "Go benchmark duration per sample")
	output := fs.String("output", "", "write Markdown report to this path")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON")
	if err := fs.Parse(reorderArgs(args, "samples", "iterations", "benchtime", "output")); err != nil {
		return 2
	}
	if fs.NArg() > 0 && fs.Arg(0) != "shen-joy-vs-shen-go" {
		fmt.Fprintf(os.Stderr, "bifrost bench: unknown benchmark %q\n", fs.Arg(0))
		return 2
	}
	if *samples < 1 || *iterations < 1 {
		fmt.Fprintln(os.Stderr, "bifrost bench: samples and iterations must be positive")
		return 2
	}
	report, err := runShenJoyVsGo(*samples, *iterations, *benchtime)
	if err != nil {
		fmt.Fprintln(os.Stderr, "bifrost bench:", err)
		return 1
	}
	var rendered []byte
	if *jsonOut {
		rendered, _ = json.MarshalIndent(report, "", "  ")
		rendered = append(rendered, '\n')
	} else {
		rendered = []byte(renderBenchMarkdown(report))
	}
	if *output != "" {
		if err := os.WriteFile(*output, rendered, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "bifrost bench: writing report:", err)
			return 1
		}
	}
	_, _ = os.Stdout.Write(rendered)
	return 0
}

func runShenJoyVsGo(samples, iterations int, benchtime string) (benchReport, error) {
	joy, joyDir := resolveJoy()
	if joy == "" {
		return benchReport{}, fmt.Errorf("shen-joy not found (use the Bifrost Nix composer or set BIFROST_SHEN_JOY)")
	}
	goDir := os.Getenv("BIFROST_SHEN_GO_DIR")
	if goDir == "" {
		goDir = sibling("shen-go")
	}
	goDir, _ = filepath.Abs(goDir)
	if _, err := os.Stat(filepath.Join(goDir, "go.mod")); err != nil {
		return benchReport{}, fmt.Errorf("shen-go checkout not found at %s (set BIFROST_SHEN_GO_DIR)", goDir)
	}
	if _, err := exec.LookPath("go"); err != nil {
		return benchReport{}, fmt.Errorf("Go toolchain not found; use the Bifrost Nix composer")
	}
	tmp, err := os.MkdirTemp("", "bifrost-shen-joy-bench-")
	if err != nil {
		return benchReport{}, err
	}
	defer os.RemoveAll(tmp)
	source, image := filepath.Join(tmp, "sum-mid.sjk"), filepath.Join(tmp, "sum-mid.sji")
	if err := os.WriteFile(source, sumMidSJK, 0o644); err != nil {
		return benchReport{}, err
	}
	if out, err := exec.Command(joy, "compile", "--profile", "core", source, "--output", image).CombinedOutput(); err != nil {
		return benchReport{}, fmt.Errorf("compiling shen-joy benchmark: %v\n%s", err, out)
	}
	joySamples := make([]float64, 0, samples)
	steps := float64(0)
	for i := 0; i < samples; i++ {
		out, err := exec.Command(joy, "bench", image, "--iterations", strconv.Itoa(iterations), "--warmup", "20").CombinedOutput()
		if err != nil {
			return benchReport{}, fmt.Errorf("shen-joy sample %d: %v\n%s", i+1, err, out)
		}
		var line joyBenchLine
		if err := json.Unmarshal(out, &line); err != nil {
			return benchReport{}, fmt.Errorf("parsing shen-joy sample: %w: %s", err, out)
		}
		joySamples = append(joySamples, line.NSPerOp)
		steps = line.StepsPerOp
	}
	cmd := exec.Command("go", "test", "./kl", "-run", "^$", "-bench", "^BenchmarkVMSumMid$", "-benchmem", "-count", strconv.Itoa(samples), "-benchtime", benchtime)
	cmd.Dir = goDir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return benchReport{}, fmt.Errorf("shen-go benchmark: %v\n%s", err, out)
	}
	matches := goBenchLine.FindAllStringSubmatch(string(out), -1)
	if len(matches) != samples {
		return benchReport{}, fmt.Errorf("expected %d shen-go samples, parsed %d\n%s", samples, len(matches), out)
	}
	goSamples := make([]float64, 0, samples)
	for _, m := range matches {
		n, _ := strconv.ParseFloat(m[1], 64)
		goSamples = append(goSamples, n)
	}
	inspect, _ := exec.Command(joy, "inspect", image).CombinedOutput()
	return benchReport{
		Schema: 1, GeneratedUTC: time.Now().UTC().Format(time.RFC3339),
		Platform: runtime.GOOS + "/" + runtime.GOARCH, Machine: machineName(),
		GoVersion: commandText("go", "version"), ShenJoyRev: gitRev(joyDir), ShenGoRev: gitRev(goDir),
		ImageSHA: fieldLine(string(inspect), "checksum:"), Workload: "sum-mid(0, 8000) = 32004000", Samples: samples,
		ShenJoyNS: joySamples, ShenGoNS: goSamples, ShenJoySteps: steps,
	}, nil
}

func resolveJoy() (string, string) {
	if p := os.Getenv("BIFROST_SHEN_JOY"); p != "" {
		a, _ := filepath.Abs(p)
		return a, sibling("shen-joy")
	}
	if p, err := exec.LookPath("shen-joy"); err == nil {
		return p, sibling("shen-joy")
	}
	d := sibling("shen-joy")
	for _, p := range []string{filepath.Join(d, "build", "shen-joy"), filepath.Join(d, "result", "bin", "shen-joy")} {
		if _, err := os.Stat(p); err == nil {
			return p, d
		}
	}
	return "", d
}

func sibling(name string) string {
	cwd, _ := os.Getwd()
	p, _ := filepath.Abs(filepath.Join(cwd, "..", name))
	return p
}

func commandText(name string, args ...string) string {
	b, _ := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(b))
}

func gitRev(dir string) string {
	s := commandText("git", "-C", dir, "rev-parse", "HEAD")
	if s == "" {
		return "unknown"
	}
	if commandText("git", "-C", dir, "status", "--porcelain") != "" {
		s += "+dirty"
	}
	return s
}

func machineName() string {
	if runtime.GOOS == "darwin" {
		if s := commandText("sysctl", "-n", "machdep.cpu.brand_string"); s != "" {
			return s
		}
	}
	if s := commandText("uname", "-m"); s != "" {
		return s
	}
	return "unknown"
}

func fieldLine(text, prefix string) string {
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(l, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(l, prefix))
		}
	}
	return "unknown"
}

func median(xs []float64) float64 {
	ys := append([]float64(nil), xs...)
	sort.Float64s(ys)
	n := len(ys)
	if n%2 == 1 {
		return ys[n/2]
	}
	return (ys[n/2-1] + ys[n/2]) / 2
}

func minmax(xs []float64) (float64, float64) {
	lo, hi := xs[0], xs[0]
	for _, x := range xs[1:] {
		if x < lo {
			lo = x
		}
		if x > hi {
			hi = x
		}
	}
	return lo, hi
}

func renderBenchMarkdown(r benchReport) string {
	jmed, gmed := median(r.ShenJoyNS), median(r.ShenGoNS)
	jlo, jhi := minmax(r.ShenJoyNS)
	glo, ghi := minmax(r.ShenGoNS)
	ratio := gmed / jmed
	comparison := fmt.Sprintf("shen-go is %.2fx faster than shen-joy", 1/ratio)
	if ratio >= 1 {
		comparison = fmt.Sprintf("shen-joy is %.2fx faster than shen-go", ratio)
	}
	return fmt.Sprintf(`# shen-joy vs shen-go benchmark

Measured %s on %s (%s). Revisions: shen-joy %s, shen-go %s. Go toolchain: %s. Image checksum: %s.

| Runtime | Median ns/op | Min-max ns/op | Samples |
|---|---:|---:|---:|
| shen-joy | %.0f | %.0f-%.0f | %d |
| shen-go | %.0f | %.0f-%.0f | %d |

On this run, %s for this kernel. This is one narrow VM comparison, not a claim about full Shen performance.

Raw ns/op samples: shen-joy [%s]; shen-go [%s].

## Method

The shared workload is a first-order self-tail-recursive sum from 8000 to zero; both return 32004000. It exercises calls, tail calls, locals, integer equality, addition, subtraction, and branching. shen-joy executes %.0f bytecode instructions per operation.

Parsing, image decoding, Shen kernel boot, and process startup are outside the timed regions. shen-joy loads and validates the image once, warms up, then times repeated calls to the allocation-free VM. shen-go uses its Go benchmark BenchmarkVMSumMid, which defines the KLambda function and parses the call before timing repeated Eval operations. The runtime entry APIs differ slightly, so treat the ratio as directional and reproduce it on the deployment CPU.

Reproduce from the Bifrost checkout:

~~~sh
nix run .#env -- shen-go shen-joy -- go run . bench shen-joy-vs-shen-go --samples %d --iterations 500 --benchtime 500ms
~~~
`, r.GeneratedUTC, r.Machine, r.Platform, shortRev(r.ShenJoyRev), shortRev(r.ShenGoRev), r.GoVersion, r.ImageSHA,
		jmed, jlo, jhi, r.Samples, gmed, glo, ghi, r.Samples, comparison,
		formatSamples(r.ShenJoyNS), formatSamples(r.ShenGoNS), r.ShenJoySteps, r.Samples)
}

func shortRev(s string) string {
	dirty := strings.HasSuffix(s, "+dirty")
	s = strings.TrimSuffix(s, "+dirty")
	if len(s) > 12 {
		s = s[:12]
	}
	if dirty {
		return s + "+dirty"
	}
	return s
}

func formatSamples(xs []float64) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.FormatFloat(x, 'f', 0, 64)
	}
	return strings.Join(parts, ", ")
}

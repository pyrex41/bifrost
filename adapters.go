package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// implOrder is the order ports appear in the matrix and in listings.
var implOrder = []string{
	"shen-cl", "shen-go", "shen-joy", "shen-erl", "shen-rust", "shen-lua", "ShenScript", "shen-scheme", "shen-julia", "shen-swift", "shen-truffle", "shen-c",
}

// BuildRecipe is a port's source-build step (git-build install + yggdrasil host).
type BuildRecipe struct {
	Cwd   string     `json:"cwd"`
	Argv  []string   `json:"argv"`
	Steps [][]string `json:"steps"` // multi-step alternative to argv (run in order)
	Out   string     `json:"out"`
}

// InstallSpec is a port's install backend (asdf/mise style).
type InstallSpec struct {
	Method  string   `json:"method"`  // brew | npm | luarocks | git-build
	Package string   `json:"package"` // for brew/npm/luarocks
	Git     string   `json:"git"`     // for git-build (clone remote)
	Ref     string   `json:"ref"`     // optional branch/tag
	Needs   []string `json:"needs"`   // toolchain precheck
}

// Adapter declares how to drive one Shen port. Templates are argv lists with
// {bin}/{expr}/{file} placeholders. A null `version` (shen-lua) unmarshals to
// nil, signalling the (version) eval fallback.
type Adapter struct {
	Env            string                     `json:"env"`
	DefaultPaths   []string                   `json:"default_paths"`
	Launcher       []string                   `json:"launcher"`
	Eval           []string                   `json:"eval"`
	Script         []string                   `json:"script"`
	Version        []string                   `json:"version"`
	Repl           []string                   `json:"repl"`
	HushWritesFile bool                       `json:"hush_writes_file"`
	Status         string                     `json:"status"`
	Kernel         string                     `json:"kernel"`
	Build          *BuildRecipe               `json:"build"`
	Install        *InstallSpec               `json:"install"`
	OSOverrides    map[string]json.RawMessage `json:"os_overrides"`
}

// Adapters is the loaded registry plus the base dir relative paths resolve to.
type Adapters struct {
	raw     map[string]json.RawMessage // name -> raw adapter JSON (with overrides unapplied)
	baseDir string                     // dir relative default_paths resolve against
}

// Impl is a resolved, available port: its effective (OS-overridden) adapter and
// launcher path.
type Impl struct {
	Name string
	Cfg  Adapter
	Bin  string
}

// findAdaptersBytes resolves the adapters config: $BIFROST_ADAPTERS ->
// ./adapters.json -> embedded default. Returns (bytes, baseDir).
func findAdaptersBytes() ([]byte, string) {
	if env := os.Getenv("BIFROST_ADAPTERS"); env != "" {
		if b, err := os.ReadFile(env); err == nil {
			abs, _ := filepath.Abs(env)
			return b, filepath.Dir(abs)
		}
	}
	cwd, _ := os.Getwd()
	cwdCfg := filepath.Join(cwd, "adapters.json")
	if b, err := os.ReadFile(cwdCfg); err == nil {
		return b, cwd
	}
	return embeddedAdapters, cwd
}

// loadAdapters parses the registry, dropping "_"-prefixed comment keys.
func loadAdapters() (*Adapters, error) {
	b, base := findAdaptersBytes()
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(b, &rawMap); err != nil {
		return nil, fmt.Errorf("parsing adapters.json: %w", err)
	}
	out := map[string]json.RawMessage{}
	for k, v := range rawMap {
		if !strings.HasPrefix(k, "_") {
			out[k] = v
		}
	}
	return &Adapters{raw: out, baseDir: base}, nil
}

func (a *Adapters) has(name string) bool { _, ok := a.raw[name]; return ok }

func (a *Adapters) names() []string {
	var ns []string
	for _, n := range implOrder {
		if a.has(n) {
			ns = append(ns, n)
		}
	}
	return ns
}

// effective parses an adapter and shallow-merges its os_overrides for the
// current platform (runtime.GOOS mapped to the adapter schema's platform keys).
func (a *Adapters) effective(name string) (Adapter, error) {
	return a.effectiveForPlatform(name, platformKey())
}

// effectiveForPlatform is the platform-parameterised core (tested on any OS).
func (a *Adapters) effectiveForPlatform(name, platform string) (Adapter, error) {
	var ad Adapter
	if err := json.Unmarshal(a.raw[name], &ad); err != nil {
		return ad, err
	}
	if ov, ok := ad.OSOverrides[platform]; ok {
		// Unmarshaling the override into the same struct overwrites exactly the
		// fields the override sets (a shallow merge).
		if err := json.Unmarshal(ov, &ad); err != nil {
			return ad, err
		}
	}
	return ad, nil
}

// platformKey maps GOOS to the sys.platform keys used in os_overrides.
func platformKey() string {
	switch runtime.GOOS {
	case "windows":
		return "win32"
	case "darwin":
		return "darwin"
	default:
		return "linux"
	}
}

// resolveBin finds a port's launcher: $ENV override, then first existing
// default_path (relative paths against baseDir). On Windows, extensionless
// candidates also match shen.exe/.cmd/.bat (findExecutablePath).
func (a *Adapters) resolveBin(name string, cfg Adapter) string {
	if cfg.Env != "" {
		if v := os.Getenv(cfg.Env); v != "" {
			if hit := findExecutablePath(v); hit != "" {
				abs, _ := filepath.Abs(hit)
				return abs
			}
		}
	}
	for _, p := range cfg.DefaultPaths {
		cand := p
		if !filepath.IsAbs(cand) {
			cand = filepath.Join(a.baseDir, cand)
		}
		if hit := findExecutablePath(cand); hit != "" {
			abs, _ := filepath.Abs(hit)
			return abs
		}
	}
	return ""
}

// discover returns the available impls (resolved launcher) and a skipped map
// (name -> reason). restrict, if non-nil, limits to those names.
func (a *Adapters) discover(restrict map[string]bool) (map[string]Impl, map[string]string) {
	available := map[string]Impl{}
	skipped := map[string]string{}
	for _, name := range a.names() {
		if restrict != nil && !restrict[name] {
			continue
		}
		cfg, err := a.effective(name)
		if err != nil {
			skipped[name] = "adapter parse error: " + err.Error()
			continue
		}
		bin := a.resolveBin(name, cfg)
		if bin == "" {
			env := cfg.Env
			if env == "" {
				env = "?"
			}
			skipped[name] = fmt.Sprintf("launcher not found (set $%s or build it)", env)
		} else {
			available[name] = Impl{Name: name, Cfg: cfg, Bin: bin}
		}
	}
	return available, skipped
}

// resolveOne resolves a single named impl, or returns an error message.
func (a *Adapters) resolveOne(name string) (Impl, error) {
	if !a.has(name) {
		return Impl{}, fmt.Errorf("unknown impl %q (known: %s)", name, strings.Join(a.names(), ", "))
	}
	cfg, err := a.effective(name)
	if err != nil {
		return Impl{}, err
	}
	bin := a.resolveBin(name, cfg)
	if bin == "" {
		return Impl{}, fmt.Errorf("%s is not available; prefer: bifrost env %s -- bifrost <command>", name, name)
	}
	return Impl{Name: name, Cfg: cfg, Bin: bin}, nil
}

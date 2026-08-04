# Bifrost

**A cross-implementation META test harness for the Shen language ports.**

> In Norse myth, *Bifrost* is the burning rainbow bridge between worlds. Here it
> is the bridge that checks the worlds **agree**: it runs the *same* programs and
> behaviours across **all** the Shen implementations and asserts they produce
> the same observable result (differential / conformance testing).

It sits alongside [Ratatoskr / Yggdrasil](../ratatoskr) — the two-stage shaker —
in the same Norse lineage, and drives it for the `--shake` deploy-path checks.

Differential testing is its origin, but Bifrost is now three things:

1. **a differential test harness** (this README's first half) — does every port
   agree on the same input?
2. **a reusable test framework** — any Shen project drops a
   [`bifrost.suite.json`](#using-bifrost-from-another-shen-program-suite-manifests)
   and runs *its own* suite across every port;
3. **a Roswell-style front door** — [`run`/`eval`/`repl`/`impls`/`use`/`install`/`build`](#bifrost-as-a-shen-front-door-roswell-style)
   give Shen one CLI across all ports, on Linux, macOS **and Windows**.

It is packaged as a [uv](https://docs.astral.sh/uv/) tool, so it runs with no
install: `uvx --from git+https://…/bifrost bifrost …`.

## The three-way distinction (read this first)

There are **three** different kinds of Shen test suite. Bifrost is the third one
and is deliberately distinct from the other two:

| Suite | What it tests | Owned by | Example |
|-------|---------------|----------|---------|
| **(a) Canonical kernel suite** | Does *this* port implement the Shen *spec* correctly? | The Shen kernel (`tests/`) | `(run "README.shen")` style kernel conformance |
| **(b) Per-port unit tests** | Does *this* port's internals (reader, writer, GC, FFI…) behave? | Each port repo | `shen-go/cmd/shen/main_test.go` |
| **(c) Bifrost (this repo)** | Do **all ports agree with each other** on the same input? | This repo | `(+ 0.1 0.2)` → are the answers the same string across ports? |

Bifrost never re-implements (a) or (b). It drives each port's *launcher* exactly
the way a user would from the shell, captures stdout, normalises launcher
chatter, and **diffs the ports against each other** (or against a golden value).

## What it covers

The corpus (`cases/*.json`, driven by `programs/*.shen`) includes:

- **Behavioural parity** — arithmetic (incl. floats), list ops, string ops,
  closures (incl. currying), `fix`, recursion, **tail calls** (a 100 000-deep
  countdown that must not blow any stack), a small **Prolog** query, and
  **`trap-error`** catchability.
- **CLI parity** — `eval -e` prints the value; `(version)` / `--version` carry
  the kernel version **41.2**; **stdin-EOF causes a clean exit** (no hang) on
  every impl.
- **Divergences** — **none open.** Every tracked cross-port difference has
  converged and is now asserted as a **hard agreement** (a regression is a real
  FAIL, no longer a documented difference):
  - `float-formatting` — `(+ 0.1 0.2)` prints the shortest round-trippable
    `0.30000000000000004` everywhere (was shen-go `0.300000` / shen-lua `0.3`;
    pyrex41/shen-go#11, pyrex41/shen-lua#24).
  - `int-div-zero` — `(/ 1 0)` raises a catchable `divide-by-zero` everywhere
    (was shen-go `maxint`).
  - `hush-file-write` — under `-q`, `pr` to a **file** stream writes everywhere
    (`*hush*` gates only stdout; was a zero-byte file on shen-lua/shen-rust).
  - `load-toplevel-echo` — `(load FILE)` echoes each top-level form's value
    regardless of fasl-cache state (shen-lua dropped the echo on a warm cache
    hit; pyrex41/shen-lua#40).
- **Heavy** (`--heavy`) — **Ratatoskr stage-1 parity**: run
  `(ratatoskr.shake ["tests/fib.shen"] OUT)` on every host and assert the
  produced `kernel.kl` + `ratatoskr.manifest` are **byte-identical** across
  hosts. (User KL differs only by gensym counter, so it is *not* asserted.)

### Expected vs. agreement modes

Each case is one of two modes:

- **`expect: output`** — assert each impl's normalised stdout equals a *golden*
  value.
- **`expect: agreement`** — no golden; assert **all available impls produce
  identical** normalised output. If a case is tagged `known_divergence`, a
  disagreement is reported as **DIVERGE** (documented) instead of **FAIL**.

## Running it

```bash
# Standalone runner — works with just python3, no third-party deps:
python3 bifrost.py                 # light cases + matrix; exit !=0 on real FAIL
python3 bifrost.py --heavy         # also run the ratatoskr stage-1 parity case
python3 bifrost.py --list          # list discovered impls + cases
python3 bifrost.py --only int-mul float-add-imprecise
python3 bifrost.py --impls shen-cl,shen-go
python3 bifrost.py --json          # machine-readable result blob

# Optional pytest wrapper (same corpus; divergences become xfail):
pytest
BIFROST_HEAVY=1 pytest -k ratatoskr
```

`DIVERGE` rows are reported in their own section and **do not** fail the run.
Only real `FAIL` rows set a non-zero exit code.

### Install

Bifrost is a single static **Go binary** (no runtime deps), distributed three
ways — pick whichever suits you:

```bash
# 1. Go toolchain — installs to $GOBIN:
go install github.com/pyrex41/bifrost@latest

# 2. Prebuilt release binary (no toolchain) — download for your OS/arch from
#    the GitHub Releases page (produced by GoReleaser on each v* tag), unpack,
#    put `bifrost` on your PATH.

# 3. uvx — builds the Go binary on your machine and runs it (needs Go installed):
uvx --from git+https://github.com/pyrex41/bifrost bifrost --list
uvx --from . bifrost impls                            # from a local checkout
```

The binary embeds `adapters.json` + the corpus, so it is self-contained — but
the Shen *ports* it drives are resolved on your machine. Point it at your ports
with the per-impl env vars (below) or a project-local `adapters.json`;
resolution order is `$BIFROST_ADAPTERS` → `./adapters.json` → the embedded
default.

> The Python implementation (`bifrost.py`) remains in the repo as the reference
> oracle the Go binary is tested against (`bifrost --json` is byte-identical to
> `python bifrost.py --json` across the corpus); `go test ./...` plus a
> Windows/Linux/macOS CI matrix guard the binary.

### Shake-then-run (deploy-path parity)

`--shake` runs each **script-mode** program through
[Ratatoskr](../ratatoskr): it tree-shakes the program once, builds a *standalone
artifact* for every target (Lisp/Lua/Go/Rust/JS/Julia), runs each artifact, and
diffs them. This checks the real stand-alone deploy path, not just
load-from-source.

```bash
python3 bifrost.py --shake --only recursion-fib-file   # all buildable targets
python3 bifrost.py --shake --impls shen-lua,ShenScript # just the fast ones
```

Artifacts map onto their impl column (lisp→`shen-cl`, lua→`shen-lua`,
go→`shen-go`, rust→`shen-rust`, js→`ShenScript`, julia→`shen-julia`,
scheme→`shen-scheme`, swift→`shen-swift`) — all eight ports now have a
Ratatoskr builder. Needs the per-target toolchains
(`sbcl`/`luajit`/`go`/`cargo`/`node`/`julia`/`chez`/`swift`) and
`$BIFROST_RATATOSKR_DIR` (default `../ratatoskr`); missing toolchains are
skipped, not failed. This mode is minutes, not seconds: go/rust compile from
scratch, and the **julia** target AOT-bakes a per-program sysimage
(PackageCompiler, ~250MB, several minutes) — the deploy artifact for
Shen-on-Julia, analogous to the Lisp saved image. The **scheme** target
compiles the shaken slice with shen-scheme's own `kl->scheme` into a
self-contained Chez program; the **swift** target drives the shen-swift
tree-walking interpreter on the shaken slice in `--shaken` mode (booting a
~200-line kernel instead of the full ~2500-line kernel).

`--shake` drives the **Go `ratatoskr` binary** (resolved on `$PATH`, else
`$RATATOSKR_BIN`, else `$BIFROST_RATATOSKR_DIR` / a sibling `../ratatoskr`); no
Python is involved. The heavy `ratatoskr-shake-parity` case (`--heavy`) likewise
shakes on each host via the Go ratatoskr and diffs the kernel.kl/manifest md5s.

## How implementations are located

Bifrost **auto-detects** each port. For every impl it resolves a launcher path
in this order, and **skips-with-report** (never errors) any that is missing:

1. the impl's env-var override (e.g. `$BIFROST_SHEN_GO`), if it points at an
   existing file;
2. the first existing path in the adapter's `default_paths`.

Defaults (see [`adapters.json`](adapters.json)):

| Impl | Env override | Default launcher |
|------|--------------|------------------|
| `shen-cl` | `BIFROST_SHEN_CL` | `…/shen-cl/bin/sbcl/shen` (also `clisp`, `ecl`) |
| `shen-go` | `BIFROST_SHEN_GO` | `.bin/shen-go` (build: `go build -o .bin/shen-go ./cmd/shen`) |
| `shen-rust` | `BIFROST_SHEN_RUST` | `…/shen-rust/target/release/shen-rust` |
| `shen-lua` | `BIFROST_SHEN_LUA` | `…/shen-lua/bin/shen` (needs `luajit`; falls back to the luarocks-installed `/usr/local/bin/shen`) |
| `ShenScript` | `BIFROST_SHENSCRIPT` | `node …/ShenScript/bin/shen.js` |
| `shen-scheme` | `BIFROST_SHEN_SCHEME` | `…/shen-scheme/_build/bin/shen-scheme` (Chez) |
| `shen-julia` | `BIFROST_SHEN_JULIA` | `…/shen-julia/bin/shen` (needs `julia`) |
| `shen-swift` | `BIFROST_SHEN_SWIFT` | `…/shen-swift/.build/release/shen-swift` (needs `swift`) |

All eight target ShenOSKernel **41.2**. Run `bifrost impls --versions` to see the
live per-port kernel version, install state, and which is active.

To build shen-go locally into the gitignored `.bin/`:

```bash
go build -o .bin/shen-go -C /path/to/shen-go ./cmd/shen
```

### Launcher quirks Bifrost encodes (real cross-impl differences)

- **shen-cl** *requires* `-q` for clean `eval`/REPL output.
- **shen-lua** has **no** `--version` flag and **no** `script` subcommand — its
  file path is `(load FILE)` (which echoes `(fn …)` and a `run time:` banner);
  Bifrost's normaliser strips that chatter. It also has no clean script value
  channel, so file-mode programs end with `(do (print …) (nl))`.
- **shen-lua / shen-rust** must be driven **without** `-q` for normal cases and
  for the ratatoskr parity case, or `pr` output is silenced (the
  `hush-file-write` divergence). This is exactly why Bifrost's eval/script
  templates for those two do **not** pass `-q`.

## Bifrost as a Shen front door (Roswell-style)

Beyond differential testing, Bifrost is the single **front door** for Shen
across every port — the way [Roswell](https://roswell.github.io/) (`ros`) is for
Common Lisp. The differential test matrix is still the default (bare `bifrost`);
these are added as verbs. They reuse the same `adapters.json` port registry, so
"run my program on shen-go", "which ports do I have", and "install shen-rust"
all share one source of truth.

```bash
bifrost run prog.shen [--impl X]   # run a .shen program on the active/chosen port
bifrost eval -e '(+ 2 3)' [--impl X]
bifrost repl [--impl X]            # interactive REPL (inherits your terminal)
bifrost impls [--versions]         # list ports: state, kernel (live probe), status, active(*)
bifrost use IMPL [--project]       # set the active port (global, or ./.bifrost-impl pin)
bifrost install IMPL [--method M] [--git URL] [--ref R] [--force]
bifrost build prog.shen OUT --target T [--run]   # standalone artifact (delegates to Ratatoskr)
```

### Choosing the implementation

`run`/`eval`/`repl` resolve which port to use in this order (rustup-style — a
per-command override beats a pin beats the global default):

1. `--impl X` on the command line,
2. a project pin file `./.bifrost-impl` (written by `bifrost use X --project`),
3. the global default `~/.config/bifrost/impl` (written by `bifrost use X`),
4. otherwise `shen-cl` if present, else the first available port.

`bifrost impls --versions` probes each port's kernel version **live** (the port
READMEs' badges drift — e.g. shen-cl's says 41.1 but it reports 41.2), so the
matrix of who-is-on-what is always accurate.

### Installing a port

Each port declares an **install backend** in `adapters.json` (asdf/mise style):

| method | ports | what runs |
|---|---|---|
| `brew` | shen-scheme (and shen-cl via `--method brew`) | `brew install <formula>` |
| `luarocks` | shen-lua via `--method luarocks` | `luarocks install shen` (rock **0.10.0-1**+ bundles kernel **41.2**; only the old 0.9.0-1 was 41.1) |
| `git-build` | shen-cl, shen-go, shen-rust, shen-lua, ShenScript, shen-julia, shen-swift | clone (if absent) + the port's `build` recipe (single `argv`, or a `steps` list run in order) |

`install` prechecks the required toolchain (and names the exact missing tool
rather than failing opaquely — when `--method` overrides the default, the
precheck follows the *chosen* backend), refuses `experimental` ports unless
`--force`, is idempotent (skips an already-resolved launcher), and verifies
the launcher resolves afterward.

Port-specific install notes (see each adapter's `_install_note`):

- **shen-go / ShenScript** — git-build clones the **pyrex41 forks**: they carry
  the standard launcher CLI (`eval -e` / `script` / `--version`, kernel 41.2)
  that Bifrost drives. Upstream `tiancaiamao/shen-go` lacks those subcommands
  (S41.1, no clean stdin-EOF exit), and the published npm `shen-script`
  package is a library with no `bin/shen.js` launcher, so neither can serve as
  a Bifrost port. ShenScript's recipe is two steps: `npm install` +
  `npm run build-kernel` (renders `lib/kernel.js`).
- **shen-lua** — runs straight from the checkout (`bin/shen`); KLambda is
  compiled on first boot and cached, so the build step just warms that cache.
  Needs `luajit` at runtime.
- **shen-cl** — a fresh clone has no `kernel/` or `compiled/` tree.
  **Bootstrap once by hand** before `bifrost install shen-cl`:
  `make fetch` (or `scripts/assemble-tarver-kernel.sh`) and then
  `make precompile SHEN=<any working Shen launcher, e.g. a built shen-go>` —
  precompiling the kernel requires an already-working Shen. After that,
  `bifrost install shen-cl` (which runs `make build-sbcl`) works as usual.

#### Fresh-machine bootstrap (verified on a clean Linux container)

On a machine with none of the ports present, this order works with the fewest
prerequisites (each port lands at the clone/launcher locations in
`adapters.json` — the bundled defaults assume sibling checkouts (for example,
`../shen-go` and `../ratatoskr`). If your ports live elsewhere, point a
project-local `adapters.json` or `$BIFROST_ADAPTERS` at your own paths first:

1. `bifrost install shen-go` — needs only `git` + `go`; gives you a working
   41.2 Shen for the shen-cl bootstrap below.
2. `bifrost install shen-rust` (needs `cargo`), `bifrost install ShenScript`
   (needs `node`/`npm`), `bifrost install shen-lua` (needs `luajit`).
3. shen-cl: clone, `make fetch` (kernel sources), `make precompile
   SHEN=<the shen-go binary from step 1>`, then `bifrost install shen-cl`
   (needs `sbcl`, e.g. `apt install sbcl`).
4. `bifrost install shen-scheme --method git-build` (needs `make` + a C
   compiler + network access to fetch Chez), `bifrost install shen-julia`
   (needs `julia`), `bifrost install shen-swift` (needs `swift`).

Every failure names the exact missing tool, and missing ports are skipped —
never a hard error — so a partial fleet (e.g. steps 1–3 on a container with
no Julia/Swift toolchain) still runs the full matrix across whatever is
installed.

**Forks** (e.g. `pyrex41/shen-cl`) are first-class:
- to *run* a fork, point the port's env var at your built binary
  (`$BIFROST_SHEN_CL=/path/to/fork/bin/sbcl/shen`) — no install needed;
- to *build* a fork, override the git-build source:
  `bifrost install shen-cl --git https://github.com/me/shen-cl --ref my-branch`
  (or `$BIFROST_SHEN_CL_GIT` / `$BIFROST_SHEN_CL_REF`). The git-build defaults
  point at the `pyrex41` forks this workspace tracks.

### Windows / Linux / macOS

The `bifrost` tool itself is pure-stdlib Python and runs on all three. The
plumbing is OS-aware:

- **Launcher resolution** — extensionless `default_paths` also match
  `shen.exe` / `shen.cmd` / `shen.bat` on Windows (via `PATHEXT`); a
  `.bat`/`.cmd`/`.sh` launcher is auto-wrapped (`cmd /c` / `sh`) so it runs
  through Python's shell-less `subprocess`.
- **Config** — the global active-impl pin lives in `%APPDATA%\bifrost\impl` on
  Windows, `$XDG_CONFIG_HOME`/`~/.config/bifrost/impl` elsewhere.
- **Per-OS adapter tweaks** — an adapter may carry an `os_overrides` block
  keyed by `win32`/`darwin`/`linux` (shallow-merged over the adapter) to give a
  port a platform-specific `default_paths`/`launcher`/template. See the
  `_os_overrides_example` in [`adapters.json`](adapters.json) — e.g. driving
  shen-lua on Windows by invoking `luajit` explicitly.

What's *not* in Bifrost's hands is whether a given **port** runs on Windows —
that's each port's own story. Natively-compiled ports (shen-go, shen-rust,
shen-cl, shen-scheme) produce a Windows `.exe`; interpreter ports work when
their runtime is present (`node` for ShenScript, `luajit`, `julia`). As
everywhere, point `$BIFROST_<IMPL>` at your launcher if auto-detection misses
it. The `portability` CI job runs the cross-platform plumbing tests on
`windows-latest` (and Linux/macOS) on every push.

## Using Bifrost from another Shen program (suite manifests)

Bifrost is not only its own corpus — it is a **reusable framework** any Shen
project can point at its *own* test suite to run it across every supported
port and assert the ports agree. A project written in Shen (e.g.
[`../shen-cas`](../shen-cas)) gets cross-implementation conformance for free:
"does my suite produce the *same* result, and pass, on every supported port?"

A project plugs in by dropping a **`bifrost.suite.json`** manifest in its repo
and running:

```bash
python3 /path/to/bifrost/bifrost.py --suite /path/to/project/bifrost.suite.json --heavy
# or, checked out side-by-side:
python3 ../bifrost/bifrost.py --suite ./bifrost.suite.json --heavy
```

Ports are resolved from Bifrost's own [`adapters.json`](adapters.json) (launcher
locations are machine-global, not project-specific), so the project supplies
**only** what is project-specific: where its sources live and how its suite
reports success.

### Manifest schema

```jsonc
{
  "name": "my-project",
  "root": ".",                       // project root; paths below resolve here.
                                     //   default = the manifest's own directory
  "programs_dir": ".",               // where script-mode {file} programs live
                                     //   (default = root)
  "default_cwd": ".",                // cwd the launcher runs in, so the suite's
                                     //   relative (load "src/...") resolves
                                     //   (default = root)
  "strip_line_prefixes": ["my loader banner"],  // extra normaliser prefixes:
                                     //   drop project-specific chatter lines
  "cases": [                         // inline cases (preferred). Alternatively
    {                                //   "cases_dir": "bifrost/cases" points at
      "name": "self-suite",          //   a dir of *.json corpus files.
      "mode": "script",              // "script" runs a .shen entrypoint
      "program": "load.shen",        // your loader, which ends by running tests
      "expect": "agreement",         // all ports must produce identical output
      "marker": "ALL PASS",          // ...AND each must print this success token
      "heavy": true,                 // big suite -> use the long timeout
      "doc": "all ports agree and report ALL PASS"
    }
  ]
}
```

### How a project's entrypoint should behave

The `"script"` entrypoint (your `load.shen` or equivalent) should:

1. load your sources,
2. run your checks, printing **deterministic** output (identical across ports —
   beware gensym counters, hash ordering, and float formatting, which Bifrost
   will surface as a divergence),
3. end by printing a **marker** line (e.g. `ALL PASS`) *iff* everything passed.

Bifrost runs it on every available port with the cwd set to your project root,
normalises launcher chatter (run-time banners, `(fn …)` load echoes, and
shen-lua's trailing value echo — so your entrypoint may safely return a
boolean), then asserts:

| `expect` | `marker` | assertion |
|----------|----------|-----------|
| `agreement` | set | normalised stdout **byte-identical across all ports** AND the (shared) output contains the marker — *the strongest mode* |
| `agreement` | — | byte-identical across all ports (pure differential) |
| `marker` | set | each port prints the marker and exits cleanly; ports need **not** agree (the project's own harness is the sole judge) |
| `output` | optional | normalised stdout equals `golden` (+ marker if set) |

A worked, runnable example lives in
[`examples/tiny-suite/`](examples/tiny-suite/) — a tiny self-test that passes
on every available port. shen-cas ships a real manifest at
[`../shen-cas/bifrost.suite.json`](../shen-cas/bifrost.suite.json).

Add `--shake` to run a suite's script-mode entrypoint through the
[deploy-path parity](#shake-then-run-deploy-path-parity) pipeline: each port's
*standalone artifact* is built and diffed, not just the load-from-source run.

### Scripting it (importable API)

`bifrost.py` is importable; the manifest path is just sugar over it:

```python
import bifrost
suite = bifrost.build_suite_from_manifest("path/to/bifrost.suite.json")
blob, results, available, skipped, cases = bifrost.run_suite(suite, include_heavy=True)
print(blob["cases"]["self-suite"]["verdict"])   # PASS / FAIL / DIVERGE
```

## Adding a case

1. If the case needs a `.shen` program, drop it in `programs/`. End it with a
   single `(do (print EXPR) (nl))` top-level form so the output normalises
   cleanly across all impls (shen-lua's `load` echoes values otherwise).
2. Add an entry to a file under `cases/` (or make a new `*.json`):

   ```json
   {
     "name": "my-case",
     "mode": "eval",            // or "script" (uses "program": "foo.shen")
     "expr": "(+ 2 3)",         // for eval mode
     "expect": "output",        // or "agreement"
     "golden": "5",             // required when expect == output
     "doc": "what this checks"
   }
   ```

   For an agreement case that is a *documented* difference, add
   `"known_divergence": "some-tag"` and omit the golden.
3. Run `python3 bifrost.py --only my-case` and confirm the matrix.

## Layout

```
*.go                the bifrost Go binary (primary):
  main.go             subcommand router + flag helpers
  adapters.go         port registry: load / discover / resolve / os_overrides
  run.go              launch path: build_argv, run, normalize, exec wrapping
  front.go            verbs: run/eval/repl/impls/use
  install.go          install backends + build (delegates to ratatoskr)
  matrix.go,runmatrix.go  differential test matrix, --suite, reporting
  shake.go            --shake + ratatoskr-parity (drive the Go ratatoskr)
  *_test.go           cross-platform unit tests (go test ./...)
bifrost.py          the original Python — kept as the reference oracle
test_bifrost.py     pytest wrapper over the same corpus (oracle checks)
pybin/              hatchling wheel that builds+bundles the Go binary for uvx
adapters.json       per-impl launchers + arg templates + hush flags + install backends
cases/*.json        data-driven case corpus (Bifrost's own suite)
programs/*.shen     .shen programs referenced by script-mode cases
examples/           worked third-party suite manifests (--suite)
.goreleaser.yaml    release-binary build (on v* tags)
.github/workflows/  go (build/test, win/linux/mac), portability, matrix, release
```

Run `python3 bifrost.py --suite PATH/bifrost.suite.json` to drive an external
project's suite (see *Using Bifrost from another Shen program* above).

## CI

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs the matrix
best-effort. It is fine for CI to build/run only a subset of impls; missing
impls are **skipped and reported**, not failed.

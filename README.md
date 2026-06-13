# Bifrost

**A cross-implementation META test harness for the Shen language ports.**

> In Norse myth, *Bifrost* is the burning rainbow bridge between worlds. Here it
> is the bridge that checks the worlds **agree**: it runs the *same* programs and
> behaviours across **all five** Shen implementations and asserts they produce
> the same observable result (differential / conformance testing).

Bifrost sits alongside [Ratatoskr / Yggdrasil](../ratatoskr) in the same Norse
lineage — Ratatoskr is the squirrel that runs *up and down* the world-tree (the
two-stage shaker); Bifrost is the bridge that verifies the worlds at the ends of
the tree agree.

## The three-way distinction (read this first)

There are **three** different kinds of Shen test suite. Bifrost is the third one
and is deliberately distinct from the other two:

| Suite | What it tests | Owned by | Example |
|-------|---------------|----------|---------|
| **(a) Canonical kernel suite** | Does *this* port implement the Shen *spec* correctly? | The Shen kernel (`tests/`) | `(run "README.shen")` style kernel conformance |
| **(b) Per-port unit tests** | Does *this* port's internals (reader, writer, GC, FFI…) behave? | Each port repo | `shen-go/cmd/shen/main_test.go` |
| **(c) Bifrost (this repo)** | Do **all ports agree with each other** on the same input? | This repo | `(+ 0.1 0.2)` → are the five answers the same string? |

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
- **Known divergences** (documented, *not* hard failures — see below):
  - `float-formatting` — `(+ 0.1 0.2)` prints `0.30000000000000004` on
    shen-cl/shen-rust/ShenScript, `0.300000` on shen-go, `0.3` on shen-lua.
  - `int-div-zero` — `(/ 1 0)` raises a catchable error on
    cl/rust/lua/ShenScript, but shen-go returns `maxint`.
  - `hush-file-write` — under `-q` (`*hush* = true`), `pr` to a **file** stream
    still writes on shen-cl/shen-go/ShenScript but is **silenced** (zero-byte
    file) on shen-lua/shen-rust.
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
| `shen-lua` | `BIFROST_SHEN_LUA` | `…/shen-lua/bin/shen` (needs `luajit`) |
| `ShenScript` | `BIFROST_SHENSCRIPT` | `node …/ShenScript/bin/shen.js` |

To build shen-go locally into the gitignored `.bin/`:

```bash
go build -o .bin/shen-go -C /Users/reuben/projects/shen/shen-go ./cmd/shen
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

## Adding a case

1. If the case needs a `.shen` program, drop it in `programs/`. End it with a
   single `(do (print EXPR) (nl))` top-level form so the output normalises
   cleanly across all five impls (shen-lua's `load` echoes values otherwise).
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
bifrost.py          standalone runner (source of truth)
test_bifrost.py     optional pytest wrapper over the same corpus
adapters.json       per-impl launcher paths + arg templates + hush flags
cases/*.json        data-driven case corpus
programs/*.shen     .shen programs referenced by script-mode cases
.bin/               (gitignored) auto-built / detected launcher binaries
.github/workflows/  best-effort CI matrix
```

## CI

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) runs the matrix
best-effort. It is fine for CI to build/run only a subset of impls; missing
impls are **skipped and reported**, not failed.

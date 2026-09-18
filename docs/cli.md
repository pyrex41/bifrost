# Bifrost CLI

Bifrost is a static Go binary. Bare `bifrost` runs the agreement matrix.
These verbs are the everyday front door (Roswell-style).

```bash
bifrost run prog.shen [--impl X]
bifrost eval -e '(+ 2 3)' [--impl X]
bifrost repl [--impl X]
bifrost impls [--versions]
bifrost use IMPL [--project]
bifrost env IMPL [IMPL ...|all] -- COMMAND
bifrost build prog.shen OUT --target T [--run]
bifrost install IMPL …          # legacy; prefer env + Nix — docs/install.md
```

## Choosing a port

`run` / `eval` / `repl` pick a port in this order:

1. `--impl X`
2. `./.bifrost-impl` (`bifrost use X --project`)
3. `~/.config/bifrost/impl` (`bifrost use X`; on Windows `%APPDATA%\bifrost\impl`)
4. `shen-cl` if present, else the first available port

`bifrost impls --versions` probes each port's `(version)` live. README badges
drift; this does not.

## How a launcher is found

For each adapter in [`adapters.json`](../adapters.json):

1. the env-var override (e.g. `$BIFROST_SHEN_GO`) if it names an existing file
2. the first existing path in `default_paths`

Missing ports are **skipped and reported**, never a hard error.

| Impl | Env | Default launcher |
|------|-----|------------------|
| `shen-cl` | `BIFROST_SHEN_CL` | `../shen-cl/bin/sbcl/shen` |
| `shen-go` | `BIFROST_SHEN_GO` | `.bin/shen-go` |
| `shen-joy` | `BIFROST_SHEN_JOY` | `../shen-joy/build/shen-joy` (bounded image; source/eval/REPL SKIP) |
| `shen-erl` | `BIFROST_SHEN_ERL` | `../shen-erl/bin/shen-erl` |
| `shen-rust` | `BIFROST_SHEN_RUST` | `../shen-rust/target/release/shen-rust` |
| `shen-lua` | `BIFROST_SHEN_LUA` | `../shen-lua/bin/shen` |
| `ShenScript` | `BIFROST_SHENSCRIPT` | `node ../ShenScript/bin/shen.js` |
| `shen-scheme` | `BIFROST_SHEN_SCHEME` | `../shen-scheme/_build/bin/shen-scheme` |
| `shen-julia` | `BIFROST_SHEN_JULIA` | `../shen-julia/bin/shen` |
| `shen-swift` | `BIFROST_SHEN_SWIFT` | `../shen-swift/.build/release/shen-swift` |
| `shen-truffle` | `BIFROST_SHEN_TRUFFLE` | `../shen-truffle/target/shen-truffle/bin/shen-truffle` |
| `shen-c` | `BIFROST_SHEN_C` | `../shen-c/bin/shen-c` (experimental) |

Production ports target ShenOSKernel **42**. `shen-joy` is a bounded S42-derived
subset. `shen-c` is experimental and self-locates its kernel (do not set
`$SHEN_C_HOME`). Unfinished `shen-forth`, `shen-inets`, `shen-ocaml`, and
`shen-odin` have flakes but are not Bifrost implementations.

### Launcher quirks encoded in adapters

- **shen-cl** needs `-q` for clean `eval` output. Do not pass `-q` as a bare
  first argument: that is rejected before the REPL starts.
- **shen-lua** has no `--version` and no `script` subcommand. File mode is
  `(load FILE)`; Bifrost strips the load chatter.
- **shen-lua / shen-rust** must be driven **without** `-q` for normal cases, or
  `pr` is silenced.

## Windows / Linux / macOS

The binary runs on all three. Extensionless `default_paths` also match
`shen.exe` / `.cmd` / `.bat` on Windows. A `.bat`/`.cmd`/`.sh` launcher is
wrapped (`cmd /c` / `sh`). An adapter may carry `os_overrides` keyed by
`win32` / `darwin` / `linux`. Whether a *port* itself runs on Windows is that
port's story.

## `bifrost build`

Delegates to Yggdrasil: shake `prog.shen` and emit a stand-alone artifact for
`--target` (`lisp`, `lua`, `go`, `joy`, `erlang`, `rust`, `js`, `julia`,
`scheme`, `swift`, `truffle`). `--run` then executes it.

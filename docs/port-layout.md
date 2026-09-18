# Shared port directory structure

Shen ports grew independently. Kernel files, the official test suite, StLib,
and the host runtime sit in different places in each repo. That makes Bifrost,
Yggdrasil, and a porter hunt.

This is the layout we want every port to converge on. It is **not** a Bifrost
gate yet: adapters still list concrete launcher paths. When a port matches
this tree, those paths become defaults instead of special cases.

## Canonical tree

```
<port>/
  README.md                 how to build and run this port
  LICENSE
  flake.nix                 exports #toolchain (and usually #devShell)
  Makefile / language build file
  kernel/
    klambda/                S42 .kl modules, load order as install.lsp
      PROVENANCE.md         tag, hash, which Tarver/community tree
      core.kl
      …
    tests/                  official kernel suite (runme.shen, harness.shen, …)
    sources/                optional Shen sources that render the .kl
    lib/                    Tarver libraries (StLib under lib/StLib or lib/stlib)
    LICENSE.txt
  src/                      host-language runtime (or crates/, cmd/, …)
  bin/                      built launcher
    shen                    preferred name
    <backend>/shen         if one repo builds several backends (sbcl/clisp)
  tests/                    *this port's* unit tests — not the kernel suite
```

## Why these names

They already exist in the reference-shaped ports:

| Path | Role | Already here |
|------|------|----------------|
| `kernel/klambda/` | boot modules | shen-cl, shen-go, shen-rust, shen-c |
| `kernel/tests/` | 134-test official suite | shen-cl, shen-go, shen-rust |
| `kernel/sources/` | Shen that compiles to KL | shen-go |
| `kernel/lib/` | StLib and friends | shen-cl, shen-go |
| `bin/shen` | user-facing launcher | shen-lua (`bin/shen`), shen-cl (`bin/sbcl/shen`) |
| `flake.nix` `#toolchain` | what `bifrost env` composes | most pyrex41 ports |

Host-language layout stays idiomatic (`crates/` for Rust, `cmd/` for Go).
Bifrost only needs to find **kernel**, **tests**, **lib**, and **the
launcher**.

## Current drift (what to fix)

| Port | Today | Move toward |
|------|--------|-------------|
| shen-lua | `klambda/`, `tests/`, `lib/StLib/` at repo root | `kernel/klambda`, `kernel/tests`, `kernel/lib` |
| shen-rust | `kernel/stlib/` | `kernel/lib/StLib` (or keep `stlib` but under `lib/`) |
| shen-go | `kernel/` is right; StLib also embedded at `cmd/shen/stlib/` | keep embed, treat `kernel/lib` as source of truth |
| yggdrasil | `KLambda/` (capital K), `tests/` are *shaker* tests | keep as a tool repo; do not pretend it is a port |
| ShenScript | kernel rendered into JS | still vendor `kernel/klambda` + `kernel/tests` for humans and Bifrost |

## What Bifrost will do with this

1. **Document** (this file).
2. **Resolve** launchers as today via `adapters.json`.
3. **Later**: if `../<port>/bin/shen` exists, prefer it; if
   `../<port>/kernel/tests/runme.shen` exists, optional “run this port’s
   official suite through Bifrost” without each adapter teaching the path.
4. **Never** fail a matrix run because a port has not moved files yet.

A port is “layout-clean” when:

- `kernel/klambda/PROVENANCE.md` names the exact S42 (or later) drop
- `kernel/tests/runme.shen` is the official suite, unmodified
- the launcher is `bin/shen` or `bin/<backend>/shen`
- `flake.nix` exports `#toolchain`

Until then, adapters remain the source of truth for paths.

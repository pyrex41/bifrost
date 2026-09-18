# Bifrost

Bifrost is the **front door for Shen ports**. One CLI runs a program, starts a
REPL, or checks that every implementation agrees on the same input.

It is named for the rainbow bridge: it does not *be* a Shen runtime. It talks
to the ports ([shen-cl](https://github.com/pyrex41/shen-cl),
[shen-go](https://github.com/pyrex41/shen-go),
[shen-rust](https://github.com/pyrex41/shen-rust),
[shen-lua](https://github.com/pyrex41/shen-lua), …) the way a user would, then
compares what they print.

[Yggdrasil](https://github.com/pyrex41/yggdrasil) is the companion shaker:
Bifrost can drive it with `--shake` to check stand-alone deploy artifacts, not
just load-from-source.

## Install

```bash
nix run github:pyrex41/bifrost -- impls
# or
go install github.com/pyrex41/bifrost@latest
```

Nix is the supported way to get **port toolchains**. The Go binary itself is
just the CLI. Sibling checkouts under one parent (`../shen-go`, `../shen-cl`,
…) are the default layout; set `BIFROST_PORTS_ROOT` if they live elsewhere.

```bash
nix run .#env -- shen-go -- bifrost repl --impl shen-go
nix run .#env -- all -- bifrost          # full agreement matrix
```

## Everyday use

```bash
bifrost run prog.shen [--impl shen-go]   # run a program
bifrost eval -e '(+ 2 3)' [--impl X]
bifrost repl [--impl X]
bifrost impls --versions                 # what is installed, which kernel
bifrost use shen-go                      # default port (or --project)
bifrost                                  # run this repo's agreement corpus
```

Which port is used: `--impl` → `./.bifrost-impl` → `~/.config/bifrost/impl` →
first available (preferring `shen-cl`).

Details: [docs/cli.md](docs/cli.md).

## What Bifrost is *not*

| Suite | Question | Owner |
|-------|----------|--------|
| Kernel `tests/` | Does *this* port implement the spec? | Each port + Shen kernel |
| Port unit tests | Do *this* port's internals work? | Each port repo |
| **Bifrost** | Do **all ports agree** on the same input? | This repo |

Bifrost never re-implements the kernel suite. It launches each port, captures
stdout, and diffs.

## Agreement tests

```bash
go run .                    # light corpus; FAIL → exit ≠ 0
go run . --heavy            # also Yggdrasil stage-1 byte-identity
go run . --only float-add-imprecise
go run . --impls shen-cl,shen-go
go run . --json
```

Cases live in `cases/*.json` and `programs/*.shen`. A case is either a golden
`expect: output` or an `expect: agreement` (all available ports print the same
normalised string). See [docs/matrix.md](docs/matrix.md).

Any Shen project can reuse this: drop a `bifrost.suite.json` and run
`bifrost --suite PATH`. See [docs/suites.md](docs/suites.md) and
[`examples/tiny-suite/`](examples/tiny-suite/).

## Shared port layout

Ports today each invent their own tree (`klambda/` at the root, `kernel/`,
`KLambda/`, StLib in three different places). We are moving them toward one
shape so Bifrost, Yggdrasil, and a human can find the same files everywhere:

```
<port>/
  kernel/klambda/     S42 modules + PROVENANCE.md
  kernel/tests/       official 134-test suite
  kernel/sources/     optional Shen sources
  kernel/lib/         StLib and other Tarver libs
  src/                host-language runtime
  bin/shen            built launcher (or bin/<backend>/shen)
  flake.nix           exports #toolchain
```

This is a **direction**, not a gate yet. Adapters still list explicit
launcher paths. The contract is in [docs/port-layout.md](docs/port-layout.md).

## More

| Topic | Doc |
|-------|-----|
| Commands, pins, Windows | [docs/cli.md](docs/cli.md) |
| Nix `env` and flakes | [docs/nix.md](docs/nix.md) |
| Corpus, `--shake`, adapters | [docs/matrix.md](docs/matrix.md) |
| `bifrost install` (legacy) | [docs/install.md](docs/install.md) |
| Third-party suites | [docs/suites.md](docs/suites.md) |
| Canonical port directories | [docs/port-layout.md](docs/port-layout.md) |

License: Apache-2.0.

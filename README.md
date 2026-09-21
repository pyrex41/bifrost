# Bifrost

One CLI for every Shen port: run a program, eval an expression, or check that
implementations print the same thing.

It is not a Shen runtime. It launches [shen-cl](https://github.com/pyrex41/shen-cl),
[shen-go](https://github.com/pyrex41/shen-go),
[shen-rust](https://github.com/pyrex41/shen-rust),
[shen-lua](https://github.com/pyrex41/shen-lua), and the others the way a user
would, then diffs stdout.

The official 134 kernel tests stay in each port. Bifrost asks a different
question: **do the ports agree?**

## Install

```bash
go install github.com/pyrex41/bifrost@latest
# or
nix run github:pyrex41/bifrost -- impls
```

Default layout is sibling checkouts (`../shen-go`, `../shen-cl`, …). Override
with `BIFROST_PORTS_ROOT`. Nix (`nix run .#env -- all -- bifrost`) is how you
get port toolchains, not just the CLI.

## Use

```bash
bifrost run prog.shen [--impl shen-go]
bifrost eval -e '(+ 2 3)' [--impl shen-rust]
bifrost repl [--impl shen-cl]
bifrost impls --versions
bifrost use shen-go                 # default port
bifrost                             # run the agreement corpus
```

`--impl` wins, then `./.bifrost-impl`, then `~/.config/bifrost/impl`, then the
first available port (prefers `shen-cl`).

## Agreement corpus

```bash
go run .                            # FAIL → exit ≠ 0
go run . --impls shen-cl,shen-go,shen-rust
go run . --only hash-agrees-with-equality,freeze-in-lambda
go run . --heavy                    # also Yggdrasil stage-1 byte-identity
```

~70 cases in `cases/*.json` + `programs/*.shen`: arithmetic, lists, strings,
closures, Prolog, trap-error, and S42 kernel behavior any port should share
(`integer?`, `=`, intern of `"true"`, hash agreeing with `=`, tuples/vectors,
`put`/`get`, freeze captured by a lambda).

A case is a golden `expect: output` or `expect: agreement` (all ports print
the same normalised string). Details: [docs/matrix.md](docs/matrix.md).

Your own project: add `bifrost.suite.json` and run `bifrost --suite PATH`.
See [docs/suites.md](docs/suites.md).

## More

| | |
|---|---|
| CLI | [docs/cli.md](docs/cli.md) |
| Nix | [docs/nix.md](docs/nix.md) |
| Corpus and `--shake` | [docs/matrix.md](docs/matrix.md) |
| Third-party suites | [docs/suites.md](docs/suites.md) |
| Canonical port tree | [docs/port-layout.md](docs/port-layout.md) |

[Yggdrasil](https://github.com/pyrex41/yggdrasil) is the companion shaker
(`bifrost --shake` for stand-alone deploy artifacts).

License: Apache-2.0.

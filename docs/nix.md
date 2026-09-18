# Nix environments

Nix is how Bifrost composes **port toolchains**. Go is only the CLI language.

Each port flake exports `#toolchain`. Bifrost does not copy those dependency
lists; `bifrost env` selects and composes them.

```bash
nix develop                         # Bifrost development only
nix develop ../shen-rust            # one port, directly
nix build                           # the Bifrost binary
nix run .                           # that binary

nix run .#env -- shen-rust -- cargo test
nix run .#env -- shen-go shen-joy -- bifrost bench shen-joy-vs-shen-go
nix run .#env -- all -- bifrost

# same, if `bifrost` is already on PATH:
bifrost env shen-go shen-joy -- bifrost bench shen-joy-vs-shen-go
```

`--` separates port names from the command. With no command, you get a shell.

Sibling checkouts are found under the parent of the active adapter file. Set
`BIFROST_PORTS_ROOT` or `--root DIR` otherwise. Missing flakes, unknown ports,
and missing Nix are explicit errors.

Prebuilt `go install` / GitHub releases do **not** install port toolchains.
Use `bifrost env` for that.

`direnv allow` in this checkout enters the lightweight default shell.
Lefthook uses the same environment for format/test on commit. The live
multi-port matrix stays an explicit `bifrost` run; it is slow on purpose.

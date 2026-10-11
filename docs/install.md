# Legacy `bifrost install`

Prefer [docs/nix.md](nix.md). `bifrost install` mutates global or checkout
state for machines that cannot use Nix.

Each adapter declares a fallback in `adapters.json`:

| method | ports | what runs |
|--------|-------|-----------|
| `brew` | shen-cl / shen-scheme via `--method brew` only | `brew install <formula>` |
| `luarocks` | shen-lua via `--method luarocks` | `luarocks --lua-version=5.1 install shen 0.11.0-1` |
| `git-build` | every port by default | clone (if absent) + the port `build` recipe |

`install` prechecks the toolchain, names the missing tool, refuses
`experimental` ports unless `--force`, skips an already-resolved launcher, and
verifies the launcher afterward.

### Kernel 42.2 transition

Every production port targets the community **ShenOSKernel 42.2**
(`Shen-Language/shen-sources` tag `shen-42.2`, commit `73507e06`, tarball
SHA-256 `1d022730…678d8`), the kernel upstream shen-scheme 0.50 ships. It was
adopted wholesale in place of Tarver's S42.0 (pyrex41/bifrost#26). Until a
port's `kernel-42.2` PR merges to its default branch, install it with
`bifrost install <impl> --ref kernel-42.2`.

The pinned releases below (shen-go `v1.5.0`, shen-lua `v0.11.0` / rock
`0.11.0-1`) predate 42.2 and still report kernel `42`. No 42.2 releases exist
yet; once they are tagged, move the `ref`/`version` pins in `adapters.json`.
Until then a shen-lua launcher or rock that reports `42` is not accepted as
current (the adapter's `kernel` is `42.2`).

### Notes

- **shen-go / ShenScript** git-build clones the **pyrex41 forks** (standard
  `eval -e` / `script` / `--version` CLI). Upstream `tiancaiamao/shen-go` and
  the npm `shen-script` package cannot serve as Bifrost ports. shen-go is
  pinned to the release Bifrost was validated against (`ref` in
  `adapters.json`, currently `v1.5.0`); an existing `../shen-go` clone is
  fetched and checked out to that tag. `bifrost install shen-go --ref master`
  or `BIFROST_SHEN_GO_REF=master` tracks master instead. `v1.5.0` is Tarver
  S42.0; use `--ref kernel-42.2` for 42.2 until a new release is tagged.
- **shen-lua** runs from the checkout (`bin/shen`); the build step warms the
  KLambda cache. Needs `luajit`. The installer pins `v0.11.0` and fetches that
  tag for an existing checkout. The published `0.11.0-1` source rock provides
  the same launcher through LuaRocks for Lua 5.1. Both are still kernel 42
  (Tarver S42.0); use `--ref kernel-42.2` until a 42.2 release and rock are
  published. Bifrost also finds
  a matching `shen` launcher on `PATH`. Set `BIFROST_SHEN_LUA` to the rock's
  launcher when a sibling checkout is present and you want to use the rock.
- **shen-cl** — a fresh clone has no `kernel/` or `compiled/`. Bootstrap once:

  ```bash
  make fetch                       # scripts/fetch-kernel.sh: ShenOSKernel-42.2, SHA-256 checked
  make precompile SHEN=<any working Shen, e.g. a built shen-go>
  bifrost install shen-cl          # make build-sbcl
  ```
- **shen-scheme** — the default is a git-build of the **pyrex41 fork**: upstream
  tizoc/shen-scheme v0.50 (kernel 42.2) merged, plus the two fork fixes the
  corpus asserts (REPL exits on stdin EOF; `*hush*` gates stdout only, so `-q`
  still writes file streams). The build runs the upstream CI recipe:
  `make fetch-kernel`, `make fetch-prebuilt` (tizoc's v0.26 binary, only used
  to precompile the kernel), `make precompile-with-prebuilt`, `make all`
  (downloads and builds Chez 10.3.0). 0.50 has no `shen.boot`: the launcher
  loads `petite.boot`, `scheme.boot` and `lib/shen-scheme/shen-scheme/runtime.so`
  from the `lib/shen-scheme` directory next to `bin/` (`$SHEN_SCHEME_HOME`
  overrides it).
  Upstream 0.50 release binaries run the same kernel; set
  `BIFROST_SHEN_SCHEME=/path/to/shen-scheme-0.50/bin/shen-scheme` to compare.
  Upstream FAILS exactly two cases by design: `hush-file-write` (stock
  `writer.kl` drops file writes under `-q`) and `stdin-eof-clean-exit` (the
  canonical REPL loops at EOF). `--method brew` is not a 42.2 path:
  homebrew-core has no `shen-scheme` formula and the
  `Shen-Language/homebrew-shen` tap ships 0.17.
- **shen-truffle** — the launcher needs JDK 23+ (it passes
  `--sun-misc-unsafe-memory-access=allow`; its flake uses GraalVM CE 25) via
  `$JAVA_HOME` or `java` on `PATH`. A set `JAVA_TOOL_OPTIONS` makes the JVM
  print `Picked up JAVA_TOOL_OPTIONS: …` on stderr; Bifrost's normaliser drops
  that line, but unset the variable when running the launcher by hand.

### Fresh machine (no Nix)

Defaults assume sibling checkouts. Override with a project `adapters.json` or
`$BIFROST_ADAPTERS`.

1. `bifrost install shen-go` — `git` + `go`; gives a Shen for step 3.
2. `shen-rust` (`cargo`), `ShenScript` (`node`/`npm`), `shen-lua` (`luajit`).
3. shen-cl: clone, `make fetch`, `make precompile SHEN=<shen-go>`, then
   `bifrost install shen-cl` (`sbcl`).
4. `shen-scheme` (git-build of the fork; `make` + C compiler + `curl`),
   `shen-julia`, `shen-swift`, `shen-truffle`.

Forks: point `$BIFROST_SHEN_CL` at a built binary, or
`bifrost install shen-cl --git URL --ref BRANCH`.

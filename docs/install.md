# Legacy `bifrost install`

Prefer [docs/nix.md](nix.md). `bifrost install` mutates global or checkout
state for machines that cannot use Nix.

Each adapter declares a fallback in `adapters.json`:

| method | ports | what runs |
|--------|-------|-----------|
| `brew` | shen-scheme (shen-cl via `--method brew`) | `brew install <formula>` |
| `luarocks` | shen-lua via `--method luarocks` | `luarocks install shen` |
| `git-build` | the rest | clone (if absent) + the port `build` recipe |

`install` prechecks the toolchain, names the missing tool, refuses
`experimental` ports unless `--force`, skips an already-resolved launcher, and
verifies the launcher afterward.

### Notes

- **shen-go / ShenScript** git-build clones the **pyrex41 forks** (standard
  `eval -e` / `script` / `--version` CLI). Upstream `tiancaiamao/shen-go` and
  the npm `shen-script` package cannot serve as Bifrost ports. shen-go is
  pinned to the release Bifrost was validated against (`ref` in
  `adapters.json`, currently `v1.5.0`); an existing `../shen-go` clone is
  fetched and checked out to that tag. `bifrost install shen-go --ref master`
  or `BIFROST_SHEN_GO_REF=master` tracks master instead.
- **shen-lua** runs from the checkout (`bin/shen`); the build step warms the
  KLambda cache. Needs `luajit`.
- **shen-cl** — a fresh clone has no `kernel/` or `compiled/`. Bootstrap once:

  ```bash
  make fetch
  make precompile SHEN=<any working Shen, e.g. a built shen-go>
  bifrost install shen-cl          # make build-sbcl
  ```

### Fresh machine (no Nix)

Defaults assume sibling checkouts. Override with a project `adapters.json` or
`$BIFROST_ADAPTERS`.

1. `bifrost install shen-go` — `git` + `go`; gives a Shen for step 3.
2. `shen-rust` (`cargo`), `ShenScript` (`node`/`npm`), `shen-lua` (`luajit`).
3. shen-cl: clone, `make fetch`, `make precompile SHEN=<shen-go>`, then
   `bifrost install shen-cl` (`sbcl`).
4. `shen-scheme --method git-build`, `shen-julia`, `shen-swift`, `shen-truffle`.

Forks: point `$BIFROST_SHEN_CL` at a built binary, or
`bifrost install shen-cl --git URL --ref BRANCH`.

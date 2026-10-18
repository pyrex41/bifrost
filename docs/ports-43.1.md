# Shen 43.1 adapter qualification

The maintained adapters now declare community ShenOSKernel 43.1. The source
is `Shen-Language/shen-sources`, tag `shen-43.1`, commit
`715824a52b55e9c81a9501df8548ff3edd8411f5`; archive SHA-256
`900eedef44f440af78f9395ed06e689ce1604546fc309f5bc7c8bc84cc651920`.
Shen-C v0.3.0 has its separately maintained S43 lineage and RC backend.
Joy declares `bounded-43.1-subset`; it does not execute a full kernel.

The new exact 43.1 version cases reject stale launchers. The historical 42.2
cases remain kernel-filtered and skip when no configured adapter matches.
A verified 42.2 bootstrap fails both 43.1 probes as an intentional control.

Go and Lua git installations pin reviewed 43.1 commits instead of historical
release tags. No new release or LuaRocks publication is implied. The retained
historical LuaRocks method does not install 43.1; use git-build. Other git
adapters build their maintained fork default branches.

## Local matrix, 2026-10-18

| Matrix | Executed rows | PASS | Known DIVERGE | FAIL | Unmatched 42.2 rows |
|---|---:|---:|---:|---:|---:|
| CL, Go, Rust, Lua, Scheme, JS, Swift, Truffle | 72 | 70 | 2 | 0 | 2 |
| CL, Erlang (Docker) | 72 | 72 | 0 | 0 | 2 |
| CL, Swift | 72 | 71 | 1 | 0 | 2 |
| CL, Truffle | 72 | 71 | 1 | 0 | 2 |
| Old CL 42.2 control, 43.1 probes only | 2 | 0 | 0 | 2 expected | 0 |

Divergences are the existing host integer precision buckets. The JSON reporter
labels unmatched kernel rows PASS with `skipped: no matching kernels`; the
table excludes them from executed/pass counts. This avoids presenting 74
reported rows as 74 executed checks.

[Machine receipts and compressed results](evidence/ports-43.1/receipt.json)
retain per-port outcomes. Julia and bounded-target qualification are recorded separately; their
absence here is not a pass. This matrix checks CLI agreement and does not
replace each port's canonical 154-case release suite or the Yggdrasil artifact
gates. Native C results remain in the separate integration evidence.

## Reproduce

Build this checkout, provide the selected launchers through the adapter env
variables, and run:

```sh
go test ./...
go build -o bifrost .
./bifrost --impls shen-cl,shen-go,shen-rust,shen-lua,shen-scheme,ShenScript --json
```

Use `BIFROST_SHENSCRIPT` for the JS launcher, LuaJIT on PATH for Lua,
`JAVA_HOME` for Truffle, and `SHEN_JULIA_NO_SYSIMAGE=1` after a Julia kernel
upgrade until its sysimage is rebuilt. For a kernel comparison, the configured
version probe must report the actual kernel rather than the port release.

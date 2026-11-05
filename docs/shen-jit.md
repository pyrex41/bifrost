# Shen/JIT

The `shen-jit` adapter drives [tizoc/shen-jit](https://github.com/tizoc/shen-jit), a
port that compiles every function with a baseline JIT, on community ShenOSKernel 43.2.
Its version cases are `version-expr-43.2` and `version-probe-43.2`, which other
adapters skip by kernel.

```sh
bifrost install shen-jit           # or set BIFROST_SHEN_JIT to a release's executable
bifrost eval -e '(version)' --impl shen-jit
bifrost --impls shen-jit,shen-go
bifrost build prog.shen --target jit --run
```

`bifrost --shake` and `bifrost build` go through Yggdrasil, whose kernel is also 43.2,
so its stage-1 host is Shen/JIT whatever port is the target. The `--heavy`
`yggdrasil-shake-parity` case runs only on adapters declaring 43.2 for the same reason.

## Local matrix, 2026-11-05

macOS arm64, Shen/JIT at `715bf60` (v0.2 and three later commits), reproducible with:

```sh
go build -o bifrost . && ./bifrost --impls shen-jit,shen-cl,shen-go,shen-lua,ShenScript --json
```

| Rows | Count |
|---|---:|
| Executed PASS | 71 |
| Known DIVERGE: the two integer precision buckets and `hush-file-write` | 3 |
| FAIL | 0 |
| Unmatched 42.2 rows, reported PASS with `skipped: no matching kernels` | 2 |

The ports are shen-jit, shen-cl, shen-go, shen-lua and ShenScript, the ones built on
that machine. On its own, shen-jit ran 70 of the then 74 rows with no FAIL. This checks
CLI agreement, not the kernel's own suite, which is Shen/JIT's CI.

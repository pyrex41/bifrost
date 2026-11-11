# Shen 43.4 adapter qualification

The maintained adapters declare community ShenOSKernel 43.4:
`Shen-Language/shen-sources`, tag `shen-43.4`, commit `f34b307b`. Shen-joy declares
`bounded-43.4-subset`, as it does not execute a full kernel; its differential reference is
Shen/CL v3.3.0. Yggdrasil shakes on 43.4 too, so `yggdrasil-shake-parity` now runs on every
full port. The [43.2 record](ports-43.2.md) and the [43.1 record](ports-43.1.md) are kept for
those kernels.

## Result

On 2026-11-09, macOS arm64, Shen/JIT v0.5 and Yggdrasil v0.2.0, in four runs. The
`--heavy` run covered 11 implementations. shen-julia and shen-truffle errored there, as julia
and java were not on `PATH`, so each was run again beside Shen/JIT and shen-go. shen-erl
was run beside Shen/JIT once its 43.4 branch merged.

| Run | Corpus result |
|---|---|
| `--heavy`, the nine other full ports and Shen/JIT | PASS, except the known divergences and `float-add-imprecise` on shen-c |
| `--heavy`, shen-truffle, Shen/JIT, shen-go | 77 PASS, 2 known DIVERGE |
| shen-julia, Shen/JIT, shen-go (`SHEN_JULIA_NO_SYSIMAGE=1`) | 76 PASS, 2 known DIVERGE |
| shen-erl, Shen/JIT | 76 PASS, 2 known DIVERGE |

The known divergences are `hush-file-write` and the integer precision buckets above 2^53
and 2^63. shen-c prints `(+ 0.1 0.2)` as `0.3`, as it did on 43.2, and every other port
prints `0.30000000000000004`.

`yggdrasil-shake-parity` shook `tests/fib.shen` on ten hosts: ShenScript, shen-c, shen-cl,
shen-go, shen-jit, shen-lua, shen-rust, shen-scheme and shen-swift in the first run, and
shen-truffle in the second. `kernel.kl` and the manifest were byte-identical on all ten
(md5 `4cff4a2a` and `d27cc480`). shen-julia is not a host without `BIFROST_PARITY_JULIA=1`,
as it is too slow with no sysimage. shen-erl was not run as a host.

Yggdrasil's parity gate, with Shen/JIT as host, passed all ten fixtures on the `c`, `go`,
`jit`, `js`, `julia`, `kl`, `lisp`, `lua`, `rust`, `scheme`, `swift` and `truffle` targets,
apart from the capability SKIPs each declares. The `erlang` target passed the seven fixtures
it reached before the run was stopped.

## Refs

| Adapter | Release | Commit |
|---|---|---|
| shen-jit | v0.5 | `246e4f9` |
| shen-cl | v3.3.0 | `b4759cd` |
| shen-go | v1.7.0 | `b50c301` |
| shen-lua | v0.13.0 | `9adc64f` |
| ShenScript | v1.1.0-rc.3 | `272aa97` |
| shen-scheme | upstream 0.54 plus two fixes | `ea8e14e` |
| shen-julia | v0.4.0 | `6eb10cf` |
| shen-c | v0.6.0 | `9c6de31` |
| shen-erl | none | `eb16e95` |
| shen-joy | none | `9daee49` |

shen-go's pin has since moved to v1.8.0 (`4b91705`). On 2026-11-09, `--heavy` with it and
Shen/JIT gave 77 PASS and the same two known divergences, and the shake was byte-identical
(md5 `4cff4a2a` and `d27cc480`).

shen-swift and shen-truffle were run at their merged 43.4 commits. They cut no
releases, and their adapters build the default branch.

shen-rust was run the same way, then pinned to v0.1.1 (`a653699`). On 2026-11-10,
Linux x86_64, that release binary passed the corpus alone: 77 PASS and the known
`hush-file-write` divergence, with shen-rust still in the `written` bucket.

The pin is now v0.1.2 (`c8e3f0a`). `(port)` and `(release)` report `0.1.2`;
`*version*` stays `43.4`. The build recipe is unchanged: `cargo build --release
--bin shen-rust`. Default features are `cedar`, `kernel-aot`, `interpreter`, and
`sha256`. v0.1.2 registers `value/or` before `shen.initialise`. On 2026-11-11,
Linux x86_64, that release binary passed the corpus alone: 77 PASS and the known
`hush-file-write` divergence, with shen-rust still in the `written` bucket.
`bifrost impls --versions` reports kernel `43.4`.

shen-cl's v3.3.0 has no Windows archive, as its 64MB stack default does not take effect
there. Its macOS arm64 archive was built from the tag outside CI, as GitHub had no macOS
runner free; the release notes say so.

CI does not build the other ports, so these numbers come from that one machine.

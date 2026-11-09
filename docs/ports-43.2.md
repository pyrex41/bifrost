# Shen 43.2 adapter qualification

The maintained adapters declare community ShenOSKernel 43.2:
`Shen-Language/shen-sources`, tag `shen-43.2`, commit `e6947f507`. Shen-joy declares
`bounded-43.2-subset`, as it does not execute a full kernel. Shen-scheme has since moved to
43.4 with upstream 0.54, and Yggdrasil to 43.4 in its v0.2.0, so `yggdrasil-shake-parity`
now runs on the 43.4 hosts, Shen/JIT and shen-scheme. The result below predates both. The older
[43.1 record](ports-43.1.md) is kept for that kernel.

## Result

`--heavy` on 2026-11-06, macOS arm64, with 12 implementations:

| Rows | Count |
|---|---:|
| PASS, including rows skipped by kernel | 73 |
| Known DIVERGE: `hush-file-write` and the two integer precision buckets | 3 |
| FAIL: `float-add-imprecise` on shen-c | 1 |

`yggdrasil-shake-parity` ran on 11 hosts and the shake was byte-identical. shen-julia is
not a host in that case without `BIFROST_PARITY_JULIA=1`, as it is too slow with no
sysimage. shen-julia ran with `SHEN_JULIA_NO_SYSIMAGE=1` throughout.

The failure predates 43.2. shen-c prints `(+ 0.1 0.2)` as `0.3`, and every other port
prints `0.30000000000000004`. It stays a FAIL until shen-c's float printing is fixed.

## Refs

| Adapter | Release | Commit |
|---|---|---|
| shen-jit | v0.2, since moved to v0.5 on kernel 43.4 | `109f347` |
| shen-cl | v3.2.1 | `120f387` |
| shen-go | v1.6.0 | `7509c6b` |
| shen-lua | v0.12.0 | `912596b` |
| ShenScript | v1.1.0-rc.2 | `6761d05` |
| shen-scheme | v0.52-pyrex41.6, since moved to upstream 0.54 plus two fixes | `0d9220d` |
| shen-julia | v0.3.0 | `ef5b552` |
| shen-c | v0.5.0 | `b7ebcea` |
| shen-rust | none | `ce9037b` |
| shen-swift | none | `6e1384b` |
| shen-erl | none | `d8f1c7b` |
| shen-truffle | none | `1997bd0` |

shen-cl's v3.2.1 has no Windows archive, as its 64MB stack default
does not take effect there.

CI does not build the other ports, so
these numbers come from that one machine.

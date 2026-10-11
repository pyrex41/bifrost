# Agreement matrix

```bash
go run .                           # light cases; FAIL → exit ≠ 0
go run . --heavy                   # + Yggdrasil stage-1 parity
go run . --list
go run . --only int-mul,float-add-imprecise
go run . --impls shen-cl,shen-go
go run . --json
go run . --shake --only recursion-fib-file
```

`DIVERGE` rows (tagged `known_divergence`) are reported and do **not** fail.
Only `FAIL` sets a non-zero exit. A divergence case may carry a `table` that
pins each port to its documented bucket; a port that leaves its bucket is a
real `FAIL`. The open divergences are platform number precision
(`bigint-2pow53-plus-1`, `bigint-2pow63-ordered`), which Shen leaves to the
host; every other tracked difference is a hard agreement.

## Kernel

Every production port runs the community **ShenOSKernel 42.2**
(`Shen-Language/shen-sources` tag `shen-42.2`, commit `73507e06`; the kernel of
upstream shen-scheme 0.50), adopted wholesale in place of Tarver's S42.0
(pyrex41/bifrost#26). Cases that used to split by kernel (`error-message-head`)
are hard agreements again.

## Corpus

`cases/*.json` driven by `programs/*.shen`:

- Arithmetic (including floats), lists, strings, closures, `fix`, recursion,
  a 100 000-deep tail-call countdown, a small Prolog query, `trap-error`.
- Kernel portables: `integer?`/`symbol?`/`boolean?`/`empty?`/`variable?`/`element?`,
  `=` of `1`/`1.0` and `true`/`(intern "true")`, hash agreeing with `=`,
  `@p`/`vector`, `put`/`get` on `*property-vector*`, freeze captured by a
  lambda, trap-error whose *operand* raises, the `(head [])` error message.
- CLI: `eval -e` prints the value; `(version)` / `--version` is kernel
  **42.2**; stdin-EOF exits cleanly.
- 42.1/42.2 behaviour (`cases/kernel-42.json`): `read` and `lineread` on a
  file that ends without a trailing newline; source-form handlers running
  before ordinary macros; namespace `externals` given an expression.
- Closed divergences (now asserted): `float-formatting`, `int-div-zero`,
  `hush-file-write`, `load-toplevel-echo`.
- Documented divergences: integers above 2^53 and 2^63 (see below).
- `--heavy`: `(yggdrasil.shake ["tests/fib.shen"] OUT)` on every host; the
  produced `kernel.kl` + `yggdrasil.manifest` must be byte-identical. User KL
  differs by gensym and is not asserted.

Each case is `expect: output` (golden) or `expect: agreement` (all impls
match).

### Capability SKIP

A launcher that exits **3** marks its cell `SKIP` (shen-joy for anything
outside its subset). Shen has no portable `exit`, so a corpus program that
needs an optional facility probes for it and, when it is missing, prints a
line starting with `BIFROST-SKIP:`; Bifrost treats that like exit 3. The
source-form and namespace cases use this: ports that do not boot
`shen.register-source-form` or `extension-namespaces.kl` SKIP instead of
failing. At the time of writing only shen-scheme and shen-rust boot the
namespaces extension.

### Integer precision (documented divergence)

Shen leaves number representation to the platform, so these cases report
buckets instead of asserting agreement:

| case | expression | buckets |
|------|------------|---------|
| `bigint-2pow53-plus-1` | `(+ 9007199254740993 0)` | `9007199254740993`: shen-cl, shen-erl, shen-scheme (bignums); shen-julia, shen-c, shen-rust, shen-swift, shen-truffle (64-bit integers). `9007199254740992`: shen-go, ShenScript, shen-lua (IEEE doubles only) |
| `bigint-2pow63-ordered` | `(> (+ 9223372036854775807 1) 9223372036854775807)` | `true`: shen-cl, shen-erl, shen-scheme (bignums). `false`: shen-julia, shen-c (wrap to -2^63); shen-rust, shen-swift, shen-truffle (overflow to a double equal to 2^63); shen-go, ShenScript, shen-lua (doubles) |

The printed 2^63 sum also differs by host float formatting
(`9223372036854776000`, `9.223372036854776e+18`, `9.223372036854776E18`), so
the second case compares a boolean.

### Upstream shen-scheme 0.50 reference

Pointing `BIFROST_SHEN_SCHEME` at an upstream tizoc 0.50 binary runs the same
kernel. It FAILS exactly two cases, by design: `hush-file-write` (stock
`writer.kl` silences file-stream `pr` under `-q`) and `stdin-eof-clean-exit`
(the canonical REPL loops on EOF; Bifrost caps captured output at 8 MB). The
pyrex41 fork (the default `shen-scheme` install) reapplies both fixes on top of
0.50.

### Known port gaps

These FAILs are port bugs, not intended differences; each was already present
on the port's Tarver S42.0 main branch:

- shen-c (experimental): `version-banner-contains` (port-local `Shen-C 0.2.3`
  banner), `float-add-imprecise` (prints `0.3`), and script mode echoing each
  toplevel value like `load` instead of the launcher's quiet load
  (`bounded-tail-sum-file`, `source-form-before-macros`).
- shen-julia: native `integer?` is false for `4.0`, and `hash` of `1` and `1.0`
  differ (`integer-float-integral`, `hash-agrees-with-equality`).

### Normaliser

Before comparing, Bifrost drops the `run time:` banner (with the blank line its
leading newline produces), `(fn NAME)` load echoes, shen-lua's trailing value
echo, JVM `Picked up JAVA_TOOL_OPTIONS: …` notices (proxies and CI images set
that variable), and leading/trailing/repeated blank lines.

## `--shake` (deploy path)

Runs each **script-mode** program through Yggdrasil: shake once, build a
stand-alone artifact per target, run, diff. Missing toolchains SKIP.

Artifacts map: lisp→`shen-cl`, lua→`shen-lua`, go→`shen-go`, joy→`shen-joy`,
erlang→`shen-erl`, rust→`shen-rust`, js→`ShenScript`, julia→`shen-julia`,
scheme→`shen-scheme`, swift→`shen-swift`, truffle→`shen-truffle`. Joy is a
bounded image: unsupported KLambda exits 3 and is SKIP. The one corpus program
inside its subset is `programs/joy-sum.shen` (`bounded-tail-sum-file`, the same
program as Yggdrasil's `tests/joy-sum.shen`): its joy artifact must print
`32004000`. `tests/fib.shen` and the other script programs are capability SKIPs
on joy. The 42.1/42.2 script cases (`source-form-before-macros`,
`namespace-externals-expression`) use `eval` and `read-from-string` at run time,
so they are not meant for `--shake`; run `--shake` with `--only` on the
first-order programs.

Needs `$BIFROST_YGGDRASIL_DIR` (default `../yggdrasil`) or
`$BIFROST_YGGDRASIL_BIN`. This mode is minutes: Go/Rust compile from scratch;
Julia AOT-bakes a sysimage.

## Adding a case

1. Optional `.shen` in `programs/`. End with `(do (print EXPR) (nl))` so
   shen-lua's load echo does not pollute the golden.
2. An entry under `cases/`:

   ```json
   {
     "name": "my-case",
     "mode": "eval",
     "expr": "(+ 2 3)",
     "expect": "output",
     "golden": "5",
     "doc": "what this checks"
   }
   ```

3. `go run . --only my-case`.

## Narrow benchmark

`bench shen-joy-vs-shen-go` times the shared first-order self-tail-recursive
`sum-mid(0, 8000)` kernel. Startup, parse, and boot sit outside both timed
regions. Markdown by default; `--json`; `--output FILE`. It is not a
full-Shen score.

```bash
nix run .#env -- shen-go shen-joy -- go run . bench shen-joy-vs-shen-go \
  --samples 10 --iterations 500 --benchtime 500ms
```

## CI

[`.github/workflows/ci.yml`](../.github/workflows/ci.yml) is best-effort.
Missing impls are skipped. `go test ./...` covers the corpus, adapters,
routing, and Windows/Linux/macOS helpers.

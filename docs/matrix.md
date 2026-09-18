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
Only `FAIL` sets a non-zero exit. There are currently **no** open divergences;
every tracked difference is now a hard agreement.

## Corpus

`cases/*.json` driven by `programs/*.shen`:

- Arithmetic (including floats), lists, strings, closures, `fix`, recursion,
  a 100 000-deep tail-call countdown, a small Prolog query, `trap-error`.
- CLI: `eval -e` prints the value; `(version)` / `--version` is kernel **42**;
  stdin-EOF exits cleanly.
- Closed divergences (now asserted): `float-formatting`, `int-div-zero`,
  `hush-file-write`, `load-toplevel-echo`.
- `--heavy`: `(yggdrasil.shake ["tests/fib.shen"] OUT)` on every host; the
  produced `kernel.kl` + `yggdrasil.manifest` must be byte-identical. User KL
  differs by gensym and is not asserted.

Each case is `expect: output` (golden) or `expect: agreement` (all impls
match).

## `--shake` (deploy path)

Runs each **script-mode** program through Yggdrasil: shake once, build a
stand-alone artifact per target, run, diff. Missing toolchains SKIP.

Artifacts map: lisp→`shen-cl`, lua→`shen-lua`, go→`shen-go`, joy→`shen-joy`,
erlang→`shen-erl`, rust→`shen-rust`, js→`ShenScript`, julia→`shen-julia`,
scheme→`shen-scheme`, swift→`shen-swift`, truffle→`shen-truffle`. Joy is a
bounded image: unsupported KLambda exits 3 and is SKIP.

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

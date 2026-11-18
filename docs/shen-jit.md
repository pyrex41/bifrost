# Shen/JIT

The `shen-jit` adapter drives [tizoc/shen-jit](https://github.com/tizoc/shen-jit), a
port that compiles every function with a baseline JIT, on community ShenOSKernel 43.6.
Its version cases are `version-expr-43.6` and `version-probe-43.6`.

```sh
bifrost install shen-jit           # or set BIFROST_SHEN_JIT to a release's executable
bifrost eval -e '(version)' --impl shen-jit
bifrost --impls shen-jit,shen-go
```

The `jit` build target needs Shen/JIT's source tree as well, so a release's executable
is not enough. Use `bifrost install shen-jit`, or set `YGGDRASIL_SHEN_JIT_DIR` to a
checkout or a release's source tarball, then:

```sh
bifrost build prog.shen out/ --target jit --run
```

`bifrost --shake` and `bifrost build` go through Yggdrasil. In the updated Yggdrasil checkout, its
kernel is 43.6 and Shen/JIT v0.8 is its default stage-1 host; the shaker refuses a host
on any other kernel. The `--heavy` `yggdrasil-shake-parity` case therefore runs on the
adapters declaring 43.6. A program that redefines a function
is a capability SKIP on `jit`, as is `--shake` on a machine with no 43.6 host.

## Results

The [43.4 qualification](ports-43.4.md) has the run across every port, made on v0.5. CI
builds v0.8 from its source tarball and runs the corpus and the agent skills' examples on
it, on Ubuntu 24.04. On 2026-11-09, on macOS arm64 with Yggdrasil v0.2.0,
`yggdrasil-shake-parity` was byte-identical on Shen/JIT and shen-scheme, before the other
ports moved to 43.4, and `--shake`
passed on the `jit` and `go` targets.

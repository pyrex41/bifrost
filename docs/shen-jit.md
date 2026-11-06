# Shen/JIT

The `shen-jit` adapter drives [tizoc/shen-jit](https://github.com/tizoc/shen-jit), a
port that compiles every function with a baseline JIT, on community ShenOSKernel 43.2.
Its version cases are `version-expr-43.2` and `version-probe-43.2`.

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

`bifrost --shake` and `bifrost build` go through Yggdrasil, whose kernel is also 43.2,
so its stage-1 host is Shen/JIT whatever port is the target. The `--heavy`
`yggdrasil-shake-parity` case runs only on adapters declaring 43.2 for the same reason.
With one such host, that case compares nothing, and says so. A program that redefines a function is a capability SKIP on `jit`, as is
`--shake` on a machine with no 43.2 host.

## Results

The [43.2 qualification](ports-43.2.md) has the run across every port. CI builds v0.2 from
its source tarball and runs the corpus on it alone, on Ubuntu 24.04.

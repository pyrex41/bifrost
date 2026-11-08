# Shen/JIT

The `shen-jit` adapter drives [tizoc/shen-jit](https://github.com/tizoc/shen-jit), a
port that compiles every function with a baseline JIT, on community ShenOSKernel 43.4.
Its version cases are `version-expr-43.4` and `version-probe-43.4`.

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

`bifrost --shake` and `bifrost build` go through Yggdrasil, whose kernel is 43.2, and
the shaker refuses a host on any other kernel. Since the adapter moved to v0.5 on 43.4,
Shen/JIT is no longer a stage-1 host: one of the 43.2 ports (shen-go, shen-cl and the
others) is. The `--heavy` `yggdrasil-shake-parity` case runs only on adapters declaring
43.2 for the same reason. The `jit` build target also goes through Yggdrasil, and has
not been rechecked with a v0.5 tree. A program that redefines a function is a capability SKIP on `jit`, as is
`--shake` on a machine with no 43.2 host.

## Results

The [43.2 qualification](ports-43.2.md) has the run across every port, made on v0.2. CI
builds v0.5 from its source tarball and runs the corpus and the agent skills' examples on
it, on Ubuntu 24.04.

# Shen-C native RC integration

Base: `b0dee048`. Work is isolated on `fix/shen-c-native-rc`.

1. Pin Shen-C v0.3.0 and declare its actual bundled kernel, 43.1, separately
   from the community 42.2 adapters. Remove obsolete Boehm/pkg-config needs.
2. Probe the kernel using `(version)` rather than interpreting a port-release
   banner as a kernel version. Preserve CLI commands and launcher overrides.
3. Select exact version goldens by kernel. Update the documented numeric
   overflow bucket to the current catchable error; keep unrelated corpus
   failures visible instead of normalizing them away.
4. Run Go tests, the live Shen-C corpus and the C artifact path through the
   updated Yggdrasil builder. Record source/binary/toolchain identities and
   outstanding runtime or platform gaps. Other adapters stay on their existing
   kernels; this is integration support, not a shared-kernel migration.

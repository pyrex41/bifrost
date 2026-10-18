# Bifrost Shen-CL/Julia 43.1 final matrix

Run date: 2026-10-18. The matrix used the pinned Shen-CL 43.1 launcher and
`shen-julia-upgrade-43.1/bin/shen` with Julia 1.12.1 and
`SHEN_JULIA_NO_SYSIMAGE=1`.

- 74 corpus rows reported
- 72 rows executed
- 2 historical 42.2 rows skipped by kernel targeting
- 71 executed rows passed
- 1 known divergence: `bigint-2pow63-ordered` (Julia's documented 64-bit integer wrap)
- 0 executed failures

The earlier pre-fix receipt is retained in
`bifrost-431-cl-julia-before-numeric-fix.json.gz`; it
recorded the two Julia numeric failures in `integer-float-integral` and
`hash-agrees-with-equality`. The final run passes both rows. The portable JSON
receipt in this directory contains only the machine-readable matrix object.

Julia implementation: `9655446d9f67561cfbbf7336635869b673b471c9`;
CL implementation: `d9494dfb4e64000d4213d4725bd8c503cfaacab1`.

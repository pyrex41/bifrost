# Shen skills checker audit

## Scope and baseline

Checker-only refinement of `183e71e041e91740dc33c828f674823dd9815fcc` (the supplied cherry-pick of
`6b04a83` onto fetched upstream `51a8f0d`). Owned changes: `skills_check.go`, `skills_test.go`,
`skills/README.md`, and this audit. Skill/reference edits belong to the concurrent content workstream;
consolidated claim review and agent trials belong to the parent. No port sources or adapters changed,
no ports rebuilt, and no repository commit/push/checkout/reset performed by this workstream.

Before editing, copied the skills to `/tmp/bifrost-skills-drive/checker-baseline-skills` and ran the
existing checker. Full baseline log: `/tmp/bifrost-skills-drive/sol-baseline.log`, begun
2026-11-10 22:05:11 UTC. This is a fresh observation, not the historical 57-block claim.

Environment: macOS arm64; `go version go1.27.3 darwin/arm64`. Inspected `adapters.json`, sibling
projects, launcher files, environment overrides, and `go run . impls --versions` before choosing ports.
Actual live versions, not merely the adapters' declared baseline:

| Runtime | Resolved executable | Live version |
|---|---|---|
| Shen/JIT | `<HOME>/projects/shen-jit/build/shen-jit` | `43.4 (port ("native" "0.5") implementation ("shen-jit" "0.5"))` |
| Shen Scheme | `<HOME>/projects/shen-scheme/_build/bin/shen-scheme` | `43.4 (port ("Scheme" "0.54") implementation ("chez-scheme" "10.3.0"))` |

Other installed launchers were not interchangeable 43.4 evidence: CL reported 43.1; the inherited
`BIFROST_SHEN_GO=<HOME>/.local/opt/shen-go-bin` reported 43.2; Lua reported 43.2; ShenScript
reported 43.1; Julia's probe did not establish a version. The default active implementation was CL,
so all documentation checks explicitly selected JIT or Scheme. Missing ports were not built.

Baseline commands/results:

- `go test ./...`: PASS.
- `go run . skills check --impl shen-jit --dir /tmp/bifrost-skills-drive/checker-baseline-skills`:
  **57 blocks, 0 failed**.
- Same command with `--impl shen-scheme`: **57 blocks, 0 failed**.

## Observed weaknesses and dispositions

Pre-edit probes and complete output: `/tmp/bifrost-skills-drive/sol-probes-baseline.log`.
The five-block probe deliberately included invalid examples; its aggregate command exited 1 because
of the runtime assertion, despite four reported passes.

| Contract / finding | Baseline observation | Current implementation and regression evidence |
|---|---|---|
| Intended definition | `reject: intended` incorrectly passed for `type error in rule 1 of unintended` | Whole-whitespace-token diagnostic matching; wrong-function, prefix and suffix cases fail. Literal punctuation and full diagnostic phrases remain supported. |
| Undefined helper | `(define intended {number --> number} X -> (absent-helper X))` incorrectly passed its named rejection: Shen reports the type error at `intended` | After a failed negative load, inspect reader-expanded forms for unresolved `fn` and zero-argument function references using arity metadata. Real-port tests reject missing unary and zero-argument helpers. |
| Malformed annotations | `<!-- rejeect: intended -->` was silently treated as an accepted block | Fail closed on unknown/malformed, stacked, detached, dangling, and non-Shen-fence directives. No new annotation syntax added. |
| Typed setup before load | Existing expression enabled checking before initial setup; a typed block could leave checking disabled for the next load | Every typed load now explicitly enables `(tc +)`. Tests cover invalid accepted bodies, setup outside the rejection trap, `(tc -)` left by preceding code, and untyped restoration. |
| Setup failure is not rejection | Already outside the rejection trap | Preserve that boundary; real type-error and runtime-error setup fixtures fail instead of satisfying the negative case. Same-block wrong-definition diagnostics also fail. |
| Runtime assertions | Failing assertion already failed with JIT exit 1 | Preserve behavior; real-port valid/mismatched assertions and runtime-error-is-not-type-error cases. No expected-output annotation language. |
| Process failure / timeout | Existing judge checked process outcome first | Preserve ordering; unit cases and actual subprocesses print both success markers, then exit 7 or hang. Both accept and reject judgments fail; timeout retains partial output. |
| Diagnostic / marker permissiveness | Judge accepted marker substrings and any diagnostic containing `type error` | Require standalone marker lines and the observed `type error in rule N of NAME` diagnostic shape with a positive numeric rule. Reject incidental text, partial markers and invalid rule numbers. |
| Rejected definitions / fresh contexts | Accumulation already excluded rejects and reset on `fresh` | Preserve; synthetic accumulation and actual runtime checks show rejected definitions unavailable later and `fresh` losing earlier definitions. Each check remains a separate process. |
| Reference independence / recursion | Recursive Markdown discovery and per-file accumulation already existed | Deep nested fixture, ignored non-Markdown file, independent references, skipped blocks, reject exclusion, and reset-plus-subsequent-accumulation tests. |
| Packaging | Existing tests mostly checked file existence | Compare complete source, embedded and installed file sets and bytes for every skill, normal and raw installation. Synthetic deep nested reference also byte-matches. Execute the installed suite through the CLI. |

Implementation details and limits matter here. Kernel 43.4's generated `shen.parse-failure` call works
but has arity `-1`; narrowly exempt that internal name rather than disabling zero-argument validation.
Named diagnostic matching alone cannot distinguish an absent helper from a genuine mismatch. The
reader-form check addresses that demonstrated ambiguity without building a second Shen parser in Go.

The additional read occurs only after a failed negative load, outside the protected load operation:
a reader/validation failure cannot become a successful rejection. It does not carry any rejected
state into another check. It can expand macros again, so negative examples must avoid read-time side
effects. This requirement and the arity trust boundary are explicit in `skills/README.md`.

## Development failures and rejected assumptions

These intermediate failures were not hidden or counted as final evidence:

- `/tmp/bifrost-skills-drive/sol-first-pass.log`: Go tests failed two old annotation-error location
  expectations (the parser now reports the annotation line). Initial validator traversed list tails
  as independent forms and misclassified terminal type symbols as zero-argument calls: **57 blocks,
  8 failed on JIT**. Fixed by mapping over actual list elements; updated error-location assertions.
  The combined malformed-annotation CLI probe correctly stopped before executing blocks.
- `/tmp/bifrost-skills-drive/sol-second-pass.log`: **57 blocks, 2 failed on each runtime**. Both
  failures were grammar negatives, due to the generated `shen.parse-failure` metadata omission
  (JIT exit 1, Scheme exit 255). Added the specific documented exception. Baseline then passed on both.
- `/tmp/bifrost-skills-drive/sol-late-helper.log`: raw kernel experiment with a typed helper later
  in the same loaded file **succeeded**, contrary to a tentative concern about forward setup. Shen's
  reader makes those typed declarations available. No single-definition restriction was retained;
  final tests positively preserve same-block forward helpers and accepted same-block setup.
- `/tmp/bifrost-skills-drive/sol-validation-1.log` and `sol-current.log`: successful intermediate
  checks, not substitutes for the final run below. An intermediate adversarial log predates the final
  forward-helper control adjustment; use `sol-final.log` for the final 44 real-port fixture outcomes.

## Final verification

Full commands, exit statuses and output: `/tmp/bifrost-skills-drive/sol-final.log`.
Built artifact outside the checkout: `/tmp/bifrost-skills-drive/bifrost-checker`.
All commands below exited **0**:

| Command / input | Result |
|---|---|
| `go test -count=1 ./...` | PASS (includes real failed/timed-out subprocess tests and byte comparisons; optional Shen tests skipped in this invocation) |
| `go vet ./...` | PASS |
| `go build -o /tmp/bifrost-skills-drive/bifrost-checker .` | PASS |
| `BIFROST_SKILLS_TEST_IMPLS=shen-jit,shen-scheme go test -count=1 -run '^TestSkillsCheckerOnPorts$' -v` | **22 fixtures per port, 44/44 passed**; expected failed checks count as passing regressions only when the checker actually fails them |
| Hardened checker, frozen baseline, JIT and Scheme | **57 blocks, 0 failed each** |
| Hardened checker, integrated `--dir skills`, JIT and Scheme | **61 blocks, 0 failed each** |
| Hardened checker, embedded skills (no `--dir`), JIT and Scheme | **61 blocks, 0 failed each** |
| `bifrost-checker skills install --target raw --dir /tmp/bifrost-skills-drive/checker-installed-final --force` | PASS |
| JIT checker with `--dir /tmp/bifrost-skills-drive/checker-installed-final` | **61 blocks, 0 failed** |
| `git diff --check -- skills_check.go skills_test.go skills/README.md` | PASS |

For every checker command above, use `bifrost-checker skills check --impl PORT`, with `--dir` as
specified. The original baseline stays frozen at 57; **61 is the concurrent integrated content
snapshot, not a checker-only increase**. Source manifests in
`/tmp/bifrost-skills-drive/sol-integrated-{before,after}.sha256` were identical across this final
build/test/check/install run (`diff -u` exit 0). Later edits by other workstreams require fresh checks.

## Remaining limits / parent handoff

- Evidence covers these two existing 43.4 binaries, not all adapters, kernels, operating systems or CI.
  Older locally installed ports were identified but not presented as portability successes.
- The checker trusts runtime diagnostics and published arities. A declaration/recorded arity does not
  prove an implementation exists or is correct. Dynamic calls, custom type axioms, and deliberately
  fabricated checker markers/errors are not a security boundary. Keep valid behavioral controls.
- Re-reading negative forms can repeat macro effects; exotic macro-generated private calls without
  published arities may conservatively fail and need investigation. The only current exception is
  the observed kernel grammar helper. No claim of exhaustive Shen dependency analysis is made.
- Standalone HTML comments outside fences are now reserved for checker directives. This intentional
  fail-closed policy rejects previously ignored comments/typos rather than guessing their intent.
- A block can deliberately alter checking/theory within its own file; re-enabling checking at load
  boundaries is not proof that arbitrary hostile source remained typed throughout.
- Go tests normally require no Shen executable; the explicit environment variable is necessary to
  run the actual-port adversarial suite. Packaging comparisons assume files remain stable between
  compilation and execution; the final manifest comparison established that for the reported run.
- No held-out agent trials, book-citation validation, semantic completeness/termination proof, or
  benchmark claim is supplied by this checker workstream. Those remain with the content/parent audit.
